package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

const claimOutboxEventsQuery = `WITH candidates AS (
    SELECT id
    FROM public.outbox_events
    WHERE (status = 'pending' AND available_at <= $1)
       OR (status = 'processing' AND available_at <= $1)
    ORDER BY available_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT $2
)
UPDATE public.outbox_events AS event
SET status = 'processing', attempts = attempts + 1, available_at = $3
FROM candidates
WHERE event.id = candidates.id
RETURNING event.id, event.event_type, event.payload, event.attempts`

const publishOutboxEventQuery = `UPDATE public.outbox_events
SET status = 'published', published_at = $2
WHERE id = $1 AND status = 'processing'`

const retryOutboxEventQuery = `UPDATE public.outbox_events
SET status = 'pending', available_at = $2
WHERE id = $1 AND status = 'processing'`

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
	rows, err := r.pool.Query(ctx, claimOutboxEventsQuery, now, limit, now.Add(lease))
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]usecase.OutboxEvent, 0, limit)
	for rows.Next() {
		var event usecase.OutboxEvent
		if err := rows.Scan(&event.ID, &event.Type, &event.Payload, &event.Attempts); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read outbox events: %w", err)
	}

	return events, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID uuid.UUID, publishedAt time.Time) error {
	commandTag, err := r.pool.Exec(ctx, publishOutboxEventQuery, toPGUUID(eventID), publishedAt)
	if err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("mark outbox event published: unexpected affected rows %d", commandTag.RowsAffected())
	}

	return nil
}

func (r *OutboxRepository) Retry(ctx context.Context, eventID uuid.UUID, availableAt time.Time) error {
	commandTag, err := r.pool.Exec(ctx, retryOutboxEventQuery, toPGUUID(eventID), availableAt)
	if err != nil {
		return fmt.Errorf("retry outbox event: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("retry outbox event: unexpected affected rows %d", commandTag.RowsAffected())
	}

	return nil
}
