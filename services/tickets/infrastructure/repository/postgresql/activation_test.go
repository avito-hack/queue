package postgresql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type activationRowStub struct {
	scan func([]any) error
}

type activationRow = pgx.Row

func (s activationRowStub) Scan(destinations ...any) error {
	return s.scan(destinations)
}

type activationQueryCall struct {
	query string
	args  []any
}

type activationExecCall struct {
	query string
	args  []any
}

type activationExecResult struct {
	commandTag pgconn.CommandTag
	err        error
}

type activationTransactionStub struct {
	rows           []activationRow
	execResults    []activationExecResult
	queryCalls     []activationQueryCall
	execCalls      []activationExecCall
	commitErr      error
	rollbackErr    error
	rollbackCtxErr error
	commitCalls    int
	rollbackCalls  int
}

func (s *activationTransactionStub) QueryRow(
	_ context.Context,
	query string,
	args ...any,
) activationRow {
	callIndex := len(s.queryCalls)
	s.queryCalls = append(s.queryCalls, activationQueryCall{query: query, args: args})
	if callIndex >= len(s.rows) {
		return activationScanErrorRow(errors.New("unexpected query row call"))
	}

	return s.rows[callIndex]
}

func (s *activationTransactionStub) Query(
	context.Context,
	string,
	...any,
) (pgx.Rows, error) {
	return nil, errors.New("unexpected query call")
}

func (s *activationTransactionStub) Exec(
	_ context.Context,
	query string,
	args ...any,
) (pgconn.CommandTag, error) {
	callIndex := len(s.execCalls)
	s.execCalls = append(s.execCalls, activationExecCall{query: query, args: args})
	if callIndex >= len(s.execResults) {
		return pgconn.CommandTag{}, errors.New("unexpected exec call")
	}

	return s.execResults[callIndex].commandTag, s.execResults[callIndex].err
}

func (s *activationTransactionStub) Commit(context.Context) error {
	s.commitCalls++

	return s.commitErr
}

func (s *activationTransactionStub) Rollback(ctx context.Context) error {
	s.rollbackCalls++
	s.rollbackCtxErr = ctx.Err()

	return s.rollbackErr
}

type activationTransactionBeginnerStub struct {
	transaction activationTransaction
	err         error
	calls       int
}

func (s *activationTransactionBeginnerStub) Begin(context.Context) (activationTransaction, error) {
	s.calls++

	return s.transaction, s.err
}

func TestActivationRepository_Prepare_IssuedTicket_ReturnPreparedActivation(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	idempotencyKey := uuid.New()
	operationID := uuid.New()
	now := time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC)
	transaction := &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			activationTicketRow(activationTicketRecord{
				ListingID:          listingID,
				SKUID:              skuID,
				Status:             domain.TicketStatusIssued,
				ActivationDeadline: now.Add(time.Minute),
			}),
			activationScanErrorRow(pgx.ErrNoRows),
			activationScanErrorRow(pgx.ErrNoRows),
		},
		execResults: []activationExecResult{{commandTag: pgconn.NewCommandTag("INSERT 0 1")}},
	}
	beginner := &activationTransactionBeginnerStub{transaction: transaction}
	repository := &ActivationRepository{
		transactions: beginner,
		newID:        func() uuid.UUID { return operationID },
	}
	command := usecase.PrepareActivationCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: idempotencyKey,
		Now:            now,
	}

	// when
	prepared, err := repository.Prepare(context.Background(), command)

	// then
	require.NoError(t, err)
	require.Equal(t, usecase.PreparedActivation{
		OperationID: operationID,
		Order: usecase.CreateOrderRequest{
			TicketID:       ticketID,
			ListingID:      listingID,
			SKUID:          skuID,
			UserID:         userID,
			IdempotencyKey: operationID,
		},
	}, prepared)
	require.Len(t, transaction.queryCalls, 4)
	require.Contains(t, transaction.queryCalls[0].query, "actor_id = $1")
	require.Contains(t, transaction.queryCalls[0].query, "FOR UPDATE")
	require.Equal(t, []any{toPGUUID(userID), activationOperation, toPGUUID(idempotencyKey)}, transaction.queryCalls[0].args)
	require.Contains(t, transaction.queryCalls[1].query, "WHERE id = $1")
	require.Contains(t, transaction.queryCalls[1].query, "user_id = $2")
	require.Contains(t, transaction.queryCalls[1].query, "FOR UPDATE")
	require.Equal(t, []any{toPGUUID(ticketID), toPGUUID(userID)}, transaction.queryCalls[1].args)
	require.Equal(t, []any{
		toPGUUID(ticketID),
		activationOperation,
		activationOperationStateProcessing,
	}, transaction.queryCalls[3].args)
	require.NotContains(t, transaction.queryCalls[2].query, "FOR UPDATE")
	require.Len(t, transaction.execCalls, 1)
	require.Contains(t, transaction.execCalls[0].query, "state")
	require.Contains(t, transaction.execCalls[0].query, "ON CONFLICT")
	require.Equal(t, []any{
		toPGUUID(operationID),
		toPGUUID(idempotencyKey),
		activationOperation,
		toPGUUID(userID),
		toPGUUID(ticketID),
		activationRequestHash(userID, ticketID),
		activationOperationStateProcessing,
		toPGTimestamptz(now.Add(activationOperationTTL)),
		toPGTimestamptz(now),
	}, transaction.execCalls[0].args)
	require.Equal(t, 1, beginner.calls)
	require.Equal(t, 1, transaction.commitCalls)
	require.Zero(t, transaction.rollbackCalls)
}

