package config

import (
	"flag"
	"fmt"
	"strings"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	Address     string `env:"RUN_ADDRESS"`
	DatabaseURI string `env:"DATABASE_URI"`
	JWTSecret   string `env:"JWT_SECRET"`
}

func InitConfig(logger *zap.SugaredLogger) (Config, error) {
	var cfg Config

	logger = logger.With("component", "config", "op", "init_config")

	_ = godotenv.Load(".env")

	if err := env.Parse(&cfg); err != nil {
		logger.Warnw("env parse error", "err", err)
		return Config{}, fmt.Errorf("env parse error: %w", err)
	}

	flag.StringVar(&cfg.Address, "a", cfg.Address, "server address")
	flag.StringVar(&cfg.DatabaseURI, "db", cfg.DatabaseURI, "database uri")

	flag.Parse()

	cfg.Address = strings.TrimSpace(cfg.Address)
	cfg.DatabaseURI = strings.TrimSpace(cfg.DatabaseURI)
	cfg.JWTSecret = strings.TrimSpace(cfg.JWTSecret)

	if cfg.JWTSecret == "" {
		logger.Warn("empty JWT secret after config init")
		return Config{}, fmt.Errorf("empty jwt secret")
	}

	if len([]byte(cfg.JWTSecret)) < 32 {
		logger.Warn("JWT secret too short")
		return Config{}, fmt.Errorf("JWT secret too short: need at least 32 bytes")
	}

	return cfg, nil
}
