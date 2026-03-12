package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	App            AppConfig            `env-prefix:"APP_"`
	TG             TelegramConfig       `env-prefix:"TG_"`
	Qdrant         QdrantConfig         `env-prefix:"QDRANT_"`
	MongoDB        MongoConfig          `env-prefix:"MONGO_"`
	Logger         LoggerConfig         `env-prefix:"LOGGER_"`
	JWT            JWTConfig            `env-prefix:"JWT_"`
	SuperAdmin     SuperAdminConfig     `env-prefix:"SUPER_ADMIN_"`
	Embedding      EmbeddingConfig      `env-prefix:"EMBEDDING_"`
	LocalEmbedding LocalEmbeddingConfig `env-prefix:"LOCAL_EMBEDDING_"`
}

type AppConfig struct {
	HTTPPort string `env:"HTTP_PORT" env-default:"8080"`
}

type TelegramConfig struct {
	Proxy string `env:"PROXY"`
}

type QdrantConfig struct {
	Host string `env:"HOST" env-required:"true"`
	Port int    `env:"PORT" env-required:"true"`
}

type MongoConfig struct {
	URI    string `env:"URI"`
	DBName string `env:"DB_NAME" env-default:"ai_search"`
}

type LoggerConfig struct {
	Level       string   `env:"LEVEL" env-default:"info"`
	Encoding    string   `env:"ENCODING" env-default:"json"`
	OutputPaths []string `env:"OUTPUT_PATHS" env-default:"stdout"`
}

type JWTConfig struct {
	Secret         string        `env:"SECRET" `
	SessionTimeout time.Duration `env:"SESSION_TIMEOUT" env-default:"15m"`
}

type EmbeddingConfig struct {
	Provider  string `env:"PROVIDER" `
	APIKey    string `env:"API_KEY"`
	ModelName string `env:"MODEL_NAME" `
}

type LocalEmbeddingConfig struct {
	URL   string `env:"URL" `
	Model string `env:"MODEL" `
}

type SuperAdminConfig struct {
	Enabled  bool   `env:"ENABLED" `
	Email    string `env:"EMAIL"`
	Password string `env:"PASSWORD"`
	Name     string `env:"NAME"`
	TenantID string `env:"TENANT_ID"`
}

func Load() *Config {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		log.Fatalf("Config error: %v", err)
	}
	return &cfg
}
