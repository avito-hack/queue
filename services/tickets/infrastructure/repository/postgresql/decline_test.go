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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

func TestDeclineRepository_Decline_IssuedTicket_CloseTicketAndWriteOutbox(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	queueEntryID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	idempotencyKey := uuid.New()
	operationID := uuid.New()
	eventID := uuid.New()
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	transaction := &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			declineTicketRow(declineTicketRecord{
				QueueEntryID:       queueEntryID,
				ListingID:          listingID,
				SKUID:              skuID,
				Status:             domain.TicketStatusIssued,
				ActivationDeadline: now.Add(time.Minute),
			}),
			activationScanErrorRow(pgx.ErrNoRows),
			declineBoolRow(false),
		},
		execResults: []activationExecResult{
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
			{commandTag: pgconn.NewCommandTag("UPDATE 1")},
			{commandTag: pgconn.NewCommandTag("UPDATE 1")},
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
		},
	}
	repository := newDeclineRepositoryForTest(transaction, operationID, eventID)
	command := usecase.DeclineTicketCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: idempotencyKey,
		Now:            now,
	}

	// when
	result, err := repository.Decline(context.Background(), command)

	// then
	require.NoError(t, err)
	assert.Equal(t, usecase.DeclineTicketResult{
		TicketID: ticketID,
		Status:   domain.TicketStatusClosed,
	}, result)
	require.Len(t, transaction.queryCalls, 4)
	assert.NotContains(t, transaction.queryCalls[0].query, "FOR UPDATE")
	assert.Equal(t, []any{
		toPGUUID(userID),
		declineOperation,
		toPGUUID(idempotencyKey),
	}, transaction.queryCalls[0].args)
	assert.Contains(t, transaction.queryCalls[1].query, "WHERE id = $1 AND user_id = $2")
	assert.Contains(t, transaction.queryCalls[1].query, "FOR UPDATE")
	assert.Equal(t, []any{toPGUUID(ticketID), toPGUUID(userID)}, transaction.queryCalls[1].args)
	assert.NotContains(t, transaction.queryCalls[2].query, "FOR UPDATE")
	assert.Contains(t, transaction.queryCalls[3].query, "SELECT EXISTS")
	assert.NotContains(t, transaction.queryCalls[3].query, "FOR UPDATE")
	assert.Equal(t, []any{
		toPGUUID(ticketID),
		activationOperation,
		activationOperationStateProcessing,
	}, transaction.queryCalls[3].args)
	require.Len(t, transaction.execCalls, 4)
	assert.Contains(t, transaction.execCalls[0].query, "ON CONFLICT")
	assert.Equal(t, []any{
		toPGUUID(operationID),
		toPGUUID(idempotencyKey),
		declineOperation,
		toPGUUID(userID),
		toPGUUID(ticketID),
		declineRequestHash(userID, ticketID),
		declineOperationStateProcessing,
		now.Add(declineOperationTTL),
		now,
	}, transaction.execCalls[0].args)
	assert.Contains(t, transaction.execCalls[1].query, "status = 'closed'")
	assert.Contains(t, transaction.execCalls[1].query, "close_reason = 'user_declined'")
	assert.Contains(t, transaction.execCalls[1].query, "activation_deadline > $3")
	assert.Equal(t, []any{toPGUUID(ticketID), toPGUUID(userID), now}, transaction.execCalls[1].args)
	assert.Contains(t, transaction.execCalls[2].query, "state = 'completed'")
	assert.Equal(t, toPGUUID(operationID), transaction.execCalls[2].args[0])
	assert.Equal(t, declineResponseStatus, transaction.execCalls[2].args[1])
	assert.JSONEq(t, `{
		"ticket_id":"`+ticketID.String()+`",
		"status":"closed"
	}`, string(transaction.execCalls[2].args[2].([]byte)))
	assert.Equal(t, now, transaction.execCalls[2].args[3])
	assert.Equal(t, toPGUUID(eventID), transaction.execCalls[3].args[0])
	assert.Equal(t, declineOutboxAggregateType, transaction.execCalls[3].args[1])
	assert.Equal(t, toPGUUID(ticketID), transaction.execCalls[3].args[2])
	assert.Equal(t, domain.TicketEventClosed, transaction.execCalls[3].args[3])
	assert.Equal(t, declineOutboxState, transaction.execCalls[3].args[5])
	assert.Equal(t, now, transaction.execCalls[3].args[6])
	assert.JSONEq(t, `{
		"ticket_id":"`+ticketID.String()+`",
		"queue_entry_id":"`+queueEntryID.String()+`",
		"listing_id":"`+listingID.String()+`",
		"sku_id":"`+skuID.String()+`",
		"user_id":"`+userID.String()+`",
		"status":"closed",
		"close_reason":"user_declined",
		"finished_at":"2026-08-07T12:00:00Z"
	}`, string(transaction.execCalls[3].args[4].([]byte)))
	assert.Equal(t, 1, transaction.commitCalls)
	assert.Zero(t, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_CompletedOperation_ReturnReplay(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	expected := usecase.DeclineTicketResult{TicketID: ticketID, Status: domain.TicketStatusClosed}
	responseBody, err := encodeDeclineResult(expected)
	require.NoError(t, err)
	responseStatus := int32(declineResponseStatus)
	transaction := &activationTransactionStub{rows: []activationRow{declineOperationRow(
		declineOperationRecord{
			ID:             uuid.New(),
			ActorID:        userID,
			TicketID:       ticketID,
			RequestHash:    declineRequestHash(userID, ticketID),
			State:          declineOperationStateComplete,
			ResponseStatus: &responseStatus,
			ResponseBody:   responseBody,
		},
	)}}
	repository := newDeclineRepositoryForTest(transaction)

	// when
	result, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: uuid.New(),
		Now:            time.Now(),
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, expected, result)
	require.Len(t, transaction.queryCalls, 1)
	assert.NotContains(t, transaction.queryCalls[0].query, "FOR UPDATE")
	assert.Empty(t, transaction.execCalls)
	assert.Equal(t, 1, transaction.commitCalls)
	assert.Zero(t, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_ProcessingOperation_ReturnInvariantError(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	transaction := &activationTransactionStub{rows: []activationRow{declineOperationRow(
		declineOperationRecord{
			ID:          uuid.New(),
			ActorID:     userID,
			TicketID:    ticketID,
			RequestHash: declineRequestHash(userID, ticketID),
			State:       declineOperationStateProcessing,
		},
	)}}
	repository := newDeclineRepositoryForTest(transaction)

	// when
	_, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: uuid.New(),
		Now:            time.Now(),
	})

	// then
	assert.EqualError(t, err, "decline operation is unexpectedly processing")
	assert.Empty(t, transaction.execCalls)
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_CompletedOperationWithInvalidResponse_ReturnError(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	transaction := &activationTransactionStub{rows: []activationRow{declineOperationRow(
		declineOperationRecord{
			ID:          uuid.New(),
			ActorID:     userID,
			TicketID:    ticketID,
			RequestHash: declineRequestHash(userID, ticketID),
			State:       declineOperationStateComplete,
		},
	)}}
	repository := newDeclineRepositoryForTest(transaction)

	// when
	_, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: uuid.New(),
		Now:            time.Now(),
	})

	// then
	assert.EqualError(t, err, "decline operation has invalid response status")
	assert.Empty(t, transaction.execCalls)
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_ConcurrentSameKeyCompletedAfterTicketLock_ReturnReplay(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	expected := usecase.DeclineTicketResult{TicketID: ticketID, Status: domain.TicketStatusClosed}
	responseBody, err := encodeDeclineResult(expected)
	require.NoError(t, err)
	responseStatus := int32(declineResponseStatus)
	transaction := &activationTransactionStub{rows: []activationRow{
		activationScanErrorRow(pgx.ErrNoRows),
		declineTicketRow(declineTicketRecord{
			QueueEntryID:       uuid.New(),
			ListingID:          uuid.New(),
			SKUID:              uuid.New(),
			Status:             domain.TicketStatusClosed,
			ActivationDeadline: time.Now().Add(-time.Minute),
		}),
		declineOperationRow(declineOperationRecord{
			ID:             uuid.New(),
			ActorID:        userID,
			TicketID:       ticketID,
			RequestHash:    declineRequestHash(userID, ticketID),
			State:          declineOperationStateComplete,
			ResponseStatus: &responseStatus,
			ResponseBody:   responseBody,
		}),
	}}
	repository := newDeclineRepositoryForTest(transaction)

	// when
	result, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: uuid.New(),
		Now:            time.Now(),
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, expected, result)
	require.Len(t, transaction.queryCalls, 3)
	assert.NotContains(t, transaction.queryCalls[0].query, "FOR UPDATE")
	assert.Contains(t, transaction.queryCalls[1].query, "FOR UPDATE")
	assert.NotContains(t, transaction.queryCalls[2].query, "FOR UPDATE")
	assert.Empty(t, transaction.execCalls)
	assert.Equal(t, 1, transaction.commitCalls)
	assert.Zero(t, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_ConcurrentSameKeyInsertedBeforeInsert_ReturnReplay(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	now := time.Now()
	expected := usecase.DeclineTicketResult{TicketID: ticketID, Status: domain.TicketStatusClosed}
	responseBody, err := encodeDeclineResult(expected)
	require.NoError(t, err)
	responseStatus := int32(declineResponseStatus)
	transaction := declineReadyTransaction(now)
	transaction.rows = append(transaction.rows, declineOperationRow(declineOperationRecord{
		ID:             uuid.New(),
		ActorID:        userID,
		TicketID:       ticketID,
		RequestHash:    declineRequestHash(userID, ticketID),
		State:          declineOperationStateComplete,
		ResponseStatus: &responseStatus,
		ResponseBody:   responseBody,
	}))
	transaction.execResults = []activationExecResult{
		{commandTag: pgconn.NewCommandTag("INSERT 0 0")},
	}
	repository := newDeclineRepositoryForTest(transaction, uuid.New())

	// when
	result, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: uuid.New(),
		Now:            now,
	})

	// then
	require.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.Len(t, transaction.queryCalls, 5)
	assert.Len(t, transaction.execCalls, 1)
	assert.Equal(t, 1, transaction.commitCalls)
	assert.Zero(t, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_SameKeyForAnotherTicket_ReturnIdempotencyConflict(t *testing.T) {
	// given
	userID := uuid.New()
	operationTicketID := uuid.New()
	transaction := &activationTransactionStub{rows: []activationRow{declineOperationRow(
		declineOperationRecord{
			ID:          uuid.New(),
			ActorID:     userID,
			TicketID:    operationTicketID,
			RequestHash: declineRequestHash(userID, operationTicketID),
			State:       declineOperationStateProcessing,
		},
	)}}
	repository := newDeclineRepositoryForTest(transaction)

	// when
	_, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         userID,
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            time.Now(),
	})

	// then
	assert.ErrorIs(t, err, usecase.ErrIdempotencyConflict)
	assert.Empty(t, transaction.execCalls)
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_TicketNotFound_ReturnNotFound(t *testing.T) {
	// given
	transaction := &activationTransactionStub{rows: []activationRow{
		activationScanErrorRow(pgx.ErrNoRows),
		activationScanErrorRow(pgx.ErrNoRows),
	}}
	repository := newDeclineRepositoryForTest(transaction)

	// when
	_, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            time.Now(),
	})

	// then
	assert.ErrorIs(t, err, usecase.ErrTicketNotFound)
	assert.Empty(t, transaction.execCalls)
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_IneligibleTicket_ReturnNotDeclinable(t *testing.T) {
	now := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		status   domain.TicketStatus
		deadline time.Time
	}{
		{name: "redeemed", status: domain.TicketStatusRedeemed, deadline: now.Add(time.Minute)},
		{name: "closed", status: domain.TicketStatusClosed, deadline: now.Add(time.Minute)},
		{name: "deadline reached", status: domain.TicketStatusIssued, deadline: now},
		{name: "deadline passed", status: domain.TicketStatusIssued, deadline: now.Add(-time.Nanosecond)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			rows := []activationRow{
				activationScanErrorRow(pgx.ErrNoRows),
				declineTicketRow(declineTicketRecord{
					QueueEntryID:       uuid.New(),
					ListingID:          uuid.New(),
					SKUID:              uuid.New(),
					Status:             test.status,
					ActivationDeadline: test.deadline,
				}),
				activationScanErrorRow(pgx.ErrNoRows),
			}
			if test.status == domain.TicketStatusIssued {
				rows = append(rows, declineBoolRow(false))
			}
			transaction := &activationTransactionStub{rows: rows}
			repository := newDeclineRepositoryForTest(transaction)

			// when
			_, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
				UserID:         uuid.New(),
				TicketID:       uuid.New(),
				IdempotencyKey: uuid.New(),
				Now:            now,
			})

			// then
			assert.ErrorIs(t, err, usecase.ErrTicketNotDeclinable)
			assert.Empty(t, transaction.execCalls)
			assert.Zero(t, transaction.commitCalls)
			assert.Equal(t, 1, transaction.rollbackCalls)
		})
	}
}

