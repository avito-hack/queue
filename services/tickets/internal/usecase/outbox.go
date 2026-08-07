package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID       uuid.UUID
	Type     string
	Payload  json.RawMessage
	Attempts int
}

type OutboxRepository interface {
	Claim(context.Context, time.Time, int, time.Duration) ([]OutboxEvent, error)
	MarkPublished(context.Context, uuid.UUID, time.Time) error
	Retry(context.Context, uuid.UUID, time.Time) error
}

type EventPublisher interface {
	Publish(context.Context, OutboxEvent) error
}
