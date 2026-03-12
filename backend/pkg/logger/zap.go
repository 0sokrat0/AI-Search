package logger

import (
	"strings"

	"MRG/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(cfg *config.Config) (*zap.Logger, error) {
	var zapConfig zap.Config

	switch strings.ToLower(cfg.Logger.Level) {
	case "debug":
		zapConfig = zap.NewDevelopmentConfig()
	case "info":
		zapConfig = zap.NewDevelopmentConfig()
	case "warn":
		zapConfig = zap.NewProductionConfig()
	case "error":
		zapConfig = zap.NewProductionConfig()
	case "dpanic":
		zapConfig = zap.NewProductionConfig()
	case "panic":
		zapConfig = zap.NewProductionConfig()
	case "fatal":
		zapConfig = zap.NewProductionConfig()
	default:
		zapConfig = zap.NewDevelopmentConfig()
	}
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Logger.Level)); err == nil {
		zapConfig.Level.SetLevel(level)
	}

	zapConfig.Encoding = cfg.Logger.Encoding

	zapConfig.OutputPaths = cfg.Logger.OutputPaths
	zapConfig.ErrorOutputPaths = cfg.Logger.OutputPaths

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	return logger, nil
}
