package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avito-hack/queue/services/queue/internal/domain"
	"github.com/avito-hack/queue/services/queue/postgresql/sqlc"
)

type TransactionManager interface {
	WithinTransaction(
		ctx context.Context,
		fn func(
			ctx context.Context,
			queueRepository domain.ItemQueueRepository,
			memberRepository domain.ItemQueueMemberRepository,
		) error,
	) error
}

type transactionManager struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewTransactionManager(pool *pgxpool.Pool, logger *slog.Logger) TransactionManager {
	return &transactionManager{
		pool:   pool,
		logger: logger,
	}
}

func (tm *transactionManager) WithinTransaction(
	ctx context.Context,
	fn func(
		ctx context.Context,
		queueRepository domain.ItemQueueRepository,
		memberRepository domain.ItemQueueMemberRepository,
	) error,
) (err error) {
	tm.logger.Debug("starting database transaction")

	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		tm.logger.Error(
			"failed to start database transaction",
			"error",
			err,
		)

		return err
	}

	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	defer func() {
		if p := recover(); p != nil {
			if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil {
				tm.logger.Error(
					"transaction rollback after panic failed",
					"error",
					rollbackErr,
				)
			}

			panic(p)
		}
	}()

	queries := sqlc.New(tx)

	queueRepository := NewItemQueueRepository(queries, tm.logger)
	memberRepository := NewItemQueueMemberRepository(queries, tm.logger)

	err = fn(ctx, queueRepository, memberRepository)

	if err != nil {
		tm.logger.Error(
			"transaction failed",
			"error",
			err,
		)

		if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil {
			tm.logger.Error(
				"transaction rollback failed",
				"error",
				rollbackErr,
			)
		}

		return err
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		tm.logger.Error(
			"transaction commit failed",
			"error",
			commitErr,
		)

		return commitErr
	}

	tm.logger.Debug("database transaction committed")

	return nil
}