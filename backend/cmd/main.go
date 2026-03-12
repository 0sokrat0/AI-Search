package main

import (
	"context"
	"os/signal"
	"syscall"

	apiapp "MRG/internal/app/api"
	"MRG/internal/config"
	"MRG/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()

	log, err := logger.New(cfg)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer func() {
		_ = log.Sync()
	}()

	log.Info("Config", zap.Any("config", cfg))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("starting api service", zap.String("http_port", cfg.App.HTTPPort))
	if err := apiapp.Run(ctx, cfg, log); err != nil {
		log.Fatal("api service exited with error", zap.Error(err))
	}

	log.Info("api service stopped")
}
