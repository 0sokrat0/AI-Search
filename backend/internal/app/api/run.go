package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"MRG/internal/config"
	httpserver "MRG/internal/delivery/http"
	v1 "MRG/internal/delivery/http/handlers/v1"
	"MRG/internal/domain/user"
	"MRG/internal/infrastructure/embeddings"
	"MRG/internal/infrastructure/search"
	contacts_repo "MRG/internal/infrastructure/storage/contacts"
	leads_repo "MRG/internal/infrastructure/storage/leads"
	messages_repo "MRG/internal/infrastructure/storage/messages"
	settings_store "MRG/internal/infrastructure/storage/settings"
	users_repo "MRG/internal/infrastructure/storage/users"
	telegram_infra "MRG/internal/infrastructure/telegram"
	"MRG/internal/usecase/auth"
	knowledge_usecase "MRG/internal/usecase/knowledge"
	lead_usecase "MRG/internal/usecase/lead"
	settings_usecase "MRG/internal/usecase/settings"
	signal_usecase "MRG/internal/usecase/signal"
	user_usecase "MRG/internal/usecase/user"
	mongo_conn "MRG/pkg/mongo"
	qdrant_conn "MRG/pkg/qdrant"

	qdrant_client "github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
)

func Run(ctx context.Context, cfg *config.Config, log *zap.Logger) error {
	mongoClient, err := mongo_conn.NewMongoDB(cfg.MongoDB.URI)
	if err != nil {
		return fmt.Errorf("connect mongodb: %w", err)
	}
	defer func() { _ = mongoClient.Disconnect(context.Background()) }()
	log.Info("mongodb connected", zap.String("db", cfg.MongoDB.DBName))

	qdrantClient, err := connectQdrant(ctx, cfg)
	if err != nil {
		return err
	}
	log.Info("qdrant connected", zap.String("host", cfg.Qdrant.Host), zap.Int("port", cfg.Qdrant.Port))

	if strings.EqualFold(cfg.Embedding.Provider, "local") {
		if err := pingOllama(ctx, cfg.LocalEmbedding.URL); err != nil {
			return fmt.Errorf("ollama not ready: %w", err)
		}
		log.Info("ollama ready", zap.String("url", cfg.LocalEmbedding.URL), zap.String("model", cfg.LocalEmbedding.Model))
	}

	db := mongoClient.Database(cfg.MongoDB.DBName)

	userRepo := users_repo.NewMongoRepository(db)
	leadRepo := leads_repo.NewMongoRepository(db)
	messageRepo := messages_repo.NewMongoRepository(db)
	contactRepo := contacts_repo.NewMongoRepository(db)
	settingsStore := settings_store.New(db)

	if err := messageRepo.EnsureIndexes(ctx); err != nil {
		log.Warn("message repo indexes check failed", zap.Error(err))
	}

	if err := ensureSuperAdmin(ctx, cfg, userRepo, log); err != nil {
		return fmt.Errorf("ensure super admin: %w", err)
	}

	authUC := auth.NewAuthUseCase(userRepo, cfg)
	userUC := user_usecase.NewUserUseCase(userRepo)
	leadUC := lead_usecase.New(leadRepo, messageRepo, contactRepo)
	settingsUC := settings_usecase.New(settingsStore, messageRepo)

	sieve := newSieve(cfg, settingsStore, qdrantClient)
	leadUC.WithSieve(sieve)
	log.Info("sieve ready", zap.String("provider", cfg.Embedding.Provider))

	ingestHandler := telegram_infra.NewIngestHandler(cfg.SuperAdmin.TenantID, log, messageRepo, userRepo, leadRepo, contactRepo, sieve, settingsStore)
	manager := telegram_infra.NewManager(cfg, log, db, ingestHandler)
	if err := manager.StartFarm(ctx); err != nil {
		log.Warn("telegram farm not started", zap.Error(err))
	} else {
		log.Info("telegram farm started")
	}

	authHandler := v1.NewAuthHandler(authUC, userUC)
	userHandler := v1.NewUserHandler(userUC)
	leadHandler := v1.NewLeadHandler(leadUC)
	signalApp := signal_usecase.NewService(messageRepo, contactRepo, leadRepo, leadUC, sieve, settingsStore)
	signalHandler := v1.NewSignalHandler(signalApp)
	settingsHandler := v1.NewSettingsHandler(settingsUC)
	knowledgeHandler := v1.NewKnowledgeHandler(knowledge_usecase.NewImportService(sieve))
	accountHandler := v1.NewAccountHandler(manager)

	server := httpserver.NewServer(cfg)
	httpserver.SetupRoutes(server.GetApp(), authHandler, userHandler, leadHandler, signalHandler, settingsHandler, knowledgeHandler, accountHandler, userRepo, cfg)

	log.Info("http server listening", zap.String("port", cfg.App.HTTPPort))

	errCh := make(chan error, 1)
	go func() { errCh <- server.Start(cfg.App.HTTPPort) }()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		log.Info("server stopped")
		return nil
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	}
}

