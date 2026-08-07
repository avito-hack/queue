package postgresql

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

const (
	declineOperation                = "decline_ticket"
	declineOperationStateProcessing = "processing"
	declineOperationStateComplete   = "completed"
	declineOperationTTL             = 24 * time.Hour
	declineResponseStatus           = 200
	declineOutboxAggregateType      = "ticket"
	declineOutboxEventType          = "ticket.closed"
	declineOutboxState              = "pending"
	declineTicketConstraint         = "uq_idempotency_operations_ticket_operation"
	declineRollbackTimeout          = 5 * time.Second
)

const findDeclineOperationQuery = `SELECT
    id,
    actor_id,
    ticket_id,
    request_hash,
    state,
    response_status,
    response_body
FROM public.idempotency_operations
WHERE actor_id = $1 AND operation = $2 AND idempotency_key = $3`

const lockTicketForDeclineQuery = `SELECT
    queue_entry_id,
    listing_id,
    sku_id,
    status,
    activation_deadline
FROM public.tickets
WHERE id = $1 AND user_id = $2
FOR UPDATE`

const hasProcessingActivationQuery = `SELECT EXISTS (
    SELECT 1
    FROM public.idempotency_operations
    WHERE ticket_id = $1 AND operation = $2 AND state = $3
)`

const insertDeclineOperationQuery = `INSERT INTO public.idempotency_operations (
    id,
    idempotency_key,
    operation,
    actor_id,
    ticket_id,
    request_hash,
    state,
    response_status,
    response_body,
    expires_at,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, NULL, $8, $9)
ON CONFLICT (actor_id, operation, idempotency_key) DO NOTHING`

const declineTicketQuery = `UPDATE public.tickets
SET status = 'closed',
    close_reason = 'user_declined',
    finished_at = $3,
    updated_at = $3,
    version = version + 1
WHERE id = $1
  AND user_id = $2
  AND status = 'issued'
  AND activation_deadline > $3`

const completeDeclineOperationQuery = `UPDATE public.idempotency_operations
SET state = 'completed',
    response_status = $2,
    response_body = $3,
    updated_at = $4
WHERE id = $1 AND state = 'processing'`

const insertDeclineOutboxQuery = `INSERT INTO public.outbox_events (
    id,
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    status,
    attempts,
    available_at,
    published_at
) VALUES ($1, $2, $3, $4, $5, $6, 0, $7, NULL)`

type DeclineRepository struct {
	transactions activationTransactionBeginner
	newID        func() uuid.UUID
}

func NewDeclineRepository(pool *pgxpool.Pool) *DeclineRepository {
	return &DeclineRepository{
		transactions: pgxActivationTransactionBeginner{pool: pool},
		newID:        uuid.New,
	}
}

func (r *DeclineRepository) Decline(
	ctx context.Context,
	command usecase.DeclineTicketCommand,
) (usecase.DeclineTicketResult, error) {
	return inDeclineTransaction(ctx, r.transactions, func(
		transaction activationTransaction,
	) (usecase.DeclineTicketResult, error) {
		return r.decline(ctx, transaction, command)
	})
}