func TestActivationRepository_Prepare_CompletedOperation_ReturnReplay(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	idempotencyKey := uuid.New()
	expected := usecase.ActivationResult{
		TicketID:    ticketID,
		Status:      domain.TicketStatusRedeemed,
		OrderID:     uuid.New(),
		CheckoutURL: "/checkout/replayed",
	}
	responseBody, err := encodeActivationResult(expected)
	require.NoError(t, err)
	responseStatus := int32(activationResponseStatus)
	transaction := &activationTransactionStub{rows: []activationRow{activationOperationRow(
		activationOperationRecord{
			ID:             uuid.New(),
			ActorID:        userID,
			TicketID:       ticketID,
			RequestHash:    activationRequestHash(userID, ticketID),
			State:          activationOperationStateComplete,
			ResponseStatus: &responseStatus,
			ResponseBody:   responseBody,
		},
	)}}
	repository := newActivationRepositoryForTest(transaction)

	// when
	prepared, err := repository.Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: idempotencyKey,
		Now:            time.Now(),
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, prepared.Replay)
	require.Equal(t, expected, *prepared.Replay)
	require.Empty(t, transaction.execCalls)
	require.Equal(t, 1, transaction.commitCalls)
	require.Zero(t, transaction.rollbackCalls)
}

func TestActivationRepository_Prepare_ExpiredProcessingOperationWithSameKey_ReturnActivationExpired(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	idempotencyKey := uuid.New()
	operationID := uuid.New()
	transaction := &activationTransactionStub{rows: []activationRow{
		activationOperationRow(activationOperationRecord{
			ID:          operationID,
			ActorID:     userID,
			TicketID:    ticketID,
			RequestHash: activationRequestHash(userID, ticketID),
			State:       activationOperationStateProcessing,
		}),
		activationTicketRow(activationTicketRecord{
			ListingID:          listingID,
			SKUID:              skuID,
			Status:             domain.TicketStatusIssued,
			ActivationDeadline: time.Now().Add(-time.Hour),
		}),
	}}
	repository := newActivationRepositoryForTest(transaction)

	// when
	_, err := repository.Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: idempotencyKey,
		Now:            time.Now(),
	})

	// then
	require.ErrorIs(t, err, usecase.ErrTicketActivationExpired)
	require.Len(t, transaction.queryCalls, 2)
	require.Empty(t, transaction.execCalls)
	require.Zero(t, transaction.commitCalls)
	require.Equal(t, 1, transaction.rollbackCalls)
}

