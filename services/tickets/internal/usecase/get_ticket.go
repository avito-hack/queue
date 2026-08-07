package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
)

var ErrTicketNotFound = errors.New("ticket not found")

type TicketGetter interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.Ticket, error)
}

type GetTicket struct {
	repository TicketGetter
	clock      Clock
}

func NewGetTicket(repository TicketGetter, clocks ...Clock) *GetTicket {
	clock := Clock(time.Now)
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}

	return &GetTicket{
		repository: repository,
		clock:      clock,
	}
}

func (u *GetTicket) Get(ctx context.Context, userID, ticketID uuid.UUID) (domain.Ticket, error) {
	ticket, err := u.repository.Get(ctx, userID, ticketID)
	if err != nil {
		return domain.Ticket{}, fmt.Errorf("get ticket: %w", err)
	}

	ticket.AvailableActions = ticket.ActionsAt(u.clock())

	return ticket, nil
}
