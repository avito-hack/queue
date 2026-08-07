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

type issueTicketRepositoryStub struct {
	result          IssueTicketResult
	err             error
	calls           int
	receivedContext context.Context
	receivedCommand IssueTicketCommand
}

type issueTicketContextKey struct{}

func (s *issueTicketRepositoryStub) Issue(
	ctx context.Context,
	command IssueTicketCommand,
) (IssueTicketResult, error) {
	s.calls++
	s.receivedContext = ctx
	s.receivedCommand = command

	return s.result, s.err
}

func Test_IssueTicket_ValidRequest_ReturnCreatedTicket(t *testing.T) {
	// given
	ctx := context.WithValue(context.Background(), issueTicketContextKey{}, "value")
	request := validIssueTicketRequest()
	idempotencyKey := uuid.New()
	now := time.Date(2026, time.August, 7, 14, 0, 0, 0, time.UTC)
	activationTTL := 15 * time.Minute
	expected := validCreatedIssueTicketResult(request, now, activationTTL)
	expected.Ticket.AvailableActions = []domain.TicketAvailableAction{
		domain.TicketAvailableActionDecline,
	}
	repository := &issueTicketRepositoryStub{result: expected}
	clockCalls := 0
	useCase := NewIssueTicket(repository, activationTTL, func() time.Time {
		clockCalls++
		return now
	})

	// when
	result, err := useCase.Issue(ctx, request, idempotencyKey)

	// then
	require.NoError(t, err)
	expected.Ticket.AvailableActions = []domain.TicketAvailableAction{
		domain.TicketAvailableActionActivate,
		domain.TicketAvailableActionDecline,
	}
	assert.Equal(t, expected, result)
	assert.Equal(t, 1, repository.calls)
	assert.Same(t, ctx, repository.receivedContext)
	assert.Equal(t, IssueTicketCommand{
		QueueEntryID:       request.QueueEntryID,
		UserID:             request.UserID,
		ListingID:          request.ListingID,
		SKUID:              request.SKUID,
		IdempotencyKey:     idempotencyKey,
		IssuedAt:           now,
		ActivationDeadline: now.Add(activationTTL),
	}, repository.receivedCommand)
	assert.Equal(t, 1, clockCalls)
}

func Test_IssueTicket_ReplayedTicket_ReturnTicketForEveryValidStatus(t *testing.T) {
	// given
	now := time.Date(2026, time.August, 7, 14, 0, 0, 0, time.UTC)
	tests := []struct {
		name            string
		status          domain.TicketStatus
		expectedActions []domain.TicketAvailableAction
	}{
		{
			name:   "issued",
			status: domain.TicketStatusIssued,
			expectedActions: []domain.TicketAvailableAction{
				domain.TicketAvailableActionActivate,
				domain.TicketAvailableActionDecline,
			},
		},
		{
			name:            "redeemed",
			status:          domain.TicketStatusRedeemed,
			expectedActions: []domain.TicketAvailableAction{},
		},
		{
			name:            "closed",
			status:          domain.TicketStatusClosed,
			expectedActions: []domain.TicketAvailableAction{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := validIssueTicketRequest()
			replayed := IssueTicketResult{
				Ticket: domain.Ticket{
					ID:                 uuid.New(),
					ListingID:          request.ListingID,
					SKUID:              request.SKUID,
					Status:             test.status,
					IssuedAt:           now.Add(-time.Hour),
					ActivationDeadline: now.Add(time.Hour),
					AvailableActions:   []domain.TicketAvailableAction{domain.TicketAvailableActionActivate},
				},
				QueueEntryID: request.QueueEntryID,
				UserID:       request.UserID,
				Created:      false,
			}
			repository := &issueTicketRepositoryStub{result: replayed}
			clockCalls := 0
			useCase := NewIssueTicket(repository, 15*time.Minute, func() time.Time {
				clockCalls++
				return now
			})

			// when
			result, err := useCase.Issue(context.Background(), request, uuid.New())

			// then
			require.NoError(t, err)
			assert.False(t, result.Created)
			assert.Equal(t, test.status, result.Ticket.Status)
			assert.Equal(t, test.expectedActions, result.Ticket.AvailableActions)
			assert.Equal(t, 1, repository.calls)
			assert.Equal(t, 1, clockCalls)
		})
	}
}

