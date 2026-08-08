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

func TestIssueRepository_Issue_NewQueueEntry_CreateTicketAndOperation(t *testing.T) {
	// given
	command := issueCommandForTest()
	operationID := uuid.New()
	ticketID := uuid.New()
	transaction := &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			issueActiveTicketCountRow(1),
		},
		execResults: []activationExecResult{
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
			{commandTag: pgconn.NewCommandTag("UPDATE 1")},
		},
	}
	repository := newIssueRepositoryForTest(transaction, operationID, ticketID)

	// when
	result, err := repository.Issue(context.Background(), command)

	// then
	require.NoError(t, err)
	assert.Equal(t, usecase.IssueTicketResult{
		Ticket: domain.Ticket{
			ID:                 ticketID,
			ListingID:          command.ListingID,
			SKUID:              command.SKUID,
			Status:             domain.TicketStatusIssued,
			IssuedAt:           command.IssuedAt,
			ActivationDeadline: command.ActivationDeadline,
		},
		QueueEntryID: command.QueueEntryID,
		UserID:       command.UserID,
		Created:      true,
	}, result)
	require.Len(t, transaction.queryCalls, 2)
	assert.NotContains(t, transaction.queryCalls[0].query, "FOR UPDATE")
	assert.Equal(t, []any{
		toPGUUID(command.UserID),
		issueOperation,
		toPGUUID(command.IdempotencyKey),
	}, transaction.queryCalls[0].args)
	assert.Contains(t, transaction.queryCalls[1].query, "status = 'issued'")
	assert.Equal(t, []any{
		toPGUUID(command.ListingID),
		toPGTimestamptz(command.IssuedAt),
	}, transaction.queryCalls[1].args)
	require.Len(t, transaction.execCalls, 3)
	assert.Contains(t, transaction.execCalls[0].query, "ticket_id")
	assert.Contains(t, transaction.execCalls[0].query, "NULL")
	assert.Contains(t, transaction.execCalls[0].query, "ON CONFLICT (actor_id, operation, idempotency_key)")
	assert.Equal(t, []any{
		toPGUUID(operationID),
		toPGUUID(command.IdempotencyKey),
		issueOperation,
		toPGUUID(command.UserID),
		issueRequestHash(command),
		issueOperationStateProcessing,
		toPGTimestamptz(command.IssuedAt.Add(issueOperationTTL)),
		toPGTimestamptz(command.IssuedAt),
	}, transaction.execCalls[0].args)
	assert.Contains(t, transaction.execCalls[1].query, "status")
	assert.Contains(t, transaction.execCalls[1].query, "'issued'")
	assert.Contains(t, transaction.execCalls[1].query, "version")
	assert.Contains(t, transaction.execCalls[1].query, "ON CONFLICT (queue_entry_id) DO NOTHING")
	assert.NotContains(t, transaction.execCalls[1].query, "ON CONFLICT DO NOTHING")
	assert.Equal(t, []any{
		toPGUUID(ticketID),
		toPGUUID(command.QueueEntryID),
		toPGUUID(command.UserID),
		toPGUUID(command.ListingID),
		toPGUUID(command.SKUID),
		toPGTimestamptz(command.ActivationDeadline),
		toPGTimestamptz(command.IssuedAt),
	}, transaction.execCalls[1].args)
	assert.Contains(t, transaction.execCalls[2].query, "state = 'completed'")
	assert.Equal(t, toPGInt4(issueCreatedResponseStatus), transaction.execCalls[2].args[0])
	assert.Equal(t, toPGUUID(operationID), transaction.execCalls[2].args[3])
	assert.JSONEq(t, `{
		"id":"`+ticketID.String()+`",
		"queue_entry_id":"`+command.QueueEntryID.String()+`",
		"user_id":"`+command.UserID.String()+`",
		"listing_id":"`+command.ListingID.String()+`",
		"sku_id":"`+command.SKUID.String()+`",
		"status":"issued",
		"issued_at":"2026-08-07T12:00:00Z",
		"activation_deadline":"2026-08-07T12:15:00Z",
		"activated_at":null,
		"order_id":null,
		"checkout_url":null,
		"finished_at":null,
		"finish_reason":null
	}`, string(transaction.execCalls[2].args[1].([]byte)))
	assert.Equal(t, toPGTimestamptz(command.IssuedAt), transaction.execCalls[2].args[2])
	assert.Equal(t, 1, transaction.commitCalls)
	assert.Zero(t, transaction.rollbackCalls)
}