func (r *DeclineRepository) decline(
	ctx context.Context,
	transaction activationTransaction,
	command usecase.DeclineTicketCommand,
) (usecase.DeclineTicketResult, error) {
	requestHash := declineRequestHash(command.UserID, command.TicketID)
	operation, err := findDeclineOperation(ctx, transaction, command.UserID, command.IdempotencyKey)
	if err == nil {
		return resolveDeclineOperation(operation, command, requestHash)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return usecase.DeclineTicketResult{}, fmt.Errorf("find decline operation: %w", err)
	}

	ticket, err := lockTicketForDecline(ctx, transaction, command.UserID, command.TicketID)
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.DeclineTicketResult{}, usecase.ErrTicketNotFound
	}
	if err != nil {
		return usecase.DeclineTicketResult{}, fmt.Errorf("lock ticket for decline: %w", err)
	}

	operation, err = findDeclineOperation(ctx, transaction, command.UserID, command.IdempotencyKey)
	if err == nil {
		return resolveDeclineOperation(operation, command, requestHash)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return usecase.DeclineTicketResult{}, fmt.Errorf("recheck decline operation: %w", err)
	}

	if !ticket.Status.Valid() {
		return usecase.DeclineTicketResult{}, fmt.Errorf("lock ticket for decline: unknown status %q", ticket.Status)
	}
	if ticket.Status != domain.TicketStatusIssued {
		return usecase.DeclineTicketResult{}, usecase.ErrTicketNotDeclinable
	}

	activationInProgress, err := hasProcessingActivation(ctx, transaction, command.TicketID)
	if err != nil {
		return usecase.DeclineTicketResult{}, fmt.Errorf("find processing ticket activation: %w", err)
	}
	if activationInProgress {
		return usecase.DeclineTicketResult{}, usecase.ErrActivationInProgress
	}
	if !command.Now.Before(ticket.ActivationDeadline) {
		return usecase.DeclineTicketResult{}, usecase.ErrTicketNotDeclinable
	}

	operationID := r.newID()
	inserted, err := insertDeclineOperation(ctx, transaction, operationID, command, requestHash)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == "23505" &&
			postgresError.ConstraintName == declineTicketConstraint {
			return usecase.DeclineTicketResult{}, usecase.ErrTicketNotDeclinable
		}

		return usecase.DeclineTicketResult{}, fmt.Errorf("insert decline operation: %w", err)
	}
	if !inserted {
		operation, err = findDeclineOperation(ctx, transaction, command.UserID, command.IdempotencyKey)
		if err != nil {
			return usecase.DeclineTicketResult{}, fmt.Errorf("find concurrent decline operation: %w", err)
		}

		return resolveDeclineOperation(operation, command, requestHash)
	}

	result := usecase.DeclineTicketResult{
		TicketID: command.TicketID,
		Status:   domain.TicketStatusClosed,
	}
	responseBody, err := encodeDeclineResult(result)
	if err != nil {
		return usecase.DeclineTicketResult{}, fmt.Errorf("encode decline response: %w", err)
	}
	outboxPayload, err := encodeDeclineOutbox(command.UserID, ticket, command.TicketID, command.Now)
	if err != nil {
		return usecase.DeclineTicketResult{}, fmt.Errorf("encode decline event: %w", err)
	}

	commandTag, err := transaction.Exec(
		ctx,
		declineTicketQuery,
		toPGUUID(command.TicketID),
		toPGUUID(command.UserID),
		command.Now,
	)
	if err != nil {
		return usecase.DeclineTicketResult{}, fmt.Errorf("decline ticket: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return usecase.DeclineTicketResult{}, usecase.ErrTicketNotDeclinable
	}

	commandTag, err = transaction.Exec(
		ctx,
		completeDeclineOperationQuery,
		toPGUUID(operationID),
		declineResponseStatus,
		responseBody,
		command.Now,
	)
	if err != nil {
		return usecase.DeclineTicketResult{}, fmt.Errorf("complete decline operation: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return usecase.DeclineTicketResult{}, fmt.Errorf(
			"complete decline operation: unexpected affected rows %d",
			commandTag.RowsAffected(),
		)
	}

	commandTag, err = transaction.Exec(
		ctx,
		insertDeclineOutboxQuery,
		toPGUUID(r.newID()),
		declineOutboxAggregateType,
		toPGUUID(command.TicketID),
		declineOutboxEventType,
		outboxPayload,
		declineOutboxState,
		command.Now,
	)
	if err != nil {
		return usecase.DeclineTicketResult{}, fmt.Errorf("insert decline event: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return usecase.DeclineTicketResult{}, fmt.Errorf(
			"insert decline event: unexpected affected rows %d",
			commandTag.RowsAffected(),
		)
	}

	return result, nil
}

type declineOperationRecord struct {
	ID             uuid.UUID
	ActorID        uuid.UUID
	TicketID       uuid.UUID
	RequestHash    string
	State          string
	ResponseStatus *int32
	ResponseBody   []byte
}