func TestActivationRepository_Prepare_SameKeyForAnotherTicket_ReturnIdempotencyConflict(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	operationTicketID := uuid.New()
	transaction := &activationTransactionStub{rows: []activationRow{activationOperationRow(
		activationOperationRecord{
			ID:          uuid.New(),
			ActorID:     userID,
			TicketID:    operationTicketID,
			RequestHash: activationRequestHash(userID, operationTicketID),
			State:       activationOperationStateProcessing,
		},
	)}}
	repository := newActivationRepositoryForTest(transaction)

	// when
	_, err := repository.Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: uuid.New(),
		Now:            time.Now(),
	})

	// then
	require.ErrorIs(t, err, usecase.ErrIdempotencyConflict)
	require.Empty(t, transaction.execCalls)
	require.Zero(t, transaction.commitCalls)
	require.Equal(t, 1, transaction.rollbackCalls)
}

func TestActivationRepository_Prepare_IneligibleTicket_ReturnDomainError(t *testing.T) {
	tests := []struct {
		name          string
		ticketRow     activationRow
		extraRows     []activationRow
		expectedError error
	}{
		{
			name:          "ticket not found",
			ticketRow:     activationScanErrorRow(pgx.ErrNoRows),
			expectedError: usecase.ErrTicketNotFound,
		},
		{
			name: "ticket already redeemed",
			ticketRow: activationTicketRow(activationTicketRecord{
				ListingID:          uuid.New(),
				SKUID:              uuid.New(),
				Status:             domain.TicketStatusRedeemed,
				ActivationDeadline: time.Now().Add(time.Hour),
			}),
			extraRows:     []activationRow{activationScanErrorRow(pgx.ErrNoRows)},
			expectedError: usecase.ErrTicketNotActivatable,
		},
		{
			name: "activation deadline reached",
			ticketRow: activationTicketRow(activationTicketRecord{
				ListingID:          uuid.New(),
				SKUID:              uuid.New(),
				Status:             domain.TicketStatusIssued,
				ActivationDeadline: time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC),
			}),
			extraRows: []activationRow{
				activationScanErrorRow(pgx.ErrNoRows),
				activationScanErrorRow(pgx.ErrNoRows),
			},
			expectedError: usecase.ErrTicketActivationExpired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			rows := []activationRow{
				activationScanErrorRow(pgx.ErrNoRows),
				test.ticketRow,
			}
			rows = append(rows, test.extraRows...)
			transaction := &activationTransactionStub{rows: rows}
			repository := newActivationRepositoryForTest(transaction)

			// when
			_, err := repository.Prepare(context.Background(), usecase.PrepareActivationCommand{
				UserID:         uuid.New(),
				TicketID:       uuid.New(),
				IdempotencyKey: uuid.New(),
				Now:            time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC),
			})

			// then
			require.ErrorIs(t, err, test.expectedError)
			require.Empty(t, transaction.execCalls)
			require.Zero(t, transaction.commitCalls)
			require.Equal(t, 1, transaction.rollbackCalls)
		})
	}
}

func TestActivationRepository_Prepare_ConcurrentSameKeyCompletion_ReturnReplay(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	expected := usecase.ActivationResult{
		TicketID:    ticketID,
		Status:      domain.TicketStatusRedeemed,
		OrderID:     uuid.New(),
		CheckoutURL: "/checkout/concurrent",
	}
	responseBody, err := encodeActivationResult(expected)
	require.NoError(t, err)
	responseStatus := int32(activationResponseStatus)
	transaction := &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			activationTicketRow(activationTicketRecord{
				ListingID:          uuid.New(),
				SKUID:              uuid.New(),
				Status:             domain.TicketStatusRedeemed,
				ActivationDeadline: time.Now().Add(time.Hour),
			}),
			activationOperationRow(activationOperationRecord{
				ID:             uuid.New(),
				ActorID:        userID,
				TicketID:       ticketID,
				RequestHash:    activationRequestHash(userID, ticketID),
				State:          activationOperationStateComplete,
				ResponseStatus: &responseStatus,
				ResponseBody:   responseBody,
			}),
		},
	}
	repository := newActivationRepositoryForTest(transaction)

	// when
	prepared, err := repository.Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: uuid.New(),
		Now:            time.Now(),
	})

	// then
	require.NoError(t, err)
	require.NotNil(t, prepared.Replay)
	require.Equal(t, expected, *prepared.Replay)
	require.Len(t, transaction.queryCalls, 3)
	require.NotContains(t, transaction.queryCalls[2].query, "FOR UPDATE")
	require.Empty(t, transaction.execCalls)
	require.Equal(t, 1, transaction.commitCalls)
}

