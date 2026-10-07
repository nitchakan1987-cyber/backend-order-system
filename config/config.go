package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

func Load() (Config, error) {
	// วิธี deploy ปัจจุบันกำหนดให้ต้องมีไฟล์ .env
	if err := godotenv.Load(".env"); err != nil {
		return Config{}, fmt.Errorf("cannot load .env: %w", err)
	}

	cfg := Config{
		HTTPAddr:    strings.TrimSpace(os.Getenv("HTTP_ADDR")),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
	}

	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New(
			"DATABASE_URL (MySQL DSN) is empty after loading .env; check file content and existing environment",
		)
	}

	return cfg, nil
}