func TestIssueRepository_Issue_CompletedOperation_ReturnReplayAsExisting(t *testing.T) {
	// given
	command := issueCommandForTest()
	activatedAt := command.IssuedAt.Add(time.Minute)
	finishedAt := command.IssuedAt.Add(2 * time.Minute)
	orderID := uuid.New()
	checkoutURL := "/checkout/replayed"
	expected := usecase.IssueTicketResult{
		Ticket: domain.Ticket{
			ID:                 uuid.New(),
			ListingID:          command.ListingID,
			SKUID:              command.SKUID,
			Status:             domain.TicketStatusRedeemed,
			IssuedAt:           command.IssuedAt,
			ActivationDeadline: command.ActivationDeadline,
			ActivatedAt:        &activatedAt,
			OrderID:            &orderID,
			CheckoutURL:        &checkoutURL,
			FinishedAt:         &finishedAt,
		},
		QueueEntryID: command.QueueEntryID,
		UserID:       command.UserID,
	}
	responseBody, err := encodeIssueResult(expected)
	require.NoError(t, err)
	responseStatus := int32(issueCreatedResponseStatus)
	transaction := &activationTransactionStub{rows: []activationRow{issueOperationRow(issueOperationRecord{
		ID:             uuid.New(),
		ActorID:        command.UserID,
		RequestHash:    issueRequestHash(command),
		State:          issueOperationStateComplete,
		ResponseStatus: &responseStatus,
		ResponseBody:   responseBody,
	})}}
	repository := newIssueRepositoryForTest(transaction)

	// when
	result, err := repository.Issue(context.Background(), command)

	// then
	require.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.False(t, result.Created)
	assert.Empty(t, result.Ticket.AvailableActions)
	assert.Empty(t, transaction.execCalls)
	assert.Equal(t, 1, transaction.commitCalls)
	assert.Zero(t, transaction.rollbackCalls)
}

func TestIssueRepository_Issue_ListingCapacityExhausted_Rollback(t *testing.T) {
	// given
	command := issueCommandForTest()
	command.ListingQuantity = 1
	transaction := &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			issueActiveTicketCountRow(2),
		},
		execResults: []activationExecResult{
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
		},
	}
	repository := newIssueRepositoryForTest(transaction, uuid.New(), uuid.New())

	// when
	_, err := repository.Issue(context.Background(), command)

	// then
	require.ErrorIs(t, err, usecase.ErrTicketNotIssuable)
	assert.Equal(t, 2, len(transaction.execCalls))
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestIssueRepository_Issue_CountListingCapacityFailure_Rollback(t *testing.T) {
	// given
	command := issueCommandForTest()
	countError := errors.New("count failed")
	transaction := &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			activationScanErrorRow(countError),
		},
		execResults: []activationExecResult{
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
		},
	}
	repository := newIssueRepositoryForTest(transaction, uuid.New(), uuid.New())

	// when
	_, err := repository.Issue(context.Background(), command)

	// then
	require.EqualError(t, err, "count active listing tickets: count failed")
	assert.ErrorIs(t, err, countError)
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestIssueRepository_Issue_ConcurrentSameKey_ReturnCompletedReplay(t *testing.T) {
	// given
	command := issueCommandForTest()
	expected := newIssueResult(command, uuid.New(), true)
	responseBody, err := encodeIssueResult(expected)
	require.NoError(t, err)
	responseStatus := int32(issueCreatedResponseStatus)
	transaction := &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			issueOperationRow(issueOperationRecord{
				ID:             uuid.New(),
				ActorID:        command.UserID,
				RequestHash:    issueRequestHash(command),
				State:          issueOperationStateComplete,
				ResponseStatus: &responseStatus,
				ResponseBody:   responseBody,
			}),
		},
		execResults: []activationExecResult{{commandTag: pgconn.NewCommandTag("INSERT 0 0")}},
	}
	repository := newIssueRepositoryForTest(transaction, uuid.New())

	// when
	result, err := repository.Issue(context.Background(), command)

	// then
	require.NoError(t, err)
	expected.Created = false
	assert.Equal(t, expected, result)
	require.Len(t, transaction.queryCalls, 2)
	assert.NotContains(t, transaction.queryCalls[1].query, "FOR UPDATE")
	assert.Len(t, transaction.execCalls, 1)
	assert.Equal(t, 1, transaction.commitCalls)
	assert.Zero(t, transaction.rollbackCalls)
}

