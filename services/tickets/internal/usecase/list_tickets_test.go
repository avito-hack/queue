package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
)

type ticketRepositoryStub struct {
	tickets        []domain.Ticket
	err            error
	receivedUserID uuid.UUID
	receivedFilter ListTicketsFilter
	calls          int
}

func (s *ticketRepositoryStub) List(_ context.Context, userID uuid.UUID, filter ListTicketsFilter) ([]domain.Ticket, error) {
	s.calls++
	s.receivedUserID = userID
	s.receivedFilter = filter

	return s.tickets, s.err
}

func TestListTickets_List_ReturnTicketsWithActions(t *testing.T) {
	// given
	now := time.Date(2026, time.August, 6, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	status := domain.TicketStatusIssued
	filter := ListTicketsFilter{Status: &status, ListingID: &listingID, SKUID: &skuID}
	repository := &ticketRepositoryStub{tickets: []domain.Ticket{
		{ID: uuid.New(), Status: domain.TicketStatusIssued, ActivationDeadline: now.Add(time.Minute)},
		{ID: uuid.New(), Status: domain.TicketStatusClosed, ActivationDeadline: now.Add(time.Minute)},
	}}
	listTickets := NewListTickets(repository, func() time.Time { return now })

	// when
	tickets, err := listTickets.List(context.Background(), userID, filter)

	// then
	require.NoError(t, err)
	require.Len(t, tickets, 2)
	assert.Equal(t, []domain.TicketAvailableAction{
		domain.TicketAvailableActionActivate,
		domain.TicketAvailableActionDecline,
	}, tickets[0].AvailableActions)
	assert.Empty(t, tickets[1].AvailableActions)
	assert.NotNil(t, tickets[1].AvailableActions)
	assert.Equal(t, userID, repository.receivedUserID)
	assert.Equal(t, filter, repository.receivedFilter)
	assert.Equal(t, 1, repository.calls)
}

func TestListTickets_List_RepositoryReturnsNil_ReturnEmptySlice(t *testing.T) {
	// given
	repository := &ticketRepositoryStub{}
	listTickets := NewListTickets(repository)

	// when
	tickets, err := listTickets.List(context.Background(), uuid.New(), ListTicketsFilter{})

	// then
	require.NoError(t, err)
	assert.Empty(t, tickets)
	assert.NotNil(t, tickets)
}

func TestListTickets_List_RepositoryReturnsError_ReturnWrappedError(t *testing.T) {
	// given
	repositoryError := errors.New("repository failed")
	repository := &ticketRepositoryStub{err: repositoryError}
	listTickets := NewListTickets(repository)

	// when
	_, err := listTickets.List(context.Background(), uuid.New(), ListTicketsFilter{})

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, repositoryError)
	assert.Equal(t, "list tickets: repository failed", err.Error())
}

func TestListTickets_List_InvalidFilter_ReturnError(t *testing.T) {
	// given
	status := domain.TicketStatus("unknown")
	repository := &ticketRepositoryStub{}
	listTickets := NewListTickets(repository)

	// when
	_, err := listTickets.List(context.Background(), uuid.New(), ListTicketsFilter{Status: &status})

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTicketFilter)
	assert.Zero(t, repository.calls)
}

func TestListTickets_List_EmptyUserID_ReturnError(t *testing.T) {
	// given
	repository := &ticketRepositoryStub{}
	listTickets := NewListTickets(repository)

	// when
	_, err := listTickets.List(context.Background(), uuid.Nil, ListTicketsFilter{})

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTicketFilter)
	assert.Zero(t, repository.calls)
}