type declineTicketRecord struct {
	QueueEntryID       uuid.UUID
	ListingID          uuid.UUID
	SKUID              uuid.UUID
	Status             domain.TicketStatus
	ActivationDeadline time.Time
}

type declineResultJSON struct {
	TicketID uuid.UUID           `json:"ticket_id"`
	Status   domain.TicketStatus `json:"status"`
}

type declineOutboxJSON struct {
	TicketID     uuid.UUID                `json:"ticket_id"`
	QueueEntryID uuid.UUID                `json:"queue_entry_id"`
	ListingID    uuid.UUID                `json:"listing_id"`
	SKUID        uuid.UUID                `json:"sku_id"`
	UserID       uuid.UUID                `json:"user_id"`
	Status       domain.TicketStatus      `json:"status"`
	CloseReason  domain.TicketCloseReason `json:"close_reason"`
	FinishedAt   time.Time                `json:"finished_at"`
}

func findDeclineOperation(
	ctx context.Context,
	transaction activationTransaction,
	userID uuid.UUID,
	idempotencyKey uuid.UUID,
) (declineOperationRecord, error) {
	return scanDeclineOperation(transaction.QueryRow(
		ctx,
		findDeclineOperationQuery,
		toPGUUID(userID),
		declineOperation,
		toPGUUID(idempotencyKey),
	))
}

func scanDeclineOperation(row activationRow) (declineOperationRecord, error) {
	var record declineOperationRecord
	var id pgtype.UUID
	var actorID pgtype.UUID
	var ticketID pgtype.UUID
	var responseStatus pgtype.Int4

	err := row.Scan(
		&id,
		&actorID,
		&ticketID,
		&record.RequestHash,
		&record.State,
		&responseStatus,
		&record.ResponseBody,
	)
	if err != nil {
		return declineOperationRecord{}, err
	}
	if !id.Valid || !actorID.Valid || !ticketID.Valid {
		return declineOperationRecord{}, fmt.Errorf("decline operation has null required UUID")
	}

	record.ID = uuid.UUID(id.Bytes)
	record.ActorID = uuid.UUID(actorID.Bytes)
	record.TicketID = uuid.UUID(ticketID.Bytes)
	if responseStatus.Valid {
		value := responseStatus.Int32
		record.ResponseStatus = &value
	}

	return record, nil
}

func lockTicketForDecline(
	ctx context.Context,
	transaction activationTransaction,
	userID uuid.UUID,
	ticketID uuid.UUID,
) (declineTicketRecord, error) {
	var record declineTicketRecord
	var queueEntryID pgtype.UUID
	var listingID pgtype.UUID
	var skuID pgtype.UUID
	var status string

	err := transaction.QueryRow(
		ctx,
		lockTicketForDeclineQuery,
		toPGUUID(ticketID),
		toPGUUID(userID),
	).Scan(
		&queueEntryID,
		&listingID,
		&skuID,
		&status,
		&record.ActivationDeadline,
	)
	if err != nil {
		return declineTicketRecord{}, err
	}
	if !queueEntryID.Valid || !listingID.Valid || !skuID.Valid {
		return declineTicketRecord{}, fmt.Errorf("ticket has null required UUID")
	}

	record.QueueEntryID = uuid.UUID(queueEntryID.Bytes)
	record.ListingID = uuid.UUID(listingID.Bytes)
	record.SKUID = uuid.UUID(skuID.Bytes)
	record.Status = domain.TicketStatus(status)

	return record, nil
}