func TestActivationRepository_Prepare_ConcurrentExpiredSameKeyProcessing_ReturnActivationExpired(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	idempotencyKey := uuid.New()
	operationID := uuid.New()
	transaction := &activationTransactionStub{rows: []activationRow{
		activationScanErrorRow(pgx.ErrNoRows),
		activationTicketRow(activationTicketRecord{
			ListingID:          listingID,
			SKUID:              skuID,
			Status:             domain.TicketStatusIssued,
			ActivationDeadline: time.Now().Add(-time.Hour),
		}),
		activationOperationRow(activationOperationRecord{
			ID:          operationID,
			ActorID:     userID,
			TicketID:    ticketID,
			RequestHash: activationRequestHash(userID, ticketID),
			State:       activationOperationStateProcessing,
		}),
	}}
	repository := newActivationRepositoryForTest(transaction)

	// when
	_, err := repository.Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: idempotencyKey,
		Now:            time.Now(),
	})

	// then
	require.ErrorIs(t, err, usecase.ErrTicketActivationExpired)
	require.Len(t, transaction.queryCalls, 3)
	require.NotContains(t, transaction.queryCalls[2].query, "FOR UPDATE")
	require.Empty(t, transaction.execCalls)
	require.Zero(t, transaction.commitCalls)
	require.Equal(t, 1, transaction.rollbackCalls)
}

func TestActivationRepository_Prepare_DifferentKeyOperationIsProcessing_ReturnActivationInProgress(t *testing.T) {
	// given
	now := time.Now()
	transaction := &activationTransactionStub{rows: []activationRow{
		activationScanErrorRow(pgx.ErrNoRows),
		activationTicketRow(activationTicketRecord{
			ListingID:          uuid.New(),
			SKUID:              uuid.New(),
			Status:             domain.TicketStatusIssued,
			ActivationDeadline: now.Add(-time.Minute),
		}),
		activationScanErrorRow(pgx.ErrNoRows),
		activationUUIDRow(uuid.New()),
	}}
	repository := newActivationRepositoryForTest(transaction)

	// when
	_, err := repository.Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            now,
	})

	// then
	require.ErrorIs(t, err, usecase.ErrActivationInProgress)
	require.Len(t, transaction.queryCalls, 4)
	require.Contains(t, transaction.queryCalls[3].query, "ticket_id = $1")
	require.Contains(t, transaction.queryCalls[3].query, "state = $3")
	require.NotContains(t, transaction.queryCalls[3].query, "FOR UPDATE")
	require.Empty(t, transaction.execCalls)
	require.Zero(t, transaction.commitCalls)
	require.Equal(t, 1, transaction.rollbackCalls)
}

func TestActivationRepository_Prepare_TicketOperationUniqueViolation_ReturnActivationInProgress(t *testing.T) {
	// given
	now := time.Now()
	uniqueViolation := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: activationTicketConstraint,
	}
	transaction := &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			activationTicketRow(activationTicketRecord{
				ListingID:          uuid.New(),
				SKUID:              uuid.New(),
				Status:             domain.TicketStatusIssued,
				ActivationDeadline: now.Add(time.Minute),
			}),
			activationScanErrorRow(pgx.ErrNoRows),
			activationScanErrorRow(pgx.ErrNoRows),
		},
		execResults: []activationExecResult{{err: uniqueViolation}},
	}
	repository := newActivationRepositoryForTest(transaction)

	// when
	_, err := repository.Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            now,
	})

	// then
	require.ErrorIs(t, err, usecase.ErrActivationInProgress)
	require.Zero(t, transaction.commitCalls)
	require.Equal(t, 1, transaction.rollbackCalls)
}

