package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Server struct {
	Port         int           `env:"SERVER_PORT" envDefault:"8080"`
	ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"10s"`
}

type Database struct {
	URL      string `env:"DATABASE_URL,required"`
	MaxConns int32  `env:"DATABASE_MAX_CONNS" envDefault:"25"`
	MinConns int32  `env:"DATABASE_MIN_CONNS" envDefault:"5"`
}

type Redis struct {
	URL string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`
}

type Crypto struct {
	MasterKey string `env:"ENCRYPTION_KEY,required"`
}

type Webhook struct {
	Timeout    time.Duration `env:"WEBHOOK_TIMEOUT" envDefault:"10s"`
	MaxRetries int           `env:"WEBHOOK_MAX_RETRIES" envDefault:"5"`
}

type Config struct {
	Server   Server
	Database Database
	Redis    Redis
	Crypto   Crypto
	Webhook  Webhook
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

func Load() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}
