package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"MRG/internal/config"

	"github.com/gotd/td/telegram/auth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type Manager struct {
	cfg     *config.Config
	log     *zap.Logger
	db      *mongo.Database
	handler MessageHandler

	clients   map[string]*Client
	clientsMu sync.RWMutex
	runs      map[string]context.CancelFunc
}

func NewManager(cfg *config.Config, log *zap.Logger, db *mongo.Database, handler MessageHandler) *Manager {
	return &Manager{
		cfg:     cfg,
		log:     log,
		db:      db,
		handler: handler,
		clients: make(map[string]*Client),
		runs:    make(map[string]context.CancelFunc),
	}
}

func (m *Manager) StartFarm(ctx context.Context) error {
	col := m.db.Collection("messenger_accounts")
	cursor, err := col.Find(ctx, bson.M{"status": StatusActive})
	if err != nil {
		return fmt.Errorf("failed to fetch accounts: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var acc AccountConfig
		if err := cursor.Decode(&acc); err != nil {
			m.log.Error("failed to decode account config", zap.Error(err))
			continue
		}

		if err := m.StartAccount(ctx, acc); err != nil {
			m.log.Error("failed to start account", zap.String("phone", acc.Phone), zap.Error(err))
		}
	}

	return nil
}

func (m *Manager) ListAccounts(ctx context.Context) ([]AccountConfig, error) {
	col := m.db.Collection("messenger_accounts")
	cursor, err := col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var accounts []AccountConfig
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (m *Manager) markAccountUnauthorized(ctx context.Context, id primitive.ObjectID) {
	_, _ = m.db.Collection("messenger_accounts").UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"status":               StatusUnauthorized,
			"waiting_for_password": false,
			"updated_at":           time.Now(),
		},
		"$unset": bson.M{
			"qr_url": "",
		},
	})
}

func (m *Manager) markAccountAuthorized(ctx context.Context, id primitive.ObjectID) {
	_, _ = m.db.Collection("messenger_accounts").UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"status":               StatusAuthorized,
			"waiting_for_password": false,
			"updated_at":           time.Now(),
		},
		"$unset": bson.M{
			"qr_url": "",
		},
	})
}

func (m *Manager) runClientLoop(ctx context.Context, id primitive.ObjectID, phone string, client *Client, delay time.Duration) {
	go func() {
		if delay > 0 {
			m.log.Info("delaying client run after auth", zap.String("phone", phone), zap.Duration("delay", delay))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		if err := client.Run(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				m.log.Info("client stopped by request", zap.String("phone", phone), zap.String("account_id", id.Hex()))
				m.clientsMu.Lock()
				delete(m.clients, phone)
				m.clientsMu.Unlock()
				return
			}
			m.log.Error("client stopped", zap.String("phone", phone), zap.Error(err))
			if strings.Contains(err.Error(), "account not authorized") {
				m.markAccountUnauthorized(context.Background(), id)
			} else {
				m.markAccountAuthorized(context.Background(), id)
			}

			m.clientsMu.Lock()
			delete(m.clients, phone)
			m.clientsMu.Unlock()
		}
	}()
}

func (m *Manager) removeClientIfMatch(phone string, client *Client) {
	if phone == "" || client == nil {
		return
	}

	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	if current, ok := m.clients[phone]; ok && current == client {
		delete(m.clients, phone)
	}
	delete(m.runs, phone)
}

func (m *Manager) stopRuntimeByPhone(phone string) {
	if phone == "" {
		return
	}

	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	if cancel, ok := m.runs[phone]; ok {
		cancel()
		delete(m.runs, phone)
	}
	delete(m.clients, phone)
}

func (m *Manager) stopRuntimeByAccountID(id primitive.ObjectID) {
	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	for phone, client := range m.clients {
		if client == nil || client.acc.ID != id {
			continue
		}
		if cancel, ok := m.runs[phone]; ok {
			cancel()
			delete(m.runs, phone)
		}
		delete(m.clients, phone)
	}
}