func TestIssueRepository_Issue_ExistingQueueEntry_ReturnExistingAndNoOutbox(t *testing.T) {
	// given
	command := issueCommandForTest()
	operationID := uuid.New()
	unusedTicketID := uuid.New()
	existing := newIssueResult(command, uuid.New(), false)
	activatedAt := command.IssuedAt.Add(time.Minute)
	finishedAt := command.IssuedAt.Add(10 * time.Minute)
	orderID := uuid.New()
	checkoutURL := "/checkout/existing"
	closeReason := domain.TicketCloseReasonSystemCancelled
	existing.Ticket.Status = domain.TicketStatusClosed
	existing.Ticket.ActivatedAt = &activatedAt
	existing.Ticket.OrderID = &orderID
	existing.Ticket.CheckoutURL = &checkoutURL
	existing.Ticket.FinishedAt = &finishedAt
	existing.Ticket.CloseReason = &closeReason
	transaction := &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			issuedTicketRow(existing),
		},
		execResults: []activationExecResult{
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
			{commandTag: pgconn.NewCommandTag("INSERT 0 0")},
			{commandTag: pgconn.NewCommandTag("UPDATE 1")},
		},
	}
	repository := newIssueRepositoryForTest(transaction, operationID, unusedTicketID)

	// when
	result, err := repository.Issue(context.Background(), command)

	// then
	require.NoError(t, err)
	assert.Equal(t, existing, result)
	require.Len(t, transaction.queryCalls, 2)
	assert.Equal(t, []any{toPGUUID(command.QueueEntryID)}, transaction.queryCalls[1].args)
	require.Len(t, transaction.execCalls, 3)
	assert.Equal(t, toPGInt4(issueExistingResponseStatus), transaction.execCalls[2].args[0])
	assert.NotContains(t, transaction.execCalls[2].query, "outbox")
	assert.Equal(t, 1, transaction.commitCalls)
	assert.Zero(t, transaction.rollbackCalls)
}

func TestIssueRepository_Issue_ExistingQueueEntryWithDifferentScope_ReturnNotIssuable(t *testing.T) {
	command := issueCommandForTest()
	tests := []struct {
		name   string
		mutate func(*usecase.IssueTicketResult)
	}{
		{name: "queue entry", mutate: func(result *usecase.IssueTicketResult) { result.QueueEntryID = uuid.New() }},
		{name: "user", mutate: func(result *usecase.IssueTicketResult) { result.UserID = uuid.New() }},
		{name: "listing", mutate: func(result *usecase.IssueTicketResult) { result.Ticket.ListingID = uuid.New() }},
		{name: "SKU", mutate: func(result *usecase.IssueTicketResult) { result.Ticket.SKUID = uuid.New() }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			existing := newIssueResult(command, uuid.New(), false)
			test.mutate(&existing)
			transaction := &activationTransactionStub{
				rows: []activationRow{
					activationScanErrorRow(pgx.ErrNoRows),
					issuedTicketRow(existing),
				},
				execResults: []activationExecResult{
					{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
					{commandTag: pgconn.NewCommandTag("INSERT 0 0")},
				},
			}
			repository := newIssueRepositoryForTest(transaction, uuid.New(), uuid.New())

			// when
			_, err := repository.Issue(context.Background(), command)

			// then
			assert.ErrorIs(t, err, usecase.ErrTicketNotIssuable)
			assert.Len(t, transaction.execCalls, 2)
			assert.Zero(t, transaction.commitCalls)
			assert.Equal(t, 1, transaction.rollbackCalls)
		})
	}
}

