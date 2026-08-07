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

type declineRepositoryStub struct {
	result          DeclineTicketResult
	err             error
	calls           int
	receivedContext context.Context
	receivedCommand DeclineTicketCommand
}

type declineContextKey struct{}

func (s *declineRepositoryStub) Decline(
	ctx context.Context,
	command DeclineTicketCommand,
) (DeclineTicketResult, error) {
	s.calls++
	s.receivedContext = ctx
	s.receivedCommand = command

	return s.result, s.err
}

func Test_DeclineTicket_EligibleTicket_ReturnClosedTicket(t *testing.T) {
	// given
	ctx := context.WithValue(context.Background(), declineContextKey{}, "value")
	userID := uuid.New()
	ticketID := uuid.New()
	idempotencyKey := uuid.New()
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	expected := DeclineTicketResult{
		TicketID: ticketID,
		Status:   domain.TicketStatusClosed,
	}
	repository := &declineRepositoryStub{result: expected}
	clockCalls := 0
	useCase := NewDeclineTicket(repository, func() time.Time {
		clockCalls++
		return now
	})

	// when
	result, err := useCase.Decline(ctx, userID, ticketID, idempotencyKey)

	// then
	require.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.Equal(t, 1, repository.calls)
	assert.Same(t, ctx, repository.receivedContext)
	assert.Equal(t, DeclineTicketCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: idempotencyKey,
		Now:            now,
	}, repository.receivedCommand)
	assert.Equal(t, 1, clockCalls)
}

func Test_DeclineTicket_InvalidInput_ReturnError(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	idempotencyKey := uuid.New()
	tests := []struct {
		name           string
		userID         uuid.UUID
		ticketID       uuid.UUID
		idempotencyKey uuid.UUID
		expectedError  string
	}{
		{
			name:           "empty user id",
			ticketID:       ticketID,
			idempotencyKey: idempotencyKey,
			expectedError:  "invalid ticket decline: empty user id",
		},
		{
			name:           "empty ticket id",
			userID:         userID,
			idempotencyKey: idempotencyKey,
			expectedError:  "invalid ticket decline: empty ticket id",
		},
		{
			name:          "empty idempotency key",
			userID:        userID,
			ticketID:      ticketID,
			expectedError: "invalid ticket decline: empty idempotency key",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &declineRepositoryStub{}
			clockCalls := 0
			useCase := NewDeclineTicket(repository, func() time.Time {
				clockCalls++
				return time.Now()
			})

			// when
			result, err := useCase.Decline(
				context.Background(),
				test.userID,
				test.ticketID,
				test.idempotencyKey,
			)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidDecline)
			assert.EqualError(t, err, test.expectedError)
			assert.Equal(t, DeclineTicketResult{}, result)
			assert.Zero(t, repository.calls)
			assert.Zero(t, clockCalls)
		})
	}
}

func Test_DeclineTicket_RepositoryError_ReturnWrappedError(t *testing.T) {
	// given
	tests := []struct {
		name string
		err  error
	}{
		{name: "ticket not found", err: ErrTicketNotFound},
		{name: "ticket not declinable", err: ErrTicketNotDeclinable},
		{name: "idempotency conflict", err: ErrIdempotencyConflict},
		{name: "activation in progress", err: ErrActivationInProgress},
		{name: "repository failure", err: errors.New("repository failure")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userID := uuid.New()
			ticketID := uuid.New()
			idempotencyKey := uuid.New()
			now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
			repository := &declineRepositoryStub{err: test.err}
			useCase := NewDeclineTicket(repository, func() time.Time { return now })

			// when
			result, err := useCase.Decline(context.Background(), userID, ticketID, idempotencyKey)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, test.err)
			assert.EqualError(t, err, "decline ticket: "+test.err.Error())
			assert.Equal(t, DeclineTicketResult{}, result)
			assert.Equal(t, 1, repository.calls)
			assert.Equal(t, DeclineTicketCommand{
				UserID:         userID,
				TicketID:       ticketID,
				IdempotencyKey: idempotencyKey,
				Now:            now,
			}, repository.receivedCommand)
		})
	}
}

func Test_DeclineTicket_InvalidRepositoryResult_ReturnError(t *testing.T) {
	// given
	ticketID := uuid.New()
	tests := []struct {
		name   string
		result DeclineTicketResult
	}{
		{
			name: "empty ticket id",
			result: DeclineTicketResult{
				Status: domain.TicketStatusClosed,
			},
		},
		{
			name: "different ticket id",
			result: DeclineTicketResult{
				TicketID: uuid.New(),
				Status:   domain.TicketStatusClosed,
			},
		},
		{
			name: "issued status",
			result: DeclineTicketResult{
				TicketID: ticketID,
				Status:   domain.TicketStatusIssued,
			},
		},
		{
			name: "active status",
			result: DeclineTicketResult{
				TicketID: ticketID,
				Status:   domain.TicketStatusActive,
			},
		},
		{
			name: "redeemed status",
			result: DeclineTicketResult{
				TicketID: ticketID,
				Status:   domain.TicketStatusRedeemed,
			},
		},
		{
			name: "unknown status",
			result: DeclineTicketResult{
				TicketID: ticketID,
				Status:   domain.TicketStatus("unknown"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &declineRepositoryStub{result: test.result}
			useCase := NewDeclineTicket(repository, time.Now)

			// when
			result, err := useCase.Decline(context.Background(), uuid.New(), ticketID, uuid.New())

			// then
			assert.EqualError(t, err, "decline ticket: invalid decline result")
			assert.Equal(t, DeclineTicketResult{}, result)
			assert.Equal(t, 1, repository.calls)
		})
	}
}
