package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	"github.com/avito-hack/queue/services/queue/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func ApplyMigrations(ctx context.Context, cfg config.PostgresConfig) error {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DB,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open postgres connection for migrations: %w", err)
	}

	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set postgres dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, cfg.MigrationsDir); err != nil {
		return fmt.Errorf("apply postgres migrations: %w", err)
	}

	return nil
}
