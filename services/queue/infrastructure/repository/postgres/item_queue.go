package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/avito-hack/queue/services/queue/internal/domain"
	"github.com/avito-hack/queue/services/queue/postgresql/sqlc"
)

type ItemQueueRepository struct {
    queries *sqlc.Queries
}

func NewItemQueueRepository(queries *sqlc.Queries) *ItemQueueRepository {
    return &ItemQueueRepository{
        queries: queries,
    }
}

func (r *ItemQueueRepository) Create(
    ctx context.Context,
    queue *domain.ItemQueue,
) error {
    return r.queries.CreateItemQueue(ctx, sqlc.CreateItemQueueParams{
        ItemID:    uuidToPg(queue.ItemID),
        State:     string(queue.State),
        CreatedAt: timeToPg(queue.CreatedAt),
        UpdatedAt: timeToPg(queue.UpdatedAt),
    })
}

func (r *ItemQueueRepository) GetByItemID(
    ctx context.Context,
    itemID uuid.UUID,
) (*domain.ItemQueue, error) {
    row, err := r.queries.GetItemQueueByID(ctx, uuidToPg(itemID))
    if err != nil {
        return nil, err
    }

    return &domain.ItemQueue{
        ItemID:    pgToUUID(row.ItemID),
        State:     domain.ItemQueueState(row.State),
        CreatedAt: row.CreatedAt.Time,
        UpdatedAt: row.UpdatedAt.Time,
    }, nil
}

func (r *ItemQueueRepository) Update(
    ctx context.Context,
    queue *domain.ItemQueue,
) error {
    return r.queries.UpdateItemQueue(ctx, sqlc.UpdateItemQueueParams{
        ItemID:    uuidToPg(queue.ItemID),
        State:     string(queue.State),
        UpdatedAt: timeToPg(queue.UpdatedAt),
    })
}

func (r *ItemQueueRepository) Delete(
    ctx context.Context,
    itemID uuid.UUID,
) error {
    return r.queries.DeleteItemQueue(ctx, uuidToPg(itemID))
}

func (r *ItemQueueRepository) Exists(
    ctx context.Context,
    itemID uuid.UUID,
) (bool, error) {
    return r.queries.ExistsItemQueue(ctx, uuidToPg(itemID))
}

func uuidToPg(id uuid.UUID) pgtype.UUID {
    return pgtype.UUID{
        Bytes: [16]byte(id),
        Valid: true,
    }
}

func pgToUUID(id pgtype.UUID) uuid.UUID {
    return uuid.UUID(id.Bytes)
}

func timeToPg(t time.Time) pgtype.Timestamp {
    return pgtype.Timestamp{
        Time:  t,
        Valid: true,
    }
}
