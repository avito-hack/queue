package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avito-hack/queue/services/queue/config"
)

func NewPool(ctx context.Context, cfg config.PostgresConfig, logger *slog.Logger) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DB,
	)

	logger.Debug(
		"creating postgres connection pool",
		"host",
		cfg.Host,
		"port",
		cfg.Port,
		"database",
		cfg.DB,
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Error(
			"failed to create postgres pool",
			"error",
			err,
		)

		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		logger.Error(
			"failed to ping postgres",
			"error",
			err,
		)

		pool.Close()

		return nil, err
	}

	logger.Info(
		"postgres connection established",
		"host",
		cfg.Host,
		"database",
		cfg.DB,
	)

	return pool, nil
}
