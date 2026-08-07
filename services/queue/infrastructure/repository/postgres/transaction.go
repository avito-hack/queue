package postgres

import (
	"context"

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
	pool *pgxpool.Pool
}

func NewTransactionManager(pool *pgxpool.Pool) TransactionManager {
	return &transactionManager{
		pool: pool,
	}
}

func (tm *transactionManager) WithinTransaction(
    ctx context.Context,
    fn func(
        ctx context.Context,
        queueRepository domain.ItemQueueRepository,
        memberRepository domain.ItemQueueMemberRepository,
    ) error,
) error {
    tx, err := tm.pool.Begin(ctx)
    if err != nil {
        return err
    }

    queries := sqlc.New(tx)

    queueRepository := NewItemQueueRepository(queries)
    memberRepository := NewItemQueueMemberRepository(queries)

    err = fn(ctx, queueRepository, memberRepository)
    if err != nil {
        _ = tx.Rollback(ctx)
        return err
    }

    return tx.Commit(ctx)
}