func Test_IssueTicket_InvalidInput_ReturnErrorBeforeClockAndRepository(t *testing.T) {
	// given
	validRequest := validIssueTicketRequest()
	validKey := uuid.New()
	tests := []struct {
		name          string
		request       IssueTicketRequest
		key           uuid.UUID
		activationTTL time.Duration
		expectedError string
	}{
		{
			name: "empty queue entry id",
			request: IssueTicketRequest{
				UserID:    validRequest.UserID,
				ListingID: validRequest.ListingID,
				SKUID:     validRequest.SKUID,
			},
			key:           validKey,
			activationTTL: time.Minute,
			expectedError: "invalid ticket issue: empty queue entry id",
		},
		{
			name: "empty user id",
			request: IssueTicketRequest{
				QueueEntryID: validRequest.QueueEntryID,
				ListingID:    validRequest.ListingID,
				SKUID:        validRequest.SKUID,
			},
			key:           validKey,
			activationTTL: time.Minute,
			expectedError: "invalid ticket issue: empty user id",
		},
		{
			name: "empty listing id",
			request: IssueTicketRequest{
				QueueEntryID: validRequest.QueueEntryID,
				UserID:       validRequest.UserID,
				SKUID:        validRequest.SKUID,
			},
			key:           validKey,
			activationTTL: time.Minute,
			expectedError: "invalid ticket issue: empty listing id",
		},
		{
			name: "empty SKU id",
			request: IssueTicketRequest{
				QueueEntryID: validRequest.QueueEntryID,
				UserID:       validRequest.UserID,
				ListingID:    validRequest.ListingID,
			},
			key:           validKey,
			activationTTL: time.Minute,
			expectedError: "invalid ticket issue: empty SKU id",
		},
		{
			name:          "empty idempotency key",
			request:       validRequest,
			activationTTL: time.Minute,
			expectedError: "invalid ticket issue: empty idempotency key",
		},
		{
			name:          "zero activation TTL",
			request:       validRequest,
			key:           validKey,
			expectedError: "invalid ticket issue: activation TTL must be positive",
		},
		{
			name:          "negative activation TTL",
			request:       validRequest,
			key:           validKey,
			activationTTL: -time.Nanosecond,
			expectedError: "invalid ticket issue: activation TTL must be positive",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &issueTicketRepositoryStub{}
			clockCalls := 0
			useCase := NewIssueTicket(repository, test.activationTTL, func() time.Time {
				clockCalls++
				return time.Now()
			})

			// when
			result, err := useCase.Issue(context.Background(), test.request, test.key)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidTicketIssue)
			assert.EqualError(t, err, test.expectedError)
			assert.Equal(t, IssueTicketResult{}, result)
			assert.Zero(t, repository.calls)
			assert.Zero(t, clockCalls)
		})
	}
}

func Test_IssueTicket_RepositoryError_ReturnWrappedError(t *testing.T) {
	// given
	tests := []struct {
		name string
		err  error
	}{
		{name: "ticket not issuable", err: ErrTicketNotIssuable},
		{name: "idempotency conflict", err: ErrIdempotencyConflict},
		{name: "repository failure", err: errors.New("repository failure")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := validIssueTicketRequest()
			idempotencyKey := uuid.New()
			now := time.Date(2026, time.August, 7, 14, 0, 0, 0, time.UTC)
			activationTTL := 15 * time.Minute
			repository := &issueTicketRepositoryStub{err: test.err}
			clockCalls := 0
			useCase := NewIssueTicket(repository, activationTTL, func() time.Time {
				clockCalls++
				return now
			})

			// when
			result, err := useCase.Issue(context.Background(), request, idempotencyKey)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, test.err)
			assert.EqualError(t, err, "issue ticket: "+test.err.Error())
			assert.Equal(t, IssueTicketResult{}, result)
			assert.Equal(t, 1, repository.calls)
			assert.Equal(t, IssueTicketCommand{
				QueueEntryID:       request.QueueEntryID,
				UserID:             request.UserID,
				ListingID:          request.ListingID,
				SKUID:              request.SKUID,
				IdempotencyKey:     idempotencyKey,
				IssuedAt:           now,
				ActivationDeadline: now.Add(activationTTL),
			}, repository.receivedCommand)
			assert.Equal(t, 1, clockCalls)
		})
	}
}