func TestActivationRepository_Complete_ProcessingOperation_ActivateTicketAndWriteOutbox(t *testing.T) {
	// given
	operationID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	order := usecase.CreatedOrder{ID: uuid.New(), CheckoutURL: "/checkout/created"}
	eventID := uuid.New()
	completedAt := time.Date(2026, time.August, 7, 10, 0, 1, 0, time.UTC)
	transaction := &activationTransactionStub{
		rows: []activationRow{activationOperationRow(activationOperationRecord{
			ID:          operationID,
			ActorID:     userID,
			TicketID:    ticketID,
			RequestHash: activationRequestHash(userID, ticketID),
			State:       activationOperationStateProcessing,
		})},
		execResults: []activationExecResult{
			{commandTag: pgconn.NewCommandTag("UPDATE 1")},
			{commandTag: pgconn.NewCommandTag("UPDATE 1")},
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
		},
	}
	repository := &ActivationRepository{
		transactions: &activationTransactionBeginnerStub{transaction: transaction},
		newID:        func() uuid.UUID { return eventID },
	}

	// when
	result, err := repository.Complete(context.Background(), operationID, order, completedAt)

	// then
	require.NoError(t, err)
	require.Equal(t, usecase.ActivationResult{
		TicketID:    ticketID,
		Status:      domain.TicketStatusRedeemed,
		OrderID:     order.ID,
		CheckoutURL: order.CheckoutURL,
	}, result)
	require.Len(t, transaction.queryCalls, 1)
	require.Contains(t, transaction.queryCalls[0].query, "WHERE id = $1")
	require.Contains(t, transaction.queryCalls[0].query, "operation = $2")
	require.Contains(t, transaction.queryCalls[0].query, "FOR UPDATE")
	require.Equal(t, []any{toPGUUID(operationID), activationOperation}, transaction.queryCalls[0].args)
	require.Len(t, transaction.execCalls, 3)
	require.Contains(t, transaction.execCalls[0].query, "status = 'redeemed'")
	require.Equal(t, []any{
		toPGTimestamptz(completedAt),
		toPGUUID(order.ID),
		toPGText(order.CheckoutURL),
		toPGUUID(ticketID),
		toPGUUID(userID),
	}, transaction.execCalls[0].args)
	require.Contains(t, transaction.execCalls[1].query, "state = 'completed'")
	require.Equal(t, toPGInt4(activationResponseStatus), transaction.execCalls[1].args[0])
	require.Equal(t, toPGUUID(operationID), transaction.execCalls[1].args[3])
	require.JSONEq(t, `{
		"ticket_id":"`+ticketID.String()+`",
		"status":"redeemed",
		"order_id":"`+order.ID.String()+`",
		"checkout_url":"/checkout/created"
	}`, string(transaction.execCalls[1].args[1].([]byte)))
	require.Contains(t, transaction.execCalls[2].query, "public.outbox_events")
	require.Equal(t, toPGUUID(eventID), transaction.execCalls[2].args[0])
	require.Equal(t, activationOutboxAggregateType, transaction.execCalls[2].args[1])
	require.Equal(t, toPGUUID(ticketID), transaction.execCalls[2].args[2])
	require.Equal(t, domain.TicketEventRedeemed, transaction.execCalls[2].args[3])
	require.Equal(t, activationOutboxState, transaction.execCalls[2].args[5])
	require.JSONEq(t, `{
		"ticket_id":"`+ticketID.String()+`",
		"user_id":"`+userID.String()+`",
		"order_id":"`+order.ID.String()+`",
		"checkout_url":"/checkout/created",
		"status":"redeemed",
		"redeemed_at":"2026-08-07T10:00:01Z"
	}`, string(transaction.execCalls[2].args[4].([]byte)))
	require.Equal(t, 1, transaction.commitCalls)
	require.Zero(t, transaction.rollbackCalls)
}

