package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/avito-hack/queue/services/queue/internal/domain"
	"github.com/avito-hack/queue/services/queue/postgresql/sqlc"
)

type ItemQueueRepository struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewItemQueueRepository(queries *sqlc.Queries, logger *slog.Logger) *ItemQueueRepository {
	return &ItemQueueRepository{
		queries: queries,
		logger:  logger,
	}
}

func (r *ItemQueueRepository) Create(ctx context.Context, queue *domain.ItemQueue) error {
	err := r.queries.CreateItemQueue(ctx, sqlc.CreateItemQueueParams{
		ItemID:    uuidToPg(queue.ItemID),
		State:     string(queue.State),
		CreatedAt: timeToPg(queue.CreatedAt),
		UpdatedAt: timeToPg(queue.UpdatedAt),
	})

	if err != nil {
		r.logger.Error(
			"failed to create item queue",
			"error",
			err,
			"item_id",
			queue.ItemID,
		)
	}

	return err
}

func (r *ItemQueueRepository) GetByItemID(ctx context.Context, itemID uuid.UUID) (*domain.ItemQueue, error) {
	row, err := r.queries.GetItemQueueByID(ctx, uuidToPg(itemID))
	if err != nil {
		r.logger.Error(
			"failed to get item queue",
			"error",
			err,
			"item_id",
			itemID,
		)

		return nil, err
	}

	return &domain.ItemQueue{
		ItemID:    pgToUUID(row.ItemID),
		State:     domain.ItemQueueState(row.State),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (r *ItemQueueRepository) Update(ctx context.Context, queue *domain.ItemQueue) error {
	err := r.queries.UpdateItemQueue(ctx, sqlc.UpdateItemQueueParams{
		ItemID:    uuidToPg(queue.ItemID),
		State:     string(queue.State),
		UpdatedAt: timeToPg(queue.UpdatedAt),
	})

	if err != nil {
		r.logger.Error(
			"failed to update item queue",
			"error",
			err,
			"item_id",
			queue.ItemID,
		)
	}

	return err
}

func (r *ItemQueueRepository) Delete(ctx context.Context, itemID uuid.UUID) error {
	err := r.queries.DeleteItemQueue(ctx, uuidToPg(itemID))

	if err != nil {
		r.logger.Error(
			"failed to delete item queue",
			"error",
			err,
			"item_id",
			itemID,
		)
	}

	return err
}

func (r *ItemQueueRepository) Exists(ctx context.Context, itemID uuid.UUID) (bool, error) {
	exists, err := r.queries.ExistsItemQueue(ctx, uuidToPg(itemID))

	if err != nil {
		r.logger.Error(
			"failed to check item queue existence",
			"error",
			err,
			"item_id",
			itemID,
		)
	}

	return exists, err
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