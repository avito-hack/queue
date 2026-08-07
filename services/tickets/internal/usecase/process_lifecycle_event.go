package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidLifecycleEvent = errors.New("invalid lifecycle event")

type LifecycleEventType string

const (
	LifecycleEventPaymentSucceeded    LifecycleEventType = "order.payment_succeeded"
	LifecycleEventReservationReleased LifecycleEventType = "order.reservation_released"
	LifecycleEventListingClosed       LifecycleEventType = "listing.closed"
	LifecycleEventSKUClosed           LifecycleEventType = "sku.closed"
	LifecycleEventSystemCancelled     LifecycleEventType = "ticket.system_cancelled"
)

type LifecycleEvent struct {
	ID         uuid.UUID
	Type       LifecycleEventType
	Source     string
	OccurredAt time.Time
	TicketID   uuid.UUID
	OrderID    uuid.UUID
	ListingID  uuid.UUID
	SKUID      uuid.UUID
	Payload    []byte
}

type TicketLifecycleRepository interface {
	ProcessLifecycleEvent(context.Context, LifecycleEvent) error
}

type ProcessLifecycleEvent struct {
	repository TicketLifecycleRepository
}

func NewProcessLifecycleEvent(repository TicketLifecycleRepository) *ProcessLifecycleEvent {
	return &ProcessLifecycleEvent{repository: repository}
}

func (u *ProcessLifecycleEvent) Process(ctx context.Context, event LifecycleEvent) error {
	if err := validateLifecycleEvent(event); err != nil {
		return err
	}
	if err := u.repository.ProcessLifecycleEvent(ctx, event); err != nil {
		return fmt.Errorf("process lifecycle event: %w", err)
	}

	return nil
}

func validateLifecycleEvent(event LifecycleEvent) error {
	if event.ID == uuid.Nil {
		return fmt.Errorf("%w: empty event id", ErrInvalidLifecycleEvent)
	}
	if strings.TrimSpace(event.Source) == "" {
		return fmt.Errorf("%w: empty source", ErrInvalidLifecycleEvent)
	}
	if event.OccurredAt.IsZero() {
		return fmt.Errorf("%w: empty occurred at", ErrInvalidLifecycleEvent)
	}

	switch event.Type {
	case LifecycleEventPaymentSucceeded, LifecycleEventReservationReleased:
		if event.OrderID == uuid.Nil {
			return fmt.Errorf("%w: empty order id", ErrInvalidLifecycleEvent)
		}
	case LifecycleEventListingClosed:
		if event.ListingID == uuid.Nil {
			return fmt.Errorf("%w: empty listing id", ErrInvalidLifecycleEvent)
		}
	case LifecycleEventSKUClosed:
		if event.ListingID == uuid.Nil || event.SKUID == uuid.Nil {
			return fmt.Errorf("%w: empty listing or SKU id", ErrInvalidLifecycleEvent)
		}
	case LifecycleEventSystemCancelled:
		if event.TicketID == uuid.Nil {
			return fmt.Errorf("%w: empty ticket id", ErrInvalidLifecycleEvent)
		}
	default:
		return fmt.Errorf("%w: unsupported event type %q", ErrInvalidLifecycleEvent, event.Type)
	}

	return nil
}