func TestActivationRepository_Complete_InvalidCreatedOrder_ReturnError(t *testing.T) {
	tests := []struct {
		name          string
		order         usecase.CreatedOrder
		expectedError string
	}{
		{
			name:          "empty order id",
			order:         usecase.CreatedOrder{CheckoutURL: "/checkout/1"},
			expectedError: "complete activation: empty order id",
		},
		{
			name:          "empty checkout URL",
			order:         usecase.CreatedOrder{ID: uuid.New(), CheckoutURL: "  "},
			expectedError: "complete activation: empty checkout URL",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			beginner := &activationTransactionBeginnerStub{}
			repository := &ActivationRepository{transactions: beginner, newID: uuid.New}

			// when
			_, err := repository.Complete(context.Background(), uuid.New(), test.order, time.Now())

			// then
			require.EqualError(t, err, test.expectedError)
			require.Zero(t, beginner.calls)
		})
	}
}

func TestActivationRepository_Complete_CompletedOperation_ReturnReplay(t *testing.T) {
	// given
	operationID := uuid.New()
	expected := usecase.ActivationResult{
		TicketID:    uuid.New(),
		Status:      domain.TicketStatusRedeemed,
		OrderID:     uuid.New(),
		CheckoutURL: "/checkout/replayed",
	}
	responseBody, err := encodeActivationResult(expected)
	require.NoError(t, err)
	responseStatus := int32(activationResponseStatus)
	transaction := &activationTransactionStub{rows: []activationRow{activationOperationRow(
		activationOperationRecord{
			ID:             operationID,
			ActorID:        uuid.New(),
			TicketID:       expected.TicketID,
			State:          activationOperationStateComplete,
			ResponseStatus: &responseStatus,
			ResponseBody:   responseBody,
		},
	)}}
	repository := newActivationRepositoryForTest(transaction)

	// when
	result, err := repository.Complete(
		context.Background(),
		operationID,
		usecase.CreatedOrder{ID: uuid.New(), CheckoutURL: "/checkout/ignored"},
		time.Now(),
	)

	// then
	require.NoError(t, err)
	require.Equal(t, expected, result)
	require.Empty(t, transaction.execCalls)
	require.Equal(t, 1, transaction.commitCalls)
	require.Zero(t, transaction.rollbackCalls)
}

func TestActivationRepository_Complete_TicketStateChanged_ReturnNotActivatable(t *testing.T) {
	// given
	transaction := &activationTransactionStub{
		rows: []activationRow{activationOperationRow(activationOperationRecord{
			ID:       uuid.New(),
			ActorID:  uuid.New(),
			TicketID: uuid.New(),
			State:    activationOperationStateProcessing,
		})},
		execResults: []activationExecResult{{commandTag: pgconn.NewCommandTag("UPDATE 0")}},
	}
	repository := newActivationRepositoryForTest(transaction)

	// when
	_, err := repository.Complete(
		context.Background(),
		uuid.New(),
		usecase.CreatedOrder{ID: uuid.New(), CheckoutURL: "/checkout/1"},
		time.Now(),
	)

	// then
	require.ErrorIs(t, err, usecase.ErrTicketNotActivatable)
	require.Len(t, transaction.execCalls, 1)
	require.Zero(t, transaction.commitCalls)
	require.Equal(t, 1, transaction.rollbackCalls)
}

func TestActivationRepository_Complete_OutboxInsertFails_RollbackTransaction(t *testing.T) {
	// given
	insertError := errors.New("insert outbox failed")
	transaction := &activationTransactionStub{
		rows: []activationRow{activationOperationRow(activationOperationRecord{
			ID:       uuid.New(),
			ActorID:  uuid.New(),
			TicketID: uuid.New(),
			State:    activationOperationStateProcessing,
		})},
		execResults: []activationExecResult{
			{commandTag: pgconn.NewCommandTag("UPDATE 1")},
			{commandTag: pgconn.NewCommandTag("UPDATE 1")},
			{err: insertError},
		},
	}
	repository := newActivationRepositoryForTest(transaction)

	// when
	_, err := repository.Complete(
		context.Background(),
		uuid.New(),
		usecase.CreatedOrder{ID: uuid.New(), CheckoutURL: "/checkout/1"},
		time.Now(),
	)

	// then
	require.Error(t, err)
	require.ErrorIs(t, err, insertError)
	require.Equal(t, "insert activation event: insert outbox failed", err.Error())
	require.Zero(t, transaction.commitCalls)
	require.Equal(t, 1, transaction.rollbackCalls)
}

