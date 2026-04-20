package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBURL       string
	Port        string
	JWTSecret   string
	JWTTTLHours int
	CORSOrigins []string
	LogLevel    string
	UploadDir   string
}

func Load() (*Config, error) {
	cfg := &Config{
		DBURL:       os.Getenv("DB_URL"),
		Port:        envOr("PORT", "8420"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTTTLHours: envIntOr("JWT_TTL_HOURS", 24),
		CORSOrigins: splitCSV(os.Getenv("CORS_ORIGINS")),
		LogLevel:    envOr("LOG_LEVEL", "info"),
		UploadDir:   envOr("UPLOAD_DIR", "./uploads"),
	}

	if cfg.DBURL == "" {
		return nil, errors.New("DB_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func splitCSV(s string) []string {
	out := []string{}
	if s == "" {
		return out
	}
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
