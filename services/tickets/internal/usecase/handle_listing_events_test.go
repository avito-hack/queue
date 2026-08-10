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

type listingEventRepositoryStub struct {
	command RevokeListingTicketsCommand
	revoked int
	err     error
}

func (s *listingEventRepositoryStub) RevokeListingTickets(
	_ context.Context,
	command RevokeListingTicketsCommand,
) (int, error) {
	s.command = command

	return s.revoked, s.err
}

func Test_HandleListingEvents_QuantityDecreased_RevokeExcessTickets(t *testing.T) {
	// given
	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	repository := &listingEventRepositoryStub{revoked: 2}
	handler := NewHandleListingEvents(repository, func() time.Time { return now })
	command := validListingQuantityChangedCommand()
	command.Quantity = 3

	// when
	revoked, err := handler.QuantityChanged(context.Background(), command)

	// then
	require.NoError(t, err)
	require.Equal(t, 2, revoked)
	require.Equal(t, 2, repository.command.RevocationLimit)
	require.Equal(t, domain.TicketCloseReasonSystemCancelled, repository.command.CloseReason)
	require.Equal(t, now, repository.command.HandledAt)
}

func Test_HandleListingEvents_StatusChanged_RevokeAllListingTickets(t *testing.T) {
	// given
	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)
	repository := &listingEventRepositoryStub{revoked: 4}
	handler := NewHandleListingEvents(repository, func() time.Time { return now })
	command := validListingStatusChangedCommand()

	// when
	revoked, err := handler.StatusChanged(context.Background(), command)

	// then
	require.NoError(t, err)
	require.Equal(t, 4, revoked)
	require.Zero(t, repository.command.RevocationLimit)
	require.True(t, repository.command.CloseAll)
	require.Equal(t, domain.TicketCloseReasonListingClosed, repository.command.CloseReason)
	require.Equal(t, now, repository.command.HandledAt)
}

func Test_HandleListingEvents_QuantityIncreased_RecordWithoutRevocation(t *testing.T) {
	// given
	repository := &listingEventRepositoryStub{}
	handler := NewHandleListingEvents(repository)
	command := validListingQuantityChangedCommand()
	command.PreviousQuantity = 3
	command.Quantity = 5

	// when
	_, err := handler.QuantityChanged(context.Background(), command)

	// then
	require.NoError(t, err)
	require.Zero(t, repository.command.RevocationLimit)
	require.False(t, repository.command.CloseAll)
}

func Test_HandleListingEvents_InvalidEvent_ReturnError(t *testing.T) {
	// given
	tests := []struct {
		name   string
		handle func(*HandleListingEvents) error
	}{
		{
			name: "negative quantity",
			handle: func(handler *HandleListingEvents) error {
				command := validListingQuantityChangedCommand()
				command.Quantity = -1
				_, err := handler.QuantityChanged(context.Background(), command)

				return err
			},
		},
		{
			name: "unknown status",
			handle: func(handler *HandleListingEvents) error {
				command := validListingStatusChangedCommand()
				command.Status = "active"
				_, err := handler.StatusChanged(context.Background(), command)

				return err
			},
		},
		{
			name: "wrong source",
			handle: func(handler *HandleListingEvents) error {
				command := validListingStatusChangedCommand()
				command.Event.Source = "unknown"
				_, err := handler.StatusChanged(context.Background(), command)

				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &listingEventRepositoryStub{}
			handler := NewHandleListingEvents(repository)

			// when
			err := test.handle(handler)

			// then
			require.ErrorIs(t, err, ErrInvalidListingEvent)
			require.Equal(t, RevokeListingTicketsCommand{}, repository.command)
		})
	}
}

func Test_HandleListingEvents_RepositoryFailed_ReturnError(t *testing.T) {
	// given
	repository := &listingEventRepositoryStub{err: errors.New("database unavailable")}
	handler := NewHandleListingEvents(repository)

	// when
	_, err := handler.QuantityChanged(context.Background(), validListingQuantityChangedCommand())

	// then
	require.EqualError(t, err, "handle listing quantity changed: database unavailable")
}

func validListingQuantityChangedCommand() ListingQuantityChangedCommand {
	return ListingQuantityChangedCommand{
		Event: IncomingListingEvent{
			ID:      uuid.New(),
			Type:    domain.ListingEventQuantityChanged,
			Source:  "avito-adapter",
			Payload: []byte(`{"listing_id":"00000000-0000-0000-0000-000000000001"}`),
		},
		ListingID:        uuid.New(),
		SellerID:         uuid.New(),
		PreviousQuantity: 5,
		Quantity:         4,
		ChangedAt:        time.Date(2026, time.August, 9, 11, 0, 0, 0, time.UTC),
	}
}

func validListingStatusChangedCommand() ListingStatusChangedCommand {
	return ListingStatusChangedCommand{
		Event: IncomingListingEvent{
			ID:      uuid.New(),
			Type:    domain.ListingEventStatusChanged,
			Source:  "avito-adapter",
			Payload: []byte(`{"listing_id":"00000000-0000-0000-0000-000000000001"}`),
		},
		ListingID:      uuid.New(),
		SellerID:       uuid.New(),
		PreviousStatus: "active",
		Status:         "paused",
		ChangedAt:      time.Date(2026, time.August, 9, 11, 0, 0, 0, time.UTC),
	}
}
