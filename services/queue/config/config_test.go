package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Load_ReturnDefaults(t *testing.T) {
	// given
	t.Setenv("HTTP_PORT", "")
	t.Setenv("POSTGRES_AUTO_MIGRATE", "")
	clearEnv(
		"HTTP_HOST",
		"HTTP_PORT",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"QUEUE_POSTGRES_DB",
		"QUEUE_POSTGRES_USER",
		"QUEUE_POSTGRES_PASSWORD",
		"POSTGRES_MAX_CONNECTIONS",
		"POSTGRES_AUTO_MIGRATE",
		"POSTGRES_MIGRATIONS_DIR",
		"JWT_SECRET",
	)

	// when
	cfg, err := Load()

	// then
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0", cfg.HTTP.Host)
	assert.Equal(t, 8080, cfg.HTTP.Port)
	assert.Equal(t, 5432, cfg.Postgres.Port)
	assert.Equal(t, "queue", cfg.Postgres.DB)
	assert.Equal(t, "queue", cfg.Postgres.User)
	assert.Equal(t, int32(10), cfg.Postgres.MaxConnections)
	assert.False(t, cfg.Postgres.AutoMigrate)
	assert.Equal(t, "postgresql/migrations", cfg.Postgres.MigrationsDir)
}

func Test_Load_ReturnConfiguredAutoMigrateAndTimeouts(t *testing.T) {
	// given
	t.Setenv("POSTGRES_AUTO_MIGRATE", "true")
	t.Setenv("POSTGRES_MIGRATIONS_DIR", "custom/migrations")
	t.Setenv("HTTP_READ_TIMEOUT", "3s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "4s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "5s")

	// when
	cfg, err := Load()

	// then
	require.NoError(t, err)
	assert.True(t, cfg.Postgres.AutoMigrate)
	assert.Equal(t, "custom/migrations", cfg.Postgres.MigrationsDir)
	assert.Equal(t, 3*time.Second, cfg.HTTP.ReadTimeout)
	assert.Equal(t, 4*time.Second, cfg.HTTP.WriteTimeout)
	assert.Equal(t, 5*time.Second, cfg.HTTP.ShutdownTimeout)
}

func Test_Load_ReturnErrorForInvalidAutoMigrate(t *testing.T) {
	// given
	t.Setenv("POSTGRES_AUTO_MIGRATE", "not-a-bool")

	// when
	_, err := Load()

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "POSTGRES_AUTO_MIGRATE")
}

func clearEnv(names ...string) {
	for _, name := range names {
		_ = os.Unsetenv(name)
	}
}