func (m *Manager) bindClientCallbacks(id primitive.ObjectID, acc AccountConfig, client *Client) {
	client.onPasswordNeeded = func() {
		m.log.Info("updating DB: waiting for 2FA password", zap.String("phone", acc.Phone))
		_, _ = m.db.Collection("messenger_accounts").UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{
			"$set": bson.M{
				"waiting_for_password": true,
				"updated_at":           time.Now(),
			},
		})
	}
	client.onAuthorized = func(identity AccountIdentity) {
		updatePhone := identity.Phone
		oldPhone := acc.Phone
		if updatePhone == "" {
			updatePhone = oldPhone
		}

		if updatePhone != "" && oldPhone != "" && updatePhone != oldPhone {
			m.clientsMu.Lock()
			if current, ok := m.clients[oldPhone]; ok && current == client {
				delete(m.clients, oldPhone)
				m.clients[updatePhone] = client
			}
			if cancel, ok := m.runs[oldPhone]; ok {
				delete(m.runs, oldPhone)
				m.runs[updatePhone] = cancel
			}
			m.clientsMu.Unlock()
		}

		client.acc.Phone = updatePhone
		_, _ = m.db.Collection("messenger_accounts").UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{
			"$set": bson.M{
				"phone":                updatePhone,
				"name":                 identity.Name,
				"username":             identity.Username,
				"status":               StatusStarting,
				"waiting_for_password": false,
				"updated_at":           time.Now(),
			},
			"$unset": bson.M{"qr_url": ""},
		})
	}
	client.onActivated = func(identity AccountIdentity) {
		phone := identity.Phone
		if phone == "" {
			phone = client.acc.Phone
		}
		if oldPhone := client.acc.Phone; phone != "" && oldPhone != "" && phone != oldPhone {
			m.clientsMu.Lock()
			if current, ok := m.clients[oldPhone]; ok && current == client {
				delete(m.clients, oldPhone)
				m.clients[phone] = client
			}
			if cancel, ok := m.runs[oldPhone]; ok {
				delete(m.runs, oldPhone)
				m.runs[phone] = cancel
			}
			m.clientsMu.Unlock()
		}
		client.acc.Phone = phone
		_, _ = m.db.Collection("messenger_accounts").UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{
			"$set": bson.M{
				"phone":                phone,
				"name":                 identity.Name,
				"username":             identity.Username,
				"status":               StatusActive,
				"waiting_for_password": false,
				"updated_at":           time.Now(),
			},
			"$unset": bson.M{"qr_url": ""},
		})
	}
}

