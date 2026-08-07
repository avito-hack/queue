package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
)

var (
	ErrInvalidDecline      = errors.New("invalid ticket decline")
	ErrTicketNotDeclinable = errors.New("ticket is not declinable")
)

type DeclineTicketCommand struct {
	UserID         uuid.UUID
	TicketID       uuid.UUID
	IdempotencyKey uuid.UUID
	Now            time.Time
}

type DeclineTicketResult struct {
	TicketID uuid.UUID
	Status   domain.TicketStatus
}

type TicketDeclineRepository interface {
	Decline(context.Context, DeclineTicketCommand) (DeclineTicketResult, error)
}

type DeclineTicket struct {
	repository TicketDeclineRepository
	clock      Clock
}

func NewDeclineTicket(repository TicketDeclineRepository, clocks ...Clock) *DeclineTicket {
	clock := Clock(time.Now)
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}

	return &DeclineTicket{
		repository: repository,
		clock:      clock,
	}
}

func (u *DeclineTicket) Decline(
	ctx context.Context,
	userID uuid.UUID,
	ticketID uuid.UUID,
	idempotencyKey uuid.UUID,
) (DeclineTicketResult, error) {
	if userID == uuid.Nil {
		return DeclineTicketResult{}, fmt.Errorf("%w: empty user id", ErrInvalidDecline)
	}
	if ticketID == uuid.Nil {
		return DeclineTicketResult{}, fmt.Errorf("%w: empty ticket id", ErrInvalidDecline)
	}
	if idempotencyKey == uuid.Nil {
		return DeclineTicketResult{}, fmt.Errorf("%w: empty idempotency key", ErrInvalidDecline)
	}

	result, err := u.repository.Decline(ctx, DeclineTicketCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: idempotencyKey,
		Now:            u.clock(),
	})
	if err != nil {
		return DeclineTicketResult{}, fmt.Errorf("decline ticket: %w", err)
	}
	if !validDeclineResult(result, ticketID) {
		return DeclineTicketResult{}, errors.New("decline ticket: invalid decline result")
	}

	return result, nil
}

func validDeclineResult(result DeclineTicketResult, ticketID uuid.UUID) bool {
	return result.TicketID == ticketID && result.Status == domain.TicketStatusClosed
}
