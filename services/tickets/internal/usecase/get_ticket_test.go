package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
)

type ticketGetterStub struct {
	ticket           domain.Ticket
	err              error
	receivedUserID   uuid.UUID
	receivedTicketID uuid.UUID
	calls            int
}

func (s *ticketGetterStub) Get(_ context.Context, userID, ticketID uuid.UUID) (domain.Ticket, error) {
	s.calls++
	s.receivedUserID = userID
	s.receivedTicketID = ticketID

	return s.ticket, s.err
}

func TestGetTicket_Get_ReturnTicketWithActions(t *testing.T) {
	// given
	now := time.Date(2026, time.August, 6, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	ticketID := uuid.New()
	repository := &ticketGetterStub{ticket: domain.Ticket{
		ID:                 ticketID,
		Status:             domain.TicketStatusIssued,
		ActivationDeadline: now.Add(time.Minute),
	}}
	getTicket := NewGetTicket(repository, func() time.Time { return now })

	// when
	ticket, err := getTicket.Get(context.Background(), userID, ticketID)

	// then
	require.NoError(t, err)
	require.Equal(t, ticketID, ticket.ID)
	require.Equal(t, []domain.TicketAvailableAction{
		domain.TicketAvailableActionActivate,
		domain.TicketAvailableActionDecline,
	}, ticket.AvailableActions)
	require.Equal(t, userID, repository.receivedUserID)
	require.Equal(t, ticketID, repository.receivedTicketID)
	require.Equal(t, 1, repository.calls)
}

func TestGetTicket_Get_RepositoryReturnsNotFound_ReturnWrappedError(t *testing.T) {
	// given
	repository := &ticketGetterStub{err: ErrTicketNotFound}
	getTicket := NewGetTicket(repository)

	// when
	_, err := getTicket.Get(context.Background(), uuid.New(), uuid.New())

	// then
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTicketNotFound)
	require.Equal(t, "get ticket: ticket not found", err.Error())
	require.Equal(t, 1, repository.calls)
}

func TestGetTicket_Get_RepositoryReturnsError_ReturnWrappedError(t *testing.T) {
	// given
	repositoryError := errors.New("repository failed")
	repository := &ticketGetterStub{err: repositoryError}
	getTicket := NewGetTicket(repository)

	// when
	_, err := getTicket.Get(context.Background(), uuid.New(), uuid.New())

	// then
	require.Error(t, err)
	require.ErrorIs(t, err, repositoryError)
	require.Equal(t, "get ticket: repository failed", err.Error())
	require.Equal(t, 1, repository.calls)
}
