package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
)

var ErrInvalidTicketFilter = errors.New("invalid ticket filter")

type ListTicketsFilter struct {
	Status    *domain.TicketStatus
	ListingID *uuid.UUID
	SKUID     *uuid.UUID
}

type TicketRepository interface {
	List(context.Context, uuid.UUID, ListTicketsFilter) ([]domain.Ticket, error)
}

type Clock func() time.Time

type ListTickets struct {
	repository TicketRepository
	clock      Clock
}

func NewListTickets(repository TicketRepository, clocks ...Clock) *ListTickets {
	clock := Clock(time.Now)
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}

	return &ListTickets{
		repository: repository,
		clock:      clock,
	}
}

func (u *ListTickets) List(ctx context.Context, userID uuid.UUID, filter ListTicketsFilter) ([]domain.Ticket, error) {
	if userID == uuid.Nil {
		return nil, fmt.Errorf("%w: empty user id", ErrInvalidTicketFilter)
	}
	if filter.Status != nil && !filter.Status.Valid() {
		return nil, fmt.Errorf("%w: unknown status %q", ErrInvalidTicketFilter, *filter.Status)
	}

	tickets, err := u.repository.List(ctx, userID, filter)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}

	if tickets == nil {
		tickets = make([]domain.Ticket, 0)
	}

	now := u.clock()
	for index := range tickets {
		tickets[index].AvailableActions = tickets[index].ActionsAt(now)
	}

	return tickets, nil
}