func connectQdrant(ctx context.Context, cfg *config.Config) (*qdrant_client.Client, error) {
	client, err := qdrant_conn.Connect(cfg)
	if err != nil {
		return nil, fmt.Errorf("connect qdrant: %w", err)
	}
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := client.HealthCheck(healthCtx); err != nil {
		return nil, fmt.Errorf("ping qdrant: %w", err)
	}
	return client, nil
}

func pingOllama(ctx context.Context, baseURL string) error {
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/api/tags"
	if endpoint == "/api/tags" {
		endpoint = "http://127.0.0.1:11434/api/tags"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func newSieve(cfg *config.Config, settingsStore *settings_store.Store, qdrantClient *qdrant_client.Client) *search.QdrantSieve {
	get := func(key string, def float64) float32 {
		return float32(settingsStore.GetFloat(context.Background(), key, def))
	}
	return search.NewQdrantSieve(newEmbedder(cfg), qdrantClient, func() float32 {
		return get("lead_threshold", 0.70)
	}, func() time.Duration {
		return time.Duration(get("sender_window_seconds", 60)) * time.Second
	}).WithCategoryThresholds(func(category string) float32 {
		switch category {
		case "trader_search":
			return get("trader_threshold", 0.60)
		case "traders":
			return get("trader_threshold", 0.60)
		case "merchants":
			return get("merchant_threshold", 0.60)
		case "ps_offers":
			return get("ps_offer_threshold", 0.60)
		default:
			return get("lead_threshold", 0.70)
		}
	})
}

func newEmbedder(cfg *config.Config) embeddings.Embedder {
	if strings.EqualFold(cfg.Embedding.Provider, "local") {
		return embeddings.NewLocalEmbedder(cfg.LocalEmbedding.URL, cfg.LocalEmbedding.Model)
	}
	return embeddings.NewGeminiEmbedder(cfg.Embedding.APIKey, cfg.Embedding.ModelName)
}

func ensureSuperAdmin(ctx context.Context, cfg *config.Config, repo user.Repository, log *zap.Logger) error {
	if !cfg.SuperAdmin.Enabled {
		return nil
	}
	if cfg.SuperAdmin.Email == "" || cfg.SuperAdmin.Password == "" {
		return errors.New("SUPER_ADMIN_EMAIL and SUPER_ADMIN_PASSWORD required")
	}
	if _, err := repo.FindByEmail(ctx, cfg.SuperAdmin.Email); err == nil {
		log.Info("super admin exists", zap.String("email", cfg.SuperAdmin.Email))
		return nil
	} else if !errors.Is(err, user.ErrUserNotFound) {
		return err
	}

	superAdmin := user.New(cfg.SuperAdmin.TenantID, cfg.SuperAdmin.Email, cfg.SuperAdmin.Name, []user.Role{user.RoleSuperAdmin})
	if err := superAdmin.SetPassword(cfg.SuperAdmin.Password); err != nil {
		return err
	}
	if err := repo.Create(ctx, superAdmin); err != nil {
		return err
	}
	log.Info("super admin created", zap.String("email", cfg.SuperAdmin.Email))
	return nil
}
