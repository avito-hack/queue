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

type issueTicketRepositoryStub struct {
	result          IssueTicketResult
	err             error
	calls           int
	receivedContext context.Context
	receivedCommand IssueTicketCommand
}

type issueTicketListingReaderStub struct {
	listing    TicketIssueListing
	err        error
	calls      int
	receivedID uuid.UUID
}

func (s *issueTicketListingReaderStub) Get(_ context.Context, listingID uuid.UUID) (TicketIssueListing, error) {
	s.calls++
	s.receivedID = listingID

	return s.listing, s.err
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
	listingReader := validIssueListingReader(request)
	clockCalls := 0
	useCase := NewIssueTicket(repository, listingReader, activationTTL, func() time.Time {
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
	require.Equal(t, expected, result)
	require.Equal(t, 1, repository.calls)
	require.Same(t, ctx, repository.receivedContext)
	require.Equal(t, IssueTicketCommand{
		QueueEntryID:       request.QueueEntryID,
		UserID:             request.UserID,
		ListingID:          request.ListingID,
		SKUID:              request.SKUID,
		ListingQuantity:    10,
		IdempotencyKey:     idempotencyKey,
		IssuedAt:           now,
		ActivationDeadline: now.Add(activationTTL),
	}, repository.receivedCommand)
	require.Equal(t, 1, listingReader.calls)
	require.Equal(t, request.ListingID, listingReader.receivedID)
	require.Equal(t, 1, clockCalls)
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
			listingReader := validIssueListingReader(request)
			clockCalls := 0
			useCase := NewIssueTicket(repository, listingReader, 15*time.Minute, func() time.Time {
				clockCalls++
				return now
			})

			// when
			result, err := useCase.Issue(context.Background(), request, uuid.New())

			// then
			require.NoError(t, err)
			require.False(t, result.Created)
			require.Equal(t, test.status, result.Ticket.Status)
			require.Equal(t, test.expectedActions, result.Ticket.AvailableActions)
			require.Equal(t, 1, repository.calls)
			require.Equal(t, 1, listingReader.calls)
			require.Equal(t, 1, clockCalls)
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
			listingReader := validIssueListingReader(validRequest)
			clockCalls := 0
			useCase := NewIssueTicket(repository, listingReader, test.activationTTL, func() time.Time {
				clockCalls++
				return time.Now()
			})

			// when
			result, err := useCase.Issue(context.Background(), test.request, test.key)

			// then
			require.Error(t, err)
			require.ErrorIs(t, err, ErrInvalidTicketIssue)
			require.EqualError(t, err, test.expectedError)
			require.Equal(t, IssueTicketResult{}, result)
			require.Zero(t, repository.calls)
			require.Zero(t, listingReader.calls)
			require.Zero(t, clockCalls)
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
			listingReader := validIssueListingReader(request)
			clockCalls := 0
			useCase := NewIssueTicket(repository, listingReader, activationTTL, func() time.Time {
				clockCalls++
				return now
			})

			// when
			result, err := useCase.Issue(context.Background(), request, idempotencyKey)

			// then
			require.Error(t, err)
			require.ErrorIs(t, err, test.err)
			require.EqualError(t, err, "issue ticket: "+test.err.Error())
			require.Equal(t, IssueTicketResult{}, result)
			require.Equal(t, 1, repository.calls)
			require.Equal(t, IssueTicketCommand{
				QueueEntryID:       request.QueueEntryID,
				UserID:             request.UserID,
				ListingID:          request.ListingID,
				SKUID:              request.SKUID,
				ListingQuantity:    10,
				IdempotencyKey:     idempotencyKey,
				IssuedAt:           now,
				ActivationDeadline: now.Add(activationTTL),
			}, repository.receivedCommand)
			require.Equal(t, 1, listingReader.calls)
			require.Equal(t, 1, clockCalls)
		})
	}
}

func Test_IssueTicket_ListingCannotIssue_ReturnNotIssuable(t *testing.T) {
	// given
	request := validIssueTicketRequest()
	tests := []struct {
		name    string
		listing TicketIssueListing
	}{
		{name: "empty quantity", listing: TicketIssueListing{ID: request.ListingID, QueueEnabled: true, Status: "active"}},
		{name: "queue disabled", listing: TicketIssueListing{ID: request.ListingID, Quantity: 1, Status: "active"}},
		{name: "listing paused", listing: TicketIssueListing{ID: request.ListingID, Quantity: 1, QueueEnabled: true, Status: "paused"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			listingReader := &issueTicketListingReaderStub{listing: test.listing}
			repository := &issueTicketRepositoryStub{}
			clockCalls := 0
			useCase := NewIssueTicket(repository, listingReader, time.Minute, func() time.Time {
				clockCalls++
				return time.Now()
			})

			// when
			_, err := useCase.Issue(context.Background(), request, uuid.New())

			// then
			require.ErrorIs(t, err, ErrTicketNotIssuable)
			require.Equal(t, 1, listingReader.calls)
			require.Zero(t, repository.calls)
			require.Zero(t, clockCalls)
		})
	}
}

func Test_IssueTicket_ListingReaderFailure_ReturnWrappedError(t *testing.T) {
	// given
	request := validIssueTicketRequest()
	listingError := errors.New("listing failed")
	listingReader := &issueTicketListingReaderStub{err: listingError}
	repository := &issueTicketRepositoryStub{}
	useCase := NewIssueTicket(repository, listingReader, time.Minute, time.Now)

	// when
	_, err := useCase.Issue(context.Background(), request, uuid.New())

	// then
	require.EqualError(t, err, "get listing for ticket issue: listing failed")
	require.ErrorIs(t, err, listingError)
	require.Zero(t, repository.calls)
}

func Test_IssueTicket_InvalidListingSnapshot_ReturnError(t *testing.T) {
	// given
	request := validIssueTicketRequest()
	listingReader := validIssueListingReader(request)
	listingReader.listing.ID = uuid.New()
	repository := &issueTicketRepositoryStub{}
	useCase := NewIssueTicket(repository, listingReader, time.Minute, time.Now)

	// when
	_, err := useCase.Issue(context.Background(), request, uuid.New())

	// then
	require.EqualError(t, err, "get listing for ticket issue: invalid listing")
	require.Zero(t, repository.calls)
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
			listingReader := validIssueListingReader(request)
			clockCalls := 0
			useCase := NewIssueTicket(repository, listingReader, activationTTL, func() time.Time {
				clockCalls++
				return now
			})

			// when
			result, err := useCase.Issue(context.Background(), request, uuid.New())

			// then
			require.EqualError(t, err, "issue ticket: invalid issue result")
			require.Equal(t, IssueTicketResult{}, result)
			require.Equal(t, 1, repository.calls)
			require.Equal(t, 1, listingReader.calls)
			require.Equal(t, 1, clockCalls)
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

func validIssueListingReader(request IssueTicketRequest) *issueTicketListingReaderStub {
	return &issueTicketListingReaderStub{listing: TicketIssueListing{
		ID:           request.ListingID,
		Quantity:     10,
		QueueEnabled: true,
		Status:       "active",
	}}
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