func Test_IssueTicket_InvalidRepositoryResult_ReturnError(t *testing.T) {
	// given
	request := validIssueTicketRequest()
	now := time.Date(2026, time.August, 7, 14, 0, 0, 0, time.UTC)
	activationTTL := 15 * time.Minute
	otherTime := now.Add(time.Second)
	otherID := uuid.New()
	checkoutURL := "/checkout"
	closeReason := domain.TicketCloseReasonSystemCancelled
	tests := []struct {
		name   string
		mutate func(*IssueTicketResult)
	}{
		{name: "empty ticket id", mutate: func(result *IssueTicketResult) {
			result.Ticket.ID = uuid.Nil
		}},
		{name: "different queue entry id", mutate: func(result *IssueTicketResult) {
			result.QueueEntryID = uuid.New()
		}},
		{name: "different user id", mutate: func(result *IssueTicketResult) {
			result.UserID = uuid.New()
		}},
		{name: "different listing id", mutate: func(result *IssueTicketResult) {
			result.Ticket.ListingID = uuid.New()
		}},
		{name: "different SKU id", mutate: func(result *IssueTicketResult) {
			result.Ticket.SKUID = uuid.New()
		}},
		{name: "unknown status", mutate: func(result *IssueTicketResult) {
			result.Ticket.Status = domain.TicketStatus("unknown")
		}},
		{name: "created active ticket", mutate: func(result *IssueTicketResult) {
			result.Ticket.Status = domain.TicketStatus("active")
		}},
		{name: "different issue time", mutate: func(result *IssueTicketResult) {
			result.Ticket.IssuedAt = otherTime
		}},
		{name: "different activation deadline", mutate: func(result *IssueTicketResult) {
			result.Ticket.ActivationDeadline = otherTime
		}},
		{name: "created ticket with activation time", mutate: func(result *IssueTicketResult) {
			result.Ticket.ActivatedAt = &otherTime
		}},
		{name: "created ticket with order", mutate: func(result *IssueTicketResult) {
			result.Ticket.OrderID = &otherID
		}},
		{name: "created ticket with checkout URL", mutate: func(result *IssueTicketResult) {
			result.Ticket.CheckoutURL = &checkoutURL
		}},
		{name: "created ticket with finish time", mutate: func(result *IssueTicketResult) {
			result.Ticket.FinishedAt = &otherTime
		}},
		{name: "created ticket with close reason", mutate: func(result *IssueTicketResult) {
			result.Ticket.CloseReason = &closeReason
		}},
		{name: "replayed ticket with unknown status", mutate: func(result *IssueTicketResult) {
			result.Created = false
			result.Ticket.Status = domain.TicketStatus("unknown")
		}},
		{name: "replayed ticket with empty issue time", mutate: func(result *IssueTicketResult) {
			result.Created = false
			result.Ticket.IssuedAt = time.Time{}
		}},
		{name: "replayed ticket with deadline at issue time", mutate: func(result *IssueTicketResult) {
			result.Created = false
			result.Ticket.ActivationDeadline = result.Ticket.IssuedAt
		}},
		{name: "replayed ticket with deadline before issue time", mutate: func(result *IssueTicketResult) {
			result.Created = false
			result.Ticket.ActivationDeadline = result.Ticket.IssuedAt.Add(-time.Nanosecond)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalidResult := validCreatedIssueTicketResult(request, now, activationTTL)
			test.mutate(&invalidResult)
			repository := &issueTicketRepositoryStub{result: invalidResult}
			clockCalls := 0
			useCase := NewIssueTicket(repository, activationTTL, func() time.Time {
				clockCalls++
				return now
			})

			// when
			result, err := useCase.Issue(context.Background(), request, uuid.New())

			// then
			assert.EqualError(t, err, "issue ticket: invalid issue result")
			assert.Equal(t, IssueTicketResult{}, result)
			assert.Equal(t, 1, repository.calls)
			assert.Equal(t, 1, clockCalls)
		})
	}
}

func validIssueTicketRequest() IssueTicketRequest {
	return IssueTicketRequest{
		QueueEntryID: uuid.New(),
		UserID:       uuid.New(),
		ListingID:    uuid.New(),
		SKUID:        uuid.New(),
	}
}

func validCreatedIssueTicketResult(
	request IssueTicketRequest,
	now time.Time,
	activationTTL time.Duration,
) IssueTicketResult {
	return IssueTicketResult{
		Ticket: domain.Ticket{
			ID:                 uuid.New(),
			ListingID:          request.ListingID,
			SKUID:              request.SKUID,
			Status:             domain.TicketStatusIssued,
			IssuedAt:           now,
			ActivationDeadline: now.Add(activationTTL),
		},
		QueueEntryID: request.QueueEntryID,
		UserID:       request.UserID,
		Created:      true,
	}
}