func (m *Manager) AddAccount(ctx context.Context, phone, proxy, proxyFallback string) (*AccountConfig, error) {
	col := m.db.Collection("messenger_accounts")

	if phone != "" {
		var existing AccountConfig
		err := col.FindOne(ctx, bson.M{"phone": phone}).Decode(&existing)
		if err == nil {
			return &existing, nil
		}
	} else {
		phone = fmt.Sprintf("pending_qr_%d", time.Now().Unix())
	}

	acc := AccountConfig{
		ID:            primitive.NewObjectID(),
		Phone:         phone,
		Proxy:         proxy,
		ProxyFallback: proxyFallback,
		Status:        StatusUnauthorized,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err := col.InsertOne(ctx, acc)
	if err != nil {
		return nil, err
	}

	return &acc, nil
}

func (m *Manager) Authenticate(ctx context.Context, id primitive.ObjectID, useQR bool) error {
	var acc AccountConfig
	err := m.db.Collection("messenger_accounts").FindOne(ctx, bson.M{"_id": id}).Decode(&acc)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	m.log.Info("starting authentication for account", zap.String("phone", acc.Phone), zap.Bool("useQR", useQR))
	m.stopRuntimeByAccountID(id)

	m.clientsMu.Lock()
	client, exists := m.clients[acc.Phone]
	if !exists {
		m.log.Debug("creating new client for auth", zap.String("phone", acc.Phone))
		client, err = NewClient(m.cfg, acc, m.log, m.db, m.handler)
		if err != nil {
			m.clientsMu.Unlock()
			m.log.Error("failed to create client for auth", zap.Error(err))
			return err
		}
		m.clients[acc.Phone] = client
	} else {
		m.log.Debug("using existing client for auth", zap.String("phone", acc.Phone))
	}
	m.clientsMu.Unlock()

	m.bindClientCallbacks(id, acc, client)

	_, _ = m.db.Collection("messenger_accounts").UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"status":               StatusAuthPending,
			"qr_url":               "",
			"waiting_for_password": false,
		},
	})

	bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	go func() {
		defer cancel()
		defer func() {
			m.removeClientIfMatch(acc.Phone, client)
			m.removeClientIfMatch(client.acc.Phone, client)
		}()
		m.log.Info("auth goroutine started", zap.String("phone", acc.Phone))

		if useQR {
			go func() {
				qrChan := client.GetQRChan()
				for {
					select {
					case qr := <-qrChan:
						m.log.Debug("updating qr url in db", zap.String("phone", acc.Phone))
						_, _ = m.db.Collection("messenger_accounts").UpdateOne(bgCtx, bson.M{"_id": id}, bson.M{"$set": bson.M{"qr_url": qr}})
					case <-bgCtx.Done():
						return
					}
				}
			}()
		}

		m.log.Info("calling client.Authenticate", zap.String("phone", acc.Phone))
		err := client.Authenticate(bgCtx, useQR)
		if err != nil {
			if errors.Is(err, auth.ErrPasswordAuthNeeded) || strings.Contains(err.Error(), "SESSION_PASSWORD_NEEDED") {
				m.log.Info("waiting for 2FA password via returned auth error", zap.String("phone", acc.Phone))

				err = client.Authenticate(bgCtx, useQR)
			}
		}

		if err != nil {
			m.log.Error("auth failed in goroutine", zap.String("phone", acc.Phone), zap.Error(err))
			m.markAccountUnauthorized(bgCtx, id)
			return
		}

		m.log.Info("auth successful, waiting for manual parser start", zap.String("phone", acc.Phone))
		_, _ = m.db.Collection("messenger_accounts").UpdateOne(bgCtx, bson.M{"_id": id}, bson.M{
			"$set": bson.M{
				"status":               StatusAuthorized,
				"waiting_for_password": false,
				"updated_at":           time.Now(),
			},
			"$unset": bson.M{"qr_url": ""},
		})
	}()

	return nil
}

func (m *Manager) ProvideCode(id primitive.ObjectID, code string) {
	var acc AccountConfig
	err := m.db.Collection("messenger_accounts").FindOne(context.Background(), bson.M{"_id": id}).Decode(&acc)
	if err != nil {
		return
	}

	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()
	if c, ok := m.clients[acc.Phone]; ok {
		c.ProvideCode(code)
	}
}

func (m *Manager) ProvidePassword(id primitive.ObjectID, pass string) {
	var acc AccountConfig
	err := m.db.Collection("messenger_accounts").FindOne(context.Background(), bson.M{"_id": id}).Decode(&acc)
	if err != nil {
		return
	}

	_, _ = m.db.Collection("messenger_accounts").UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"waiting_for_password": false,
			"updated_at":           time.Now(),
		},
	})

	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()
	if c, ok := m.clients[acc.Phone]; ok {
		c.ProvidePassword(pass)
	}
}

func (m *Manager) StartAccount(ctx context.Context, acc AccountConfig) error {
	runCtx, cancel := context.WithCancel(context.Background())

	m.clientsMu.Lock()
	if existingCancel, ok := m.runs[acc.Phone]; ok {
		existingCancel()
		delete(m.runs, acc.Phone)
	}
	client, exists := m.clients[acc.Phone]
	if !exists {
		var err error
		client, err = NewClient(m.cfg, acc, m.log, m.db, m.handler)
		if err != nil {
			m.clientsMu.Unlock()
			cancel()
			return err
		}
		m.clients[acc.Phone] = client
	}
	m.runs[acc.Phone] = cancel
	m.clientsMu.Unlock()

	m.bindClientCallbacks(acc.ID, acc, client)

	_, _ = m.db.Collection("messenger_accounts").UpdateOne(ctx, bson.M{"_id": acc.ID}, bson.M{
		"$set": bson.M{
			"status":     StatusStarting,
			"updated_at": time.Now(),
		},
	})

	m.runClientLoop(runCtx, acc.ID, acc.Phone, client, 0)

	return nil
}