func TestActivationRepository_Prepare_CommitFails_RollbackTransaction(t *testing.T) {
	// given
	commitError := errors.New("commit failed")
	now := time.Now()
	transaction := &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			activationTicketRow(activationTicketRecord{
				ListingID:          uuid.New(),
				SKUID:              uuid.New(),
				Status:             domain.TicketStatusIssued,
				ActivationDeadline: now.Add(time.Minute),
			}),
			activationScanErrorRow(pgx.ErrNoRows),
			activationScanErrorRow(pgx.ErrNoRows),
		},
		execResults: []activationExecResult{{commandTag: pgconn.NewCommandTag("INSERT 0 1")}},
		commitErr:   commitError,
	}
	repository := newActivationRepositoryForTest(transaction)

	// when
	_, err := repository.Prepare(context.Background(), usecase.PrepareActivationCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            now,
	})

	// then
	require.Error(t, err)
	require.ErrorIs(t, err, commitError)
	require.Equal(t, "commit activation transaction: commit failed", err.Error())
	require.Equal(t, 1, transaction.commitCalls)
	require.Equal(t, 1, transaction.rollbackCalls)
}

func TestActivationRepository_Prepare_RequestCanceled_RollbackWithDetachedContext(t *testing.T) {
	// given
	queryError := errors.New("query failed")
	transaction := &activationTransactionStub{rows: []activationRow{activationScanErrorRow(queryError)}}
	repository := newActivationRepositoryForTest(transaction)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// when
	_, err := repository.Prepare(ctx, usecase.PrepareActivationCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            time.Now(),
	})

	// then
	require.ErrorIs(t, err, queryError)
	require.Equal(t, 1, transaction.rollbackCalls)
	require.NoError(t, transaction.rollbackCtxErr)
}

func activationScanErrorRow(err error) activationRow {
	return activationRowStub{scan: func([]any) error { return err }}
}

func activationOperationRow(record activationOperationRecord) activationRow {
	return activationRowStub{scan: func(destinations []any) error {
		*destinations[0].(*pgtype.UUID) = toPGUUID(record.ID)
		*destinations[1].(*pgtype.UUID) = toPGUUID(record.ActorID)
		*destinations[2].(*pgtype.UUID) = toPGUUID(record.TicketID)
		*destinations[3].(*string) = record.RequestHash
		*destinations[4].(*string) = record.State
		if record.ResponseStatus != nil {
			*destinations[5].(*pgtype.Int4) = pgtype.Int4{Int32: *record.ResponseStatus, Valid: true}
		}
		*destinations[6].(*[]byte) = record.ResponseBody

		return nil
	}}
}

func activationTicketRow(record activationTicketRecord) activationRow {
	return activationRowStub{scan: func(destinations []any) error {
		*destinations[0].(*pgtype.UUID) = toPGUUID(record.ListingID)
		*destinations[1].(*pgtype.UUID) = toPGUUID(record.SKUID)
		*destinations[2].(*string) = string(record.Status)
		*destinations[3].(*pgtype.Timestamptz) = toPGTimestamptz(record.ActivationDeadline)

		return nil
	}}
}

func activationUUIDRow(value uuid.UUID) activationRow {
	return activationRowStub{scan: func(destinations []any) error {
		*destinations[0].(*pgtype.UUID) = toPGUUID(value)

		return nil
	}}
}

func newActivationRepositoryForTest(transaction activationTransaction) *ActivationRepository {
	return &ActivationRepository{
		transactions: &activationTransactionBeginnerStub{transaction: transaction},
		newID:        uuid.New,
	}
}
