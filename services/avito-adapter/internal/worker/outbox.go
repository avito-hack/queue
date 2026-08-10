package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/avito-hack/queue/services/avito-adapter/infrastructure/messaging/rabbitmq"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Outbox struct {
	database  *pgxpool.Pool
	publisher *rabbitmq.Publisher
}

func NewOutbox(database *pgxpool.Pool, publisher *rabbitmq.Publisher) *Outbox {
	return &Outbox{database: database, publisher: publisher}
}
func (w *Outbox) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := w.runOnce(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "publish outbox events", "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (w *Outbox) runOnce(ctx context.Context) error {
	rows, err := w.database.Query(ctx, "WITH claimed AS (SELECT id FROM public.outbox_events WHERE published_at IS NULL AND available_at <= now() AND (lease_until IS NULL OR lease_until < now()) ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 20) UPDATE public.outbox_events o SET claimed_at = now(), lease_until = now() + interval '30 seconds', attempts = attempts + 1 FROM claimed WHERE o.id = claimed.id RETURNING o.id, o.event_type, o.payload")
	if err != nil {
		return fmt.Errorf("claim outbox: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, eventType string
		var payload []byte
		if err := rows.Scan(&id, &eventType, &payload); err != nil {
			return err
		}
		if err := w.publisher.Publish(ctx, id, eventType, payload); err != nil {
			_, retryErr := w.database.Exec(ctx, "UPDATE public.outbox_events SET available_at = now() + interval '5 seconds', lease_until = NULL WHERE id = $1", id)
			if retryErr != nil {
				return fmt.Errorf("retry outbox: %w", retryErr)
			}
			return err
		}
		if _, err := w.database.Exec(ctx, "UPDATE public.outbox_events SET published_at = now(), lease_until = NULL WHERE id = $1", id); err != nil {
			return fmt.Errorf("mark published: %w", err)
		}
	}
	return rows.Err()
}
