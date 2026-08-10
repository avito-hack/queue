package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlgen "github.com/avito-hack/queue/services/tickets/gen/sql"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) Claim(
	ctx context.Context,
	now time.Time,
	limit int,
	lease time.Duration,
) ([]usecase.OutboxEvent, error) {
	rows, err := sqlgen.New(r.pool).ClaimOutboxEvents(ctx, sqlgen.ClaimOutboxEventsParams{
		LeaseDeadline: toPGTimestamptz(now.Add(lease)),
		ClaimedAt:     toPGTimestamptz(now),
		BatchSize:     int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}

	events := make([]usecase.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		if !row.ID.Valid {
			return nil, fmt.Errorf("scan outbox event: id is null")
		}
		events = append(events, usecase.OutboxEvent{
			ID:       uuid.UUID(row.ID.Bytes),
			Type:     row.EventType,
			Payload:  row.Payload,
			Attempts: int(row.Attempts),
		})
	}

	return events, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID uuid.UUID, publishedAt time.Time) error {
	rowsAffected, err := sqlgen.New(r.pool).PublishOutboxEvent(ctx, sqlgen.PublishOutboxEventParams{
		PublishedAt: toPGTimestamptz(publishedAt),
		ID:          toPGUUID(eventID),
	})
	if err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("mark outbox event published: unexpected affected rows %d", rowsAffected)
	}

	return nil
}

func (r *OutboxRepository) Retry(ctx context.Context, eventID uuid.UUID, availableAt time.Time) error {
	rowsAffected, err := sqlgen.New(r.pool).RetryOutboxEvent(ctx, sqlgen.RetryOutboxEventParams{
		AvailableAt: toPGTimestamptz(availableAt),
		ID:          toPGUUID(eventID),
	})
	if err != nil {
		return fmt.Errorf("retry outbox event: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("retry outbox event: unexpected affected rows %d", rowsAffected)
	}

	return nil
}
