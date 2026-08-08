package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
)

var ErrInvalidListingEvent = errors.New("invalid listing event")

type IncomingListingEvent struct {
	ID      uuid.UUID
	Type    string
	Source  string
	Payload []byte
}

type ListingQuantityChangedCommand struct {
	Event            IncomingListingEvent
	ListingID        uuid.UUID
	SellerID         uuid.UUID
	PreviousQuantity int
	Quantity         int
	ChangedAt        time.Time
}

type ListingStatusChangedCommand struct {
	Event          IncomingListingEvent
	ListingID      uuid.UUID
	SellerID       uuid.UUID
	PreviousStatus string
	Status         string
	ChangedAt      time.Time
}

type RevokeListingTicketsCommand struct {
	Event           IncomingListingEvent
	ListingID       uuid.UUID
	RevocationLimit int
	CloseAll        bool
	CloseReason     domain.TicketCloseReason
	HandledAt       time.Time
}

type ListingEventRepository interface {
	RevokeListingTickets(context.Context, RevokeListingTicketsCommand) (int, error)
}

type HandleListingEvents struct {
	repository ListingEventRepository
	clock      Clock
}

func NewHandleListingEvents(repository ListingEventRepository, clocks ...Clock) *HandleListingEvents {
	clock := Clock(time.Now)
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}

	return &HandleListingEvents{repository: repository, clock: clock}
}

func (u *HandleListingEvents) QuantityChanged(
	ctx context.Context,
	command ListingQuantityChangedCommand,
) (int, error) {
	if err := validateIncomingListingEvent(command.Event, domain.ListingEventQuantityChanged); err != nil {
		return 0, err
	}
	if command.ListingID == uuid.Nil {
		return 0, fmt.Errorf("%w: empty listing id", ErrInvalidListingEvent)
	}
	if command.SellerID == uuid.Nil {
		return 0, fmt.Errorf("%w: empty seller id", ErrInvalidListingEvent)
	}
	if command.PreviousQuantity < 0 || command.Quantity < 0 {
		return 0, fmt.Errorf("%w: negative listing quantity", ErrInvalidListingEvent)
	}
	if command.ChangedAt.IsZero() {
		return 0, fmt.Errorf("%w: empty changed time", ErrInvalidListingEvent)
	}

	revocationLimit := 0
	if command.Quantity < command.PreviousQuantity {
		revocationLimit = command.PreviousQuantity - command.Quantity
	}
	revoked, err := u.repository.RevokeListingTickets(ctx, RevokeListingTicketsCommand{
		Event:           command.Event,
		ListingID:       command.ListingID,
		RevocationLimit: revocationLimit,
		CloseReason:     domain.TicketCloseReasonSystemCancelled,
		HandledAt:       u.clock(),
	})
	if err != nil {
		return 0, fmt.Errorf("handle listing quantity changed: %w", err)
	}

	return revoked, nil
}

func (u *HandleListingEvents) StatusChanged(
	ctx context.Context,
	command ListingStatusChangedCommand,
) (int, error) {
	if err := validateIncomingListingEvent(command.Event, domain.ListingEventStatusChanged); err != nil {
		return 0, err
	}
	if command.ListingID == uuid.Nil {
		return 0, fmt.Errorf("%w: empty listing id", ErrInvalidListingEvent)
	}
	if command.SellerID == uuid.Nil {
		return 0, fmt.Errorf("%w: empty seller id", ErrInvalidListingEvent)
	}
	if command.PreviousStatus != "active" && command.PreviousStatus != "paused" {
		return 0, fmt.Errorf("%w: unexpected previous listing status %q", ErrInvalidListingEvent, command.PreviousStatus)
	}
	if command.Status != "paused" && command.Status != "removed" {
		return 0, fmt.Errorf("%w: unexpected listing status %q", ErrInvalidListingEvent, command.Status)
	}
	if command.ChangedAt.IsZero() {
		return 0, fmt.Errorf("%w: empty changed time", ErrInvalidListingEvent)
	}

	revoked, err := u.repository.RevokeListingTickets(ctx, RevokeListingTicketsCommand{
		Event:       command.Event,
		ListingID:   command.ListingID,
		CloseAll:    true,
		CloseReason: domain.TicketCloseReasonListingClosed,
		HandledAt:   u.clock(),
	})
	if err != nil {
		return 0, fmt.Errorf("handle listing status changed: %w", err)
	}

	return revoked, nil
}

func validateIncomingListingEvent(event IncomingListingEvent, expectedType string) error {
	if event.ID == uuid.Nil {
		return fmt.Errorf("%w: empty event id", ErrInvalidListingEvent)
	}
	if event.Type != expectedType {
		return fmt.Errorf("%w: unexpected event type %q", ErrInvalidListingEvent, event.Type)
	}
	if event.Source != "avito-adapter" {
		return fmt.Errorf("%w: unexpected event source %q", ErrInvalidListingEvent, event.Source)
	}
	if len(event.Payload) == 0 {
		return fmt.Errorf("%w: empty event payload", ErrInvalidListingEvent)
	}

	return nil
}
