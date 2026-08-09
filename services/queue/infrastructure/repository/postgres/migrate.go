package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"

	"github.com/avito-hack/queue/services/queue/config"

	pgxstdlib "github.com/jackc/pgx/v5/stdlib"
)

func ApplyMigrations(ctx context.Context, cfg config.PostgresConfig, logger *slog.Logger) error {
	_ = pgxstdlib.GetDefaultDriver()
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DB,
	)

	logger.Info(
		"starting postgres migrations",
		"database",
		cfg.DB,
		"migrations_dir",
		cfg.MigrationsDir,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Error(
			"failed to open postgres connection for migrations",
			"error",
			err,
		)

		return fmt.Errorf("open postgres connection for migrations: %w", err)
	}

	defer func() {
		_ = db.Close()
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		logger.Error(
			"failed to set postgres goose dialect",
			"error",
			err,
		)

		return fmt.Errorf("set postgres dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, cfg.MigrationsDir); err != nil {
		logger.Error(
			"failed to apply postgres migrations",
			"error",
			err,
			"migrations_dir",
			cfg.MigrationsDir,
		)

		return fmt.Errorf("apply postgres migrations: %w", err)
	}

	logger.Info("postgres migrations applied successfully")

	return nil
}