func TestDeclineRepository_Decline_ActivationProcessing_ReturnActivationInProgress(t *testing.T) {
	// given
	now := time.Now()
	transaction := &activationTransactionStub{rows: []activationRow{
		activationScanErrorRow(pgx.ErrNoRows),
		declineTicketRow(declineTicketRecord{
			QueueEntryID:       uuid.New(),
			ListingID:          uuid.New(),
			SKUID:              uuid.New(),
			Status:             domain.TicketStatusIssued,
			ActivationDeadline: now,
		}),
		activationScanErrorRow(pgx.ErrNoRows),
		declineBoolRow(true),
	}}
	repository := newDeclineRepositoryForTest(transaction)

	// when
	_, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            now,
	})

	// then
	assert.ErrorIs(t, err, usecase.ErrActivationInProgress)
	require.Len(t, transaction.queryCalls, 4)
	assert.Contains(t, transaction.queryCalls[3].query, "SELECT EXISTS")
	assert.NotContains(t, transaction.queryCalls[3].query, "FOR UPDATE")
	assert.Empty(t, transaction.execCalls)
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_TicketOperationUniqueViolation_ReturnNotDeclinable(t *testing.T) {
	// given
	now := time.Now()
	transaction := declineReadyTransaction(now)
	transaction.execResults = []activationExecResult{{err: &pgconn.PgError{
		Code:           "23505",
		ConstraintName: declineTicketConstraint,
	}}}
	repository := newDeclineRepositoryForTest(transaction, uuid.New())

	// when
	_, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            now,
	})

	// then
	assert.ErrorIs(t, err, usecase.ErrTicketNotDeclinable)
	assert.Len(t, transaction.execCalls, 1)
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_TicketUpdateAffectsNoRows_ReturnNotDeclinable(t *testing.T) {
	// given
	now := time.Now()
	transaction := declineReadyTransaction(now)
	transaction.execResults = []activationExecResult{
		{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
		{commandTag: pgconn.NewCommandTag("UPDATE 0")},
	}
	repository := newDeclineRepositoryForTest(transaction, uuid.New())

	// when
	_, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            now,
	})

	// then
	assert.ErrorIs(t, err, usecase.ErrTicketNotDeclinable)
	assert.Len(t, transaction.execCalls, 2)
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_OutboxInsertFails_RollbackTransaction(t *testing.T) {
	// given
	now := time.Now()
	insertError := errors.New("insert outbox failed")
	transaction := declineReadyTransaction(now)
	transaction.execResults = []activationExecResult{
		{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
		{commandTag: pgconn.NewCommandTag("UPDATE 1")},
		{commandTag: pgconn.NewCommandTag("UPDATE 1")},
		{err: insertError},
	}
	repository := newDeclineRepositoryForTest(transaction, uuid.New(), uuid.New())

	// when
	_, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            now,
	})

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, insertError)
	assert.Equal(t, "insert decline event: insert outbox failed", err.Error())
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_CommitFails_RollbackTransaction(t *testing.T) {
	// given
	now := time.Now()
	commitError := errors.New("commit failed")
	transaction := declineReadyTransaction(now)
	transaction.execResults = []activationExecResult{
		{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
		{commandTag: pgconn.NewCommandTag("UPDATE 1")},
		{commandTag: pgconn.NewCommandTag("UPDATE 1")},
		{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
	}
	transaction.commitErr = commitError
	repository := newDeclineRepositoryForTest(transaction, uuid.New(), uuid.New())

	// when
	_, err := repository.Decline(context.Background(), usecase.DeclineTicketCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            now,
	})

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, commitError)
	assert.Equal(t, "commit decline transaction: commit failed", err.Error())
	assert.Equal(t, 1, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestDeclineRepository_Decline_RequestCanceled_RollbackWithDetachedContext(t *testing.T) {
	// given
	queryError := errors.New("query failed")
	transaction := &activationTransactionStub{rows: []activationRow{activationScanErrorRow(queryError)}}
	repository := newDeclineRepositoryForTest(transaction)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// when
	_, err := repository.Decline(ctx, usecase.DeclineTicketCommand{
		UserID:         uuid.New(),
		TicketID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Now:            time.Now(),
	})

	// then
	require.ErrorIs(t, err, queryError)
	assert.Equal(t, 1, transaction.rollbackCalls)
	assert.NoError(t, transaction.rollbackCtxErr)
}

func declineOperationRow(record declineOperationRecord) activationRow {
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

func declineTicketRow(record declineTicketRecord) activationRow {
	return activationRowStub{scan: func(destinations []any) error {
		*destinations[0].(*pgtype.UUID) = toPGUUID(record.QueueEntryID)
		*destinations[1].(*pgtype.UUID) = toPGUUID(record.ListingID)
		*destinations[2].(*pgtype.UUID) = toPGUUID(record.SKUID)
		*destinations[3].(*string) = string(record.Status)
		*destinations[4].(*time.Time) = record.ActivationDeadline

		return nil
	}}
}

func declineBoolRow(value bool) activationRow {
	return activationRowStub{scan: func(destinations []any) error {
		*destinations[0].(*bool) = value

		return nil
	}}
}

func declineReadyTransaction(now time.Time) *activationTransactionStub {
	return &activationTransactionStub{rows: []activationRow{
		activationScanErrorRow(pgx.ErrNoRows),
		declineTicketRow(declineTicketRecord{
			QueueEntryID:       uuid.New(),
			ListingID:          uuid.New(),
			SKUID:              uuid.New(),
			Status:             domain.TicketStatusIssued,
			ActivationDeadline: now.Add(time.Minute),
		}),
		activationScanErrorRow(pgx.ErrNoRows),
		declineBoolRow(false),
	}}
}

func newDeclineRepositoryForTest(
	transaction activationTransaction,
	ids ...uuid.UUID,
) *DeclineRepository {
	index := 0
	return &DeclineRepository{
		transactions: &activationTransactionBeginnerStub{transaction: transaction},
		newID: func() uuid.UUID {
			if index >= len(ids) {
				return uuid.New()
			}

			id := ids[index]
			index++

			return id
		},
	}
}