func hasProcessingActivation(
	ctx context.Context,
	transaction activationTransaction,
	ticketID uuid.UUID,
) (bool, error) {
	var exists bool
	err := transaction.QueryRow(
		ctx,
		hasProcessingActivationQuery,
		toPGUUID(ticketID),
		activationOperation,
		activationOperationStateProcessing,
	).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func insertDeclineOperation(
	ctx context.Context,
	transaction activationTransaction,
	operationID uuid.UUID,
	command usecase.DeclineTicketCommand,
	requestHash string,
) (bool, error) {
	commandTag, err := transaction.Exec(
		ctx,
		insertDeclineOperationQuery,
		toPGUUID(operationID),
		toPGUUID(command.IdempotencyKey),
		declineOperation,
		toPGUUID(command.UserID),
		toPGUUID(command.TicketID),
		requestHash,
		declineOperationStateProcessing,
		command.Now.Add(declineOperationTTL),
		command.Now,
	)
	if err != nil {
		return false, err
	}

	return commandTag.RowsAffected() == 1, nil
}

func resolveDeclineOperation(
	operation declineOperationRecord,
	command usecase.DeclineTicketCommand,
	requestHash string,
) (usecase.DeclineTicketResult, error) {
	if operation.ActorID != command.UserID ||
		operation.TicketID != command.TicketID ||
		operation.RequestHash != requestHash {
		return usecase.DeclineTicketResult{}, usecase.ErrIdempotencyConflict
	}

	switch operation.State {
	case declineOperationStateComplete:
		return decodeDeclineResult(operation)
	case declineOperationStateProcessing:
		return usecase.DeclineTicketResult{}, errors.New("decline operation is unexpectedly processing")
	default:
		return usecase.DeclineTicketResult{}, fmt.Errorf("decline operation has unknown state %q", operation.State)
	}
}

func declineRequestHash(userID, ticketID uuid.UUID) string {
	var input [32]byte
	copy(input[:16], userID[:])
	copy(input[16:], ticketID[:])
	hash := sha256.Sum256(input[:])

	return hex.EncodeToString(hash[:])
}

func encodeDeclineResult(result usecase.DeclineTicketResult) ([]byte, error) {
	return json.Marshal(declineResultJSON{
		TicketID: result.TicketID,
		Status:   result.Status,
	})
}

func decodeDeclineResult(operation declineOperationRecord) (usecase.DeclineTicketResult, error) {
	if operation.ResponseStatus == nil || *operation.ResponseStatus != declineResponseStatus {
		return usecase.DeclineTicketResult{}, fmt.Errorf("decline operation has invalid response status")
	}

	var response declineResultJSON
	if err := json.Unmarshal(operation.ResponseBody, &response); err != nil {
		return usecase.DeclineTicketResult{}, fmt.Errorf("decode decline response: %w", err)
	}
	if response.TicketID != operation.TicketID || response.Status != domain.TicketStatusClosed {
		return usecase.DeclineTicketResult{}, fmt.Errorf("decline operation has invalid response")
	}

	return usecase.DeclineTicketResult{
		TicketID: response.TicketID,
		Status:   response.Status,
	}, nil
}

func encodeDeclineOutbox(
	userID uuid.UUID,
	ticket declineTicketRecord,
	ticketID uuid.UUID,
	declinedAt time.Time,
) ([]byte, error) {
	return json.Marshal(declineOutboxJSON{
		TicketID:     ticketID,
		QueueEntryID: ticket.QueueEntryID,
		ListingID:    ticket.ListingID,
		SKUID:        ticket.SKUID,
		UserID:       userID,
		Status:       domain.TicketStatusClosed,
		CloseReason:  domain.TicketCloseReasonUserDeclined,
		FinishedAt:   declinedAt,
	})
}

func inDeclineTransaction[T any](
	ctx context.Context,
	transactions activationTransactionBeginner,
	action func(activationTransaction) (T, error),
) (result T, err error) {
	transaction, err := transactions.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin decline transaction: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		rollbackContext, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), declineRollbackTimeout)
		defer cancelRollback()
		rollbackErr := transaction.Rollback(rollbackContext)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}

		wrappedRollbackErr := fmt.Errorf("rollback decline transaction: %w", rollbackErr)
		if err == nil {
			err = wrappedRollbackErr
			return
		}
		err = errors.Join(err, wrappedRollbackErr)
	}()

	result, err = action(transaction)
	if err != nil {
		return result, err
	}
	if err = transaction.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit decline transaction: %w", err)
	}
	committed = true

	return result, nil
}
