package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_happy(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@localhost:5433/d?sslmode=disable")
	t.Setenv("PORT", "8420")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("JWT_TTL_HOURS", "48")
	t.Setenv("CORS_ORIGINS", "http://a.test,http://b.test")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("UPLOAD_DIR", "/tmp/u")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "postgres://u:p@localhost:5433/d?sslmode=disable", cfg.DBURL)
	assert.Equal(t, "8420", cfg.Port)
	assert.Len(t, cfg.JWTSecret, 64)
	assert.Equal(t, 48, cfg.JWTTTLHours)
	assert.Equal(t, []string{"http://a.test", "http://b.test"}, cfg.CORSOrigins)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "/tmp/u", cfg.UploadDir)
}

func TestLoad_defaults(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@h/d")
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("PORT", "")
	t.Setenv("JWT_TTL_HOURS", "")
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("UPLOAD_DIR", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "8420", cfg.Port)
	assert.Equal(t, 24, cfg.JWTTTLHours)
	assert.Equal(t, []string{}, cfg.CORSOrigins)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "./uploads", cfg.UploadDir)
}

func TestLoad_missing_required(t *testing.T) {
	t.Setenv("DB_URL", "")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_URL")
}