func TestIssueRepository_Issue_OperationScopeConflict_ReturnIdempotencyConflict(t *testing.T) {
	command := issueCommandForTest()
	tests := []struct {
		name        string
		actorID     uuid.UUID
		requestHash string
	}{
		{name: "actor", actorID: uuid.New(), requestHash: issueRequestHash(command)},
		{name: "request hash", actorID: command.UserID, requestHash: "different"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			transaction := &activationTransactionStub{rows: []activationRow{issueOperationRow(issueOperationRecord{
				ID:          uuid.New(),
				ActorID:     test.actorID,
				RequestHash: test.requestHash,
				State:       issueOperationStateComplete,
			})}}
			repository := newIssueRepositoryForTest(transaction)

			// when
			_, err := repository.Issue(context.Background(), command)

			// then
			assert.ErrorIs(t, err, usecase.ErrIdempotencyConflict)
			assert.Empty(t, transaction.execCalls)
			assert.Zero(t, transaction.commitCalls)
			assert.Equal(t, 1, transaction.rollbackCalls)
		})
	}
}

func TestIssueRepository_Issue_OperationInInvalidState_ReturnInvariantError(t *testing.T) {
	command := issueCommandForTest()
	tests := []struct {
		name          string
		state         string
		expectedError string
	}{
		{name: "processing", state: issueOperationStateProcessing, expectedError: "issue operation is unexpectedly processing"},
		{name: "unknown", state: "unknown", expectedError: `issue operation has unknown state "unknown"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			transaction := &activationTransactionStub{rows: []activationRow{issueOperationRow(issueOperationRecord{
				ID:          uuid.New(),
				ActorID:     command.UserID,
				RequestHash: issueRequestHash(command),
				State:       test.state,
			})}}
			repository := newIssueRepositoryForTest(transaction)

			// when
			_, err := repository.Issue(context.Background(), command)

			// then
			assert.EqualError(t, err, test.expectedError)
			assert.Zero(t, transaction.commitCalls)
			assert.Equal(t, 1, transaction.rollbackCalls)
		})
	}
}

func TestIssueRepository_Issue_CompletedOperationWithInvalidResponse_ReturnError(t *testing.T) {
	command := issueCommandForTest()
	validResult := newIssueResult(command, uuid.New(), false)
	validBody, err := encodeIssueResult(validResult)
	require.NoError(t, err)
	validStatus := int32(issueCreatedResponseStatus)
	invalidStatus := int32(500)
	wrongScope := validResult
	wrongScope.Ticket.SKUID = uuid.New()
	wrongScopeBody, err := encodeIssueResult(wrongScope)
	require.NoError(t, err)
	invalidSnapshot := validResult
	invalidSnapshot.Ticket.ID = uuid.Nil
	invalidSnapshotBody, err := encodeIssueResult(invalidSnapshot)
	require.NoError(t, err)
	tests := []struct {
		name          string
		status        *int32
		body          []byte
		expectedError string
	}{
		{name: "missing status", body: validBody, expectedError: "issue operation has invalid response status"},
		{name: "unexpected status", status: &invalidStatus, body: validBody, expectedError: "issue operation has invalid response status"},
		{name: "malformed JSON", status: &validStatus, body: []byte("{"), expectedError: "decode issue response:"},
		{name: "invalid snapshot", status: &validStatus, body: invalidSnapshotBody, expectedError: "issue operation has invalid response"},
		{name: "wrong scope", status: &validStatus, body: wrongScopeBody, expectedError: "issue operation has invalid response scope"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			transaction := &activationTransactionStub{rows: []activationRow{issueOperationRow(issueOperationRecord{
				ID:             uuid.New(),
				ActorID:        command.UserID,
				RequestHash:    issueRequestHash(command),
				State:          issueOperationStateComplete,
				ResponseStatus: test.status,
				ResponseBody:   test.body,
			})}}
			repository := newIssueRepositoryForTest(transaction)

			// when
			_, err := repository.Issue(context.Background(), command)

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.expectedError)
			assert.Zero(t, transaction.commitCalls)
			assert.Equal(t, 1, transaction.rollbackCalls)
		})
	}
}

func TestIssueRepository_Issue_OperationWithNullRequiredUUID_ReturnError(t *testing.T) {
	// given
	transaction := &activationTransactionStub{rows: []activationRow{activationRowStub{scan: func(destinations []any) error {
		*destinations[0].(*pgtype.UUID) = pgtype.UUID{}
		*destinations[1].(*pgtype.UUID) = toPGUUID(uuid.New())

		return nil
	}}}}
	repository := newIssueRepositoryForTest(transaction)

	// when
	_, err := repository.Issue(context.Background(), issueCommandForTest())

	// then
	assert.EqualError(t, err, "find issue operation: issue operation has null required UUID")
	assert.Zero(t, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestIssueRepository_Issue_DatabasePathError_Rollback(t *testing.T) {
	command := issueCommandForTest()
	testError := errors.New("database failed")
	tests := []struct {
		name          string
		transaction   func() *activationTransactionStub
		ids           []uuid.UUID
		expectedError string
	}{
		{
			name: "find operation",
			transaction: func() *activationTransactionStub {
				return &activationTransactionStub{rows: []activationRow{activationScanErrorRow(testError)}}
			},
			expectedError: "find issue operation: database failed",
		},
		{
			name: "insert operation",
			transaction: func() *activationTransactionStub {
				return &activationTransactionStub{
					rows:        []activationRow{activationScanErrorRow(pgx.ErrNoRows)},
					execResults: []activationExecResult{{err: testError}},
				}
			},
			ids:           []uuid.UUID{uuid.New()},
			expectedError: "insert issue operation: database failed",
		},
		{
			name: "find concurrent operation",
			transaction: func() *activationTransactionStub {
				return &activationTransactionStub{
					rows: []activationRow{
						activationScanErrorRow(pgx.ErrNoRows),
						activationScanErrorRow(testError),
					},
					execResults: []activationExecResult{{commandTag: pgconn.NewCommandTag("INSERT 0 0")}},
				}
			},
			ids:           []uuid.UUID{uuid.New()},
			expectedError: "find concurrent issue operation: database failed",
		},
		{
			name: "insert ticket",
			transaction: func() *activationTransactionStub {
				return &activationTransactionStub{
					rows: []activationRow{activationScanErrorRow(pgx.ErrNoRows)},
					execResults: []activationExecResult{
						{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
						{err: testError},
					},
				}
			},
			ids:           []uuid.UUID{uuid.New(), uuid.New()},
			expectedError: "insert issued ticket: database failed",
		},
		{
			name: "existing ticket disappeared",
			transaction: func() *activationTransactionStub {
				return issueExistingPathTransaction(activationScanErrorRow(pgx.ErrNoRows))
			},
			ids:           []uuid.UUID{uuid.New(), uuid.New()},
			expectedError: "issued ticket conflict has no existing ticket",
		},
		{
			name: "find existing ticket",
			transaction: func() *activationTransactionStub {
				return issueExistingPathTransaction(activationScanErrorRow(testError))
			},
			ids:           []uuid.UUID{uuid.New(), uuid.New()},
			expectedError: "find existing issued ticket: database failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			transaction := test.transaction()
			repository := newIssueRepositoryForTest(transaction, test.ids...)

			// when
			_, err := repository.Issue(context.Background(), command)

			// then
			assert.EqualError(t, err, test.expectedError)
			assert.Zero(t, transaction.commitCalls)
			assert.Equal(t, 1, transaction.rollbackCalls)
		})
	}
}

func TestIssueRepository_Issue_CompleteOperationFailure_Rollback(t *testing.T) {
	command := issueCommandForTest()
	testError := errors.New("write failed")
	tests := []struct {
		name          string
		execResults   []activationExecResult
		expectedError string
	}{
		{
			name: "complete operation error",
			execResults: []activationExecResult{
				{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
				{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
				{err: testError},
			},
			expectedError: "complete issue operation: write failed",
		},
		{
			name: "complete operation no rows",
			execResults: []activationExecResult{
				{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
				{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
				{commandTag: pgconn.NewCommandTag("UPDATE 0")},
			},
			expectedError: "complete issue operation: unexpected affected rows 0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			transaction := &activationTransactionStub{
				rows: []activationRow{
					activationScanErrorRow(pgx.ErrNoRows),
					issueActiveTicketCountRow(1),
				},
				execResults: test.execResults,
			}
			repository := newIssueRepositoryForTest(transaction, uuid.New(), uuid.New(), uuid.New())

			// when
			_, err := repository.Issue(context.Background(), command)

			// then
			assert.EqualError(t, err, test.expectedError)
			assert.Zero(t, transaction.commitCalls)
			assert.Equal(t, 1, transaction.rollbackCalls)
		})
	}
}

func TestIssueRepository_Issue_InvalidExistingTicketRow_ReturnError(t *testing.T) {
	command := issueCommandForTest()
	tests := []struct {
		name          string
		row           activationRow
		expectedError string
	}{
		{
			name: "scan",
			row: activationRowStub{scan: func([]any) error {
				return errors.New("scan failed")
			}},
			expectedError: "find existing issued ticket: scan failed",
		},
		{
			name: "null required UUID",
			row: issueInvalidTicketRow(func(destinations []any) {
				*destinations[0].(*pgtype.UUID) = pgtype.UUID{}
			}),
			expectedError: "find existing issued ticket: issued ticket has null required UUID",
		},
		{
			name: "zero required UUID",
			row: issueInvalidTicketRow(func(destinations []any) {
				*destinations[0].(*pgtype.UUID) = toPGUUID(uuid.Nil)
			}),
			expectedError: "find existing issued ticket: issued ticket has invalid snapshot",
		},
		{
			name: "unknown status",
			row: issueInvalidTicketRow(func(destinations []any) {
				*destinations[5].(*string) = "unknown"
			}),
			expectedError: `find existing issued ticket: issued ticket has unknown status "unknown"`,
		},
		{
			name: "unknown close reason",
			row: issueInvalidTicketRow(func(destinations []any) {
				*destinations[12].(*pgtype.Text) = pgtype.Text{String: "unknown", Valid: true}
			}),
			expectedError: `find existing issued ticket: issued ticket has unknown close reason "unknown"`,
		},
		{
			name: "invalid deadline",
			row: issueInvalidTicketRow(func(destinations []any) {
				issuedAt := *destinations[6].(*pgtype.Timestamptz)
				*destinations[7].(*pgtype.Timestamptz) = issuedAt
			}),
			expectedError: "find existing issued ticket: issued ticket has invalid snapshot",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			transaction := issueExistingPathTransaction(test.row)
			repository := newIssueRepositoryForTest(transaction, uuid.New(), uuid.New())

			// when
			_, err := repository.Issue(context.Background(), command)

			// then
			assert.EqualError(t, err, test.expectedError)
			assert.Zero(t, transaction.commitCalls)
			assert.Equal(t, 1, transaction.rollbackCalls)
		})
	}
}

func TestIssueRepository_Issue_BeginFails_ReturnWrappedError(t *testing.T) {
	// given
	command := issueCommandForTest()
	beginError := errors.New("begin failed")
	beginner := &activationTransactionBeginnerStub{err: beginError}
	repository := &IssueRepository{transactions: beginner, newID: uuid.New}

	// when
	_, err := repository.Issue(context.Background(), command)

	// then
	assert.EqualError(t, err, "begin issue transaction: begin failed")
	assert.Equal(t, 1, beginner.calls)
}

func TestIssueRepository_Issue_CommitFails_RollbackTransaction(t *testing.T) {
	// given
	command := issueCommandForTest()
	commitError := errors.New("commit failed")
	transaction := issueSuccessfulTransaction()
	transaction.commitErr = commitError
	repository := newIssueRepositoryForTest(transaction, uuid.New(), uuid.New(), uuid.New())

	// when
	_, err := repository.Issue(context.Background(), command)

	// then
	assert.EqualError(t, err, "commit issue transaction: commit failed")
	assert.Equal(t, 1, transaction.commitCalls)
	assert.Equal(t, 1, transaction.rollbackCalls)
}

func TestIssueRepository_Issue_RollbackFails_ReturnOperationAndRollbackErrors(t *testing.T) {
	// given
	command := issueCommandForTest()
	queryError := errors.New("query failed")
	rollbackError := errors.New("rollback failed")
	transaction := &activationTransactionStub{
		rows:        []activationRow{activationScanErrorRow(queryError)},
		rollbackErr: rollbackError,
	}
	repository := newIssueRepositoryForTest(transaction)

	// when
	_, err := repository.Issue(context.Background(), command)

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, queryError)
	assert.ErrorIs(t, err, rollbackError)
}

func TestIssueRepository_Issue_RequestCanceled_RollbackWithDetachedContext(t *testing.T) {
	// given
	queryError := errors.New("query failed")
	transaction := &activationTransactionStub{rows: []activationRow{activationScanErrorRow(queryError)}}
	repository := newIssueRepositoryForTest(transaction)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// when
	_, err := repository.Issue(ctx, issueCommandForTest())

	// then
	require.ErrorIs(t, err, queryError)
	assert.Equal(t, 1, transaction.rollbackCalls)
	assert.NoError(t, transaction.rollbackCtxErr)
}

func TestIssueRequestHash_IdempotencyKeyAndTimesChanged_ReturnSameHash(t *testing.T) {
	// given
	command := issueCommandForTest()
	original := issueRequestHash(command)
	changedKeyAndTimes := command
	changedKeyAndTimes.IdempotencyKey = uuid.New()
	changedKeyAndTimes.IssuedAt = command.IssuedAt.Add(time.Hour)
	changedKeyAndTimes.ActivationDeadline = command.ActivationDeadline.Add(time.Hour)

	// when
	actual := issueRequestHash(changedKeyAndTimes)

	// then
	assert.Equal(t, original, actual)
}

func TestIssueRequestHash_ScopeUUIDChanged_ReturnDifferentHash(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*usecase.IssueTicketCommand)
	}{
		{name: "queue entry", mutate: func(value *usecase.IssueTicketCommand) { value.QueueEntryID = uuid.New() }},
		{name: "user", mutate: func(value *usecase.IssueTicketCommand) { value.UserID = uuid.New() }},
		{name: "listing", mutate: func(value *usecase.IssueTicketCommand) { value.ListingID = uuid.New() }},
		{name: "SKU", mutate: func(value *usecase.IssueTicketCommand) { value.SKUID = uuid.New() }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			command := issueCommandForTest()
			original := issueRequestHash(command)
			changed := command
			test.mutate(&changed)

			// when
			actual := issueRequestHash(changed)

			// then
			assert.NotEqual(t, original, actual)
		})
	}
}

func issueCommandForTest() usecase.IssueTicketCommand {
	issuedAt := time.Date(2026, time.August, 7, 12, 0, 0, 0, time.UTC)

	return usecase.IssueTicketCommand{
		QueueEntryID:       uuid.New(),
		UserID:             uuid.New(),
		ListingID:          uuid.New(),
		SKUID:              uuid.New(),
		ListingQuantity:    10,
		IdempotencyKey:     uuid.New(),
		IssuedAt:           issuedAt,
		ActivationDeadline: issuedAt.Add(15 * time.Minute),
	}
}

func issueActiveTicketCountRow(value int64) activationRow {
	return activationRowStub{scan: func(destinations []any) error {
		*destinations[0].(*int64) = value

		return nil
	}}
}

func newIssueResult(
	command usecase.IssueTicketCommand,
	ticketID uuid.UUID,
	created bool,
) usecase.IssueTicketResult {
	return usecase.IssueTicketResult{
		Ticket: domain.Ticket{
			ID:                 ticketID,
			ListingID:          command.ListingID,
			SKUID:              command.SKUID,
			Status:             domain.TicketStatusIssued,
			IssuedAt:           command.IssuedAt,
			ActivationDeadline: command.ActivationDeadline,
		},
		QueueEntryID: command.QueueEntryID,
		UserID:       command.UserID,
		Created:      created,
	}
}

func issueOperationRow(record issueOperationRecord) activationRow {
	return activationRowStub{scan: func(destinations []any) error {
		*destinations[0].(*pgtype.UUID) = toPGUUID(record.ID)
		*destinations[1].(*pgtype.UUID) = toPGUUID(record.ActorID)
		*destinations[2].(*string) = record.RequestHash
		*destinations[3].(*string) = record.State
		if record.ResponseStatus != nil {
			*destinations[4].(*pgtype.Int4) = pgtype.Int4{Int32: *record.ResponseStatus, Valid: true}
		}
		*destinations[5].(*[]byte) = record.ResponseBody

		return nil
	}}
}

func issuedTicketRow(result usecase.IssueTicketResult) activationRow {
	return activationRowStub{scan: func(destinations []any) error {
		setIssuedTicketDestinations(destinations, result)

		return nil
	}}
}

func issueInvalidTicketRow(mutate func([]any)) activationRow {
	command := issueCommandForTest()
	result := newIssueResult(command, uuid.New(), false)

	return activationRowStub{scan: func(destinations []any) error {
		setIssuedTicketDestinations(destinations, result)
		mutate(destinations)

		return nil
	}}
}

func setIssuedTicketDestinations(destinations []any, result usecase.IssueTicketResult) {
	*destinations[0].(*pgtype.UUID) = toPGUUID(result.Ticket.ID)
	*destinations[1].(*pgtype.UUID) = toPGUUID(result.QueueEntryID)
	*destinations[2].(*pgtype.UUID) = toPGUUID(result.UserID)
	*destinations[3].(*pgtype.UUID) = toPGUUID(result.Ticket.ListingID)
	*destinations[4].(*pgtype.UUID) = toPGUUID(result.Ticket.SKUID)
	*destinations[5].(*string) = string(result.Ticket.Status)
	*destinations[6].(*pgtype.Timestamptz) = toPGTimestamptz(result.Ticket.IssuedAt)
	*destinations[7].(*pgtype.Timestamptz) = toPGTimestamptz(result.Ticket.ActivationDeadline)
	if result.Ticket.ActivatedAt != nil {
		*destinations[8].(*pgtype.Timestamptz) = pgtype.Timestamptz{Time: *result.Ticket.ActivatedAt, Valid: true}
	}
	if result.Ticket.OrderID != nil {
		*destinations[9].(*pgtype.UUID) = toPGUUID(*result.Ticket.OrderID)
	}
	if result.Ticket.CheckoutURL != nil {
		*destinations[10].(*pgtype.Text) = pgtype.Text{String: *result.Ticket.CheckoutURL, Valid: true}
	}
	if result.Ticket.FinishedAt != nil {
		*destinations[11].(*pgtype.Timestamptz) = pgtype.Timestamptz{Time: *result.Ticket.FinishedAt, Valid: true}
	}
	if result.Ticket.CloseReason != nil {
		*destinations[12].(*pgtype.Text) = pgtype.Text{String: string(*result.Ticket.CloseReason), Valid: true}
	}
}

func issueExistingPathTransaction(row activationRow) *activationTransactionStub {
	return &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			row,
		},
		execResults: []activationExecResult{
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
			{commandTag: pgconn.NewCommandTag("INSERT 0 0")},
		},
	}
}

func issueSuccessfulTransaction() *activationTransactionStub {
	return &activationTransactionStub{
		rows: []activationRow{
			activationScanErrorRow(pgx.ErrNoRows),
			issueActiveTicketCountRow(1),
		},
		execResults: []activationExecResult{
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
			{commandTag: pgconn.NewCommandTag("UPDATE 1")},
			{commandTag: pgconn.NewCommandTag("INSERT 0 1")},
		},
	}
}

func newIssueRepositoryForTest(
	transaction activationTransaction,
	ids ...uuid.UUID,
) *IssueRepository {
	index := 0

	return &IssueRepository{
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