func (m *Manager) StartAccountByID(ctx context.Context, id primitive.ObjectID) error {
	var acc AccountConfig
	if err := m.db.Collection("messenger_accounts").FindOne(ctx, bson.M{"_id": id}).Decode(&acc); err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	if acc.Status == StatusActive || acc.Status == StatusAuthPending {
		return nil
	}

	m.clientsMu.RLock()
	_, exists := m.clients[acc.Phone]
	m.clientsMu.RUnlock()
	if acc.Status == StatusStarting && exists {
		return nil
	}

	return m.StartAccount(context.Background(), acc)
}

func (m *Manager) StopAccount(id primitive.ObjectID) {
	var acc AccountConfig
	err := m.db.Collection("messenger_accounts").FindOne(context.Background(), bson.M{"_id": id}).Decode(&acc)
	if err != nil {
		m.log.Warn("stop account skipped: account not found", zap.String("account_id", id.Hex()), zap.Error(err))
		return
	}

	m.log.Info("stopping parser account", zap.String("account_id", id.Hex()), zap.String("phone", acc.Phone), zap.String("status", string(acc.Status)))
	m.stopRuntimeByAccountID(id)

	nextStatus := StatusUnauthorized
	if len(acc.SessionData) > 0 {
		nextStatus = StatusAuthorized
	}
	_, _ = m.db.Collection("messenger_accounts").UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"status":               nextStatus,
			"waiting_for_password": false,
			"updated_at":           time.Now(),
		},
		"$unset": bson.M{"qr_url": ""},
	})
	m.log.Info("parser account stopped", zap.String("account_id", id.Hex()), zap.String("phone", acc.Phone), zap.String("next_status", string(nextStatus)))
}

func (m *Manager) StopAccountByID(ctx context.Context, id primitive.ObjectID) error {
	var acc AccountConfig
	if err := m.db.Collection("messenger_accounts").FindOne(ctx, bson.M{"_id": id}).Decode(&acc); err != nil {
		return fmt.Errorf("account not found: %w", err)
	}
	m.StopAccount(id)
	return nil
}

func (m *Manager) RestartAccountByID(ctx context.Context, id primitive.ObjectID) error {
	var acc AccountConfig
	if err := m.db.Collection("messenger_accounts").FindOne(ctx, bson.M{"_id": id}).Decode(&acc); err != nil {
		return fmt.Errorf("account not found: %w", err)
	}
	if len(acc.SessionData) == 0 {
		return fmt.Errorf("account has no saved session, re-authentication required")
	}

	m.StopAccount(id)

	if err := m.db.Collection("messenger_accounts").FindOne(ctx, bson.M{"_id": id}).Decode(&acc); err != nil {
		return fmt.Errorf("account not found after stop: %w", err)
	}

	return m.StartAccount(context.Background(), acc)
}

func (m *Manager) DeleteAccount(ctx context.Context, id primitive.ObjectID) error {
	var acc AccountConfig
	err := m.db.Collection("messenger_accounts").FindOne(ctx, bson.M{"_id": id}).Decode(&acc)
	if err == nil {
		m.clientsMu.Lock()
		delete(m.clients, acc.Phone)
		m.clientsMu.Unlock()
	}

	_, err = m.db.Collection("messenger_accounts").DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (m *Manager) GetStatus() map[string]string {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	status := make(map[string]string)
	for phone := range m.clients {
		status[phone] = "online"
	}
	return status
}
