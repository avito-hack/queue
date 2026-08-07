package postgresql

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
	activationOperation                = "activate_ticket"
	activationOperationStateProcessing = "processing"
	activationOperationStateComplete   = "completed"
	activationOperationStateFailed     = "failed"
	activationOperationTTL             = 24 * time.Hour
	activationResponseStatus           = 200
	activationOutboxAggregateType      = "ticket"
	activationOutboxEventType          = "ticket.redeemed"
	activationOutboxState              = "pending"
	activationTicketConstraint         = "uq_idempotency_operations_ticket_operation"
	activationRollbackTimeout          = 5 * time.Second
)

const findActivationOperationQuery = `SELECT
    id,
    actor_id,
    ticket_id,
    request_hash,
    state,
    response_status,
    response_body
FROM public.idempotency_operations
WHERE actor_id = $1 AND operation = $2 AND idempotency_key = $3
FOR UPDATE`

const getActivationOperationQuery = `SELECT
    id,
    actor_id,
    ticket_id,
    request_hash,
    state,
    response_status,
    response_body
FROM public.idempotency_operations
WHERE actor_id = $1 AND operation = $2 AND idempotency_key = $3`

const lockActivationOperationQuery = `SELECT
    id,
    actor_id,
    ticket_id,
    request_hash,
    state,
    response_status,
    response_body
FROM public.idempotency_operations
WHERE id = $1 AND operation = $2
FOR UPDATE`

const lockTicketForActivationQuery = `SELECT
    listing_id,
    sku_id,
    status,
    activation_deadline
FROM public.tickets
WHERE id = $1 AND user_id = $2
FOR UPDATE`

const findProcessingTicketActivationQuery = `SELECT id
FROM public.idempotency_operations
WHERE ticket_id = $1 AND operation = $2 AND state = $3`

const insertActivationOperationQuery = `INSERT INTO public.idempotency_operations (
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

const activateTicketQuery = `UPDATE public.tickets
SET status = 'redeemed',
    activated_at = $3,
    order_id = $4,
    checkout_url = $5,
    finished_at = $3,
    updated_at = $3,
    version = version + 1
WHERE id = $1 AND user_id = $2 AND status = 'issued'`

const completeActivationOperationQuery = `UPDATE public.idempotency_operations
SET state = 'completed',
    response_status = $2,
    response_body = $3,
    updated_at = $4
WHERE id = $1 AND state = 'processing'`

const failActivationOperationQuery = `UPDATE public.idempotency_operations
SET state = 'failed', updated_at = $2
WHERE id = $1 AND operation = 'activate_ticket' AND state = 'processing'`

const retryActivationOperationQuery = `UPDATE public.idempotency_operations
SET state = 'processing', updated_at = $2
WHERE id = $1 AND operation = 'activate_ticket' AND state = 'failed'`

const insertActivationOutboxQuery = `INSERT INTO public.outbox_events (
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

type activationRow interface {
	Scan(...any) error
}

type activationTransaction interface {
	QueryRow(context.Context, string, ...any) activationRow
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Commit(context.Context) error
	Rollback(context.Context) error
}

type activationTransactionBeginner interface {
	Begin(context.Context) (activationTransaction, error)
}

type pgxActivationTransaction struct {
	transaction pgx.Tx
}

func (t pgxActivationTransaction) QueryRow(ctx context.Context, query string, args ...any) activationRow {
	return t.transaction.QueryRow(ctx, query, args...)
}

func (t pgxActivationTransaction) Exec(
	ctx context.Context,
	query string,
	args ...any,
) (pgconn.CommandTag, error) {
	return t.transaction.Exec(ctx, query, args...)
}

func (t pgxActivationTransaction) Commit(ctx context.Context) error {
	return t.transaction.Commit(ctx)
}

func (t pgxActivationTransaction) Rollback(ctx context.Context) error {
	return t.transaction.Rollback(ctx)
}

type pgxActivationTransactionBeginner struct {
	pool *pgxpool.Pool
}

func (b pgxActivationTransactionBeginner) Begin(ctx context.Context) (activationTransaction, error) {
	transaction, err := b.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return pgxActivationTransaction{transaction: transaction}, nil
}

type ActivationRepository struct {
	transactions activationTransactionBeginner
	newID        func() uuid.UUID
}

func NewActivationRepository(pool *pgxpool.Pool) *ActivationRepository {
	return &ActivationRepository{
		transactions: pgxActivationTransactionBeginner{pool: pool},
		newID:        uuid.New,
	}
}

func (r *ActivationRepository) Prepare(
	ctx context.Context,
	command usecase.PrepareActivationCommand,
) (usecase.PreparedActivation, error) {
	return inActivationTransaction(ctx, r.transactions, func(
		transaction activationTransaction,
	) (usecase.PreparedActivation, error) {
		return r.prepare(ctx, transaction, command)
	})
}

func (r *ActivationRepository) Complete(
	ctx context.Context,
	operationID uuid.UUID,
	order usecase.CreatedOrder,
	completedAt time.Time,
) (usecase.ActivationResult, error) {
	if order.ID == uuid.Nil {
		return usecase.ActivationResult{}, fmt.Errorf("complete activation: empty order id")
	}
	if strings.TrimSpace(order.CheckoutURL) == "" {
		return usecase.ActivationResult{}, fmt.Errorf("complete activation: empty checkout URL")
	}

	return inActivationTransaction(ctx, r.transactions, func(
		transaction activationTransaction,
	) (usecase.ActivationResult, error) {
		return r.complete(ctx, transaction, operationID, order, completedAt)
	})
}

func (r *ActivationRepository) Fail(ctx context.Context, operationID uuid.UUID, failedAt time.Time) error {
	_, err := inActivationTransaction(ctx, r.transactions, func(
		transaction activationTransaction,
	) (struct{}, error) {
		commandTag, err := transaction.Exec(
			ctx,
			failActivationOperationQuery,
			toPGUUID(operationID),
			failedAt,
		)
		if err != nil {
			return struct{}{}, fmt.Errorf("fail activation operation: %w", err)
		}
		if commandTag.RowsAffected() != 1 {
			return struct{}{}, fmt.Errorf(
				"fail activation operation: unexpected affected rows %d",
				commandTag.RowsAffected(),
			)
		}

		return struct{}{}, nil
	})

	return err
}

func (r *ActivationRepository) prepare(
	ctx context.Context,
	transaction activationTransaction,
	command usecase.PrepareActivationCommand,
) (usecase.PreparedActivation, error) {
	requestHash := activationRequestHash(command.UserID, command.TicketID)
	operation, err := findActivationOperation(ctx, transaction, command.UserID, command.IdempotencyKey)
	if err == nil {
		return r.resolvePreparedActivation(ctx, transaction, operation, command, requestHash, nil)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return usecase.PreparedActivation{}, fmt.Errorf("find activation operation: %w", err)
	}

	ticket, err := lockTicketForActivation(ctx, transaction, command.UserID, command.TicketID)
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.PreparedActivation{}, usecase.ErrTicketNotFound
	}
	if err != nil {
		return usecase.PreparedActivation{}, fmt.Errorf("lock ticket for activation: %w", err)
	}
	operation, err = getActivationOperation(ctx, transaction, command.UserID, command.IdempotencyKey)
	if err == nil {
		return r.resolvePreparedActivation(ctx, transaction, operation, command, requestHash, &ticket)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return usecase.PreparedActivation{}, fmt.Errorf("recheck activation operation: %w", err)
	}
	if !ticket.Status.Valid() {
		return usecase.PreparedActivation{}, fmt.Errorf("lock ticket for activation: unknown status %q", ticket.Status)
	}
	if ticket.Status != domain.TicketStatusIssued {
		return usecase.PreparedActivation{}, usecase.ErrTicketNotActivatable
	}
	processingOperationExists, err := findProcessingTicketActivation(ctx, transaction, command.TicketID)
	if err != nil {
		return usecase.PreparedActivation{}, fmt.Errorf("find processing ticket activation: %w", err)
	}
	if processingOperationExists {
		return usecase.PreparedActivation{}, usecase.ErrActivationInProgress
	}
	if !command.Now.Before(ticket.ActivationDeadline) {
		return usecase.PreparedActivation{}, usecase.ErrTicketActivationExpired
	}

	operationID := r.newID()
	inserted, err := insertActivationOperation(ctx, transaction, operationID, command, requestHash)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == "23505" &&
			postgresError.ConstraintName == activationTicketConstraint {
			return usecase.PreparedActivation{}, usecase.ErrActivationInProgress
		}

		return usecase.PreparedActivation{}, fmt.Errorf("insert activation operation: %w", err)
	}
	if !inserted {
		operation, err = findActivationOperation(ctx, transaction, command.UserID, command.IdempotencyKey)
		if err != nil {
			return usecase.PreparedActivation{}, fmt.Errorf("find concurrent activation operation: %w", err)
		}

		return r.resolvePreparedActivation(ctx, transaction, operation, command, requestHash, &ticket)
	}

	return preparedActivation(operationID, command, ticket), nil
}

func (r *ActivationRepository) complete(
	ctx context.Context,
	transaction activationTransaction,
	operationID uuid.UUID,
	order usecase.CreatedOrder,
	completedAt time.Time,
) (usecase.ActivationResult, error) {
	operation, err := lockActivationOperation(ctx, transaction, operationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return usecase.ActivationResult{}, fmt.Errorf("activation operation not found")
	}
	if err != nil {
		return usecase.ActivationResult{}, fmt.Errorf("lock activation operation: %w", err)
	}

	switch operation.State {
	case activationOperationStateComplete:
		return decodeActivationResult(operation)
	case activationOperationStateProcessing:
	default:
		return usecase.ActivationResult{}, fmt.Errorf("activation operation has unknown state %q", operation.State)
	}

	result := usecase.ActivationResult{
		TicketID:    operation.TicketID,
		Status:      domain.TicketStatusRedeemed,
		OrderID:     order.ID,
		CheckoutURL: order.CheckoutURL,
	}
	responseBody, err := encodeActivationResult(result)
	if err != nil {
		return usecase.ActivationResult{}, fmt.Errorf("encode activation response: %w", err)
	}
	outboxPayload, err := encodeActivationOutbox(operation.ActorID, result, completedAt)
	if err != nil {
		return usecase.ActivationResult{}, fmt.Errorf("encode activation event: %w", err)
	}

	commandTag, err := transaction.Exec(
		ctx,
		activateTicketQuery,
		toPGUUID(operation.TicketID),
		toPGUUID(operation.ActorID),
		completedAt,
		toPGUUID(order.ID),
		order.CheckoutURL,
	)
	if err != nil {
		return usecase.ActivationResult{}, fmt.Errorf("activate ticket: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return usecase.ActivationResult{}, usecase.ErrTicketNotActivatable
	}

	commandTag, err = transaction.Exec(
		ctx,
		completeActivationOperationQuery,
		toPGUUID(operation.ID),
		activationResponseStatus,
		responseBody,
		completedAt,
	)
	if err != nil {
		return usecase.ActivationResult{}, fmt.Errorf("complete activation operation: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return usecase.ActivationResult{}, fmt.Errorf("complete activation operation: unexpected affected rows %d", commandTag.RowsAffected())
	}

	commandTag, err = transaction.Exec(
		ctx,
		insertActivationOutboxQuery,
		toPGUUID(r.newID()),
		activationOutboxAggregateType,
		toPGUUID(operation.TicketID),
		activationOutboxEventType,
		outboxPayload,
		activationOutboxState,
		completedAt,
	)
	if err != nil {
		return usecase.ActivationResult{}, fmt.Errorf("insert activation event: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return usecase.ActivationResult{}, fmt.Errorf("insert activation event: unexpected affected rows %d", commandTag.RowsAffected())
	}

	return result, nil
}

type activationOperationRecord struct {
	ID             uuid.UUID
	ActorID        uuid.UUID
	TicketID       uuid.UUID
	RequestHash    string
	State          string
	ResponseStatus *int32
	ResponseBody   []byte
}

type activationTicketRecord struct {
	ListingID          uuid.UUID
	SKUID              uuid.UUID
	Status             domain.TicketStatus
	ActivationDeadline time.Time
}

type activationResultJSON struct {
	TicketID    uuid.UUID           `json:"ticket_id"`
	Status      domain.TicketStatus `json:"status"`
	OrderID     uuid.UUID           `json:"order_id"`
	CheckoutURL string              `json:"checkout_url"`
}

type activationOutboxJSON struct {
	TicketID    uuid.UUID           `json:"ticket_id"`
	UserID      uuid.UUID           `json:"user_id"`
	OrderID     uuid.UUID           `json:"order_id"`
	CheckoutURL string              `json:"checkout_url"`
	Status      domain.TicketStatus `json:"status"`
	RedeemedAt  time.Time           `json:"redeemed_at"`
}

func findActivationOperation(
	ctx context.Context,
	transaction activationTransaction,
	userID uuid.UUID,
	idempotencyKey uuid.UUID,
) (activationOperationRecord, error) {
	return scanActivationOperation(transaction.QueryRow(
		ctx,
		findActivationOperationQuery,
		toPGUUID(userID),
		activationOperation,
		toPGUUID(idempotencyKey),
	))
}

func getActivationOperation(
	ctx context.Context,
	transaction activationTransaction,
	userID uuid.UUID,
	idempotencyKey uuid.UUID,
) (activationOperationRecord, error) {
	return scanActivationOperation(transaction.QueryRow(
		ctx,
		getActivationOperationQuery,
		toPGUUID(userID),
		activationOperation,
		toPGUUID(idempotencyKey),
	))
}

func lockActivationOperation(
	ctx context.Context,
	transaction activationTransaction,
	operationID uuid.UUID,
) (activationOperationRecord, error) {
	return scanActivationOperation(transaction.QueryRow(
		ctx,
		lockActivationOperationQuery,
		toPGUUID(operationID),
		activationOperation,
	))
}

func scanActivationOperation(row activationRow) (activationOperationRecord, error) {
	var record activationOperationRecord
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
		return activationOperationRecord{}, err
	}
	if !id.Valid || !actorID.Valid || !ticketID.Valid {
		return activationOperationRecord{}, fmt.Errorf("activation operation has null required UUID")
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

func lockTicketForActivation(
	ctx context.Context,
	transaction activationTransaction,
	userID uuid.UUID,
	ticketID uuid.UUID,
) (activationTicketRecord, error) {
	var record activationTicketRecord
	var listingID pgtype.UUID
	var skuID pgtype.UUID
	var status string

	err := transaction.QueryRow(
		ctx,
		lockTicketForActivationQuery,
		toPGUUID(ticketID),
		toPGUUID(userID),
	).Scan(&listingID, &skuID, &status, &record.ActivationDeadline)
	if err != nil {
		return activationTicketRecord{}, err
	}
	if !listingID.Valid || !skuID.Valid {
		return activationTicketRecord{}, fmt.Errorf("ticket has null required UUID")
	}

	record.ListingID = uuid.UUID(listingID.Bytes)
	record.SKUID = uuid.UUID(skuID.Bytes)
	record.Status = domain.TicketStatus(status)

	return record, nil
}

func insertActivationOperation(
	ctx context.Context,
	transaction activationTransaction,
	operationID uuid.UUID,
	command usecase.PrepareActivationCommand,
	requestHash string,
) (bool, error) {
	commandTag, err := transaction.Exec(
		ctx,
		insertActivationOperationQuery,
		toPGUUID(operationID),
		toPGUUID(command.IdempotencyKey),
		activationOperation,
		toPGUUID(command.UserID),
		toPGUUID(command.TicketID),
		requestHash,
		activationOperationStateProcessing,
		command.Now.Add(activationOperationTTL),
		command.Now,
	)
	if err != nil {
		return false, err
	}

	return commandTag.RowsAffected() == 1, nil
}

func findProcessingTicketActivation(
	ctx context.Context,
	transaction activationTransaction,
	ticketID uuid.UUID,
) (bool, error) {
	var operationID pgtype.UUID
	err := transaction.QueryRow(
		ctx,
		findProcessingTicketActivationQuery,
		toPGUUID(ticketID),
		activationOperation,
		activationOperationStateProcessing,
	).Scan(&operationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !operationID.Valid {
		return false, fmt.Errorf("processing activation operation has null id")
	}

	return true, nil
}

func (r *ActivationRepository) resolvePreparedActivation(
	ctx context.Context,
	transaction activationTransaction,
	operation activationOperationRecord,
	command usecase.PrepareActivationCommand,
	requestHash string,
	lockedTicket *activationTicketRecord,
) (usecase.PreparedActivation, error) {
	if operation.TicketID != command.TicketID || operation.RequestHash != requestHash {
		return usecase.PreparedActivation{}, usecase.ErrIdempotencyConflict
	}

	switch operation.State {
	case activationOperationStateComplete:
		result, err := decodeActivationResult(operation)
		if err != nil {
			return usecase.PreparedActivation{}, err
		}

		return usecase.PreparedActivation{Replay: &result}, nil
	case activationOperationStateProcessing:
		var ticket activationTicketRecord
		if lockedTicket == nil {
			var err error
			ticket, err = lockTicketForActivation(ctx, transaction, command.UserID, command.TicketID)
			if errors.Is(err, pgx.ErrNoRows) {
				return usecase.PreparedActivation{}, usecase.ErrTicketNotFound
			}
			if err != nil {
				return usecase.PreparedActivation{}, fmt.Errorf("lock ticket for activation retry: %w", err)
			}
		} else {
			ticket = *lockedTicket
		}
		if !ticket.Status.Valid() {
			return usecase.PreparedActivation{}, fmt.Errorf("lock ticket for activation retry: unknown status %q", ticket.Status)
		}
		if ticket.Status != domain.TicketStatusIssued {
			return usecase.PreparedActivation{}, usecase.ErrTicketNotActivatable
		}
		if !command.Now.Before(ticket.ActivationDeadline) {
			return usecase.PreparedActivation{}, usecase.ErrTicketActivationExpired
		}

		return preparedActivation(operation.ID, command, ticket), nil
	case activationOperationStateFailed:
		var ticket activationTicketRecord
		if lockedTicket == nil {
			var err error
			ticket, err = lockTicketForActivation(ctx, transaction, command.UserID, command.TicketID)
			if errors.Is(err, pgx.ErrNoRows) {
				return usecase.PreparedActivation{}, usecase.ErrTicketNotFound
			}
			if err != nil {
				return usecase.PreparedActivation{}, fmt.Errorf("lock ticket for activation retry: %w", err)
			}
		} else {
			ticket = *lockedTicket
		}
		if !ticket.Status.Valid() {
			return usecase.PreparedActivation{}, fmt.Errorf("lock ticket for activation retry: unknown status %q", ticket.Status)
		}
		if ticket.Status != domain.TicketStatusIssued {
			return usecase.PreparedActivation{}, usecase.ErrTicketNotActivatable
		}
		if !command.Now.Before(ticket.ActivationDeadline) {
			return usecase.PreparedActivation{}, usecase.ErrTicketActivationExpired
		}
		commandTag, err := transaction.Exec(
			ctx,
			retryActivationOperationQuery,
			toPGUUID(operation.ID),
			command.Now,
		)
		if err != nil {
			return usecase.PreparedActivation{}, fmt.Errorf("retry activation operation: %w", err)
		}
		if commandTag.RowsAffected() != 1 {
			return usecase.PreparedActivation{}, fmt.Errorf(
				"retry activation operation: unexpected affected rows %d",
				commandTag.RowsAffected(),
			)
		}

		return preparedActivation(operation.ID, command, ticket), nil
	default:
		return usecase.PreparedActivation{}, fmt.Errorf("activation operation has unknown state %q", operation.State)
	}
}

func preparedActivation(
	operationID uuid.UUID,
	command usecase.PrepareActivationCommand,
	ticket activationTicketRecord,
) usecase.PreparedActivation {
	return usecase.PreparedActivation{
		OperationID: operationID,
		Order: usecase.CreateOrderRequest{
			TicketID:       command.TicketID,
			ListingID:      ticket.ListingID,
			SKUID:          ticket.SKUID,
			UserID:         command.UserID,
			IdempotencyKey: operationID,
		},
	}
}

func activationRequestHash(userID, ticketID uuid.UUID) string {
	var input [32]byte
	copy(input[:16], userID[:])
	copy(input[16:], ticketID[:])
	hash := sha256.Sum256(input[:])

	return hex.EncodeToString(hash[:])
}

func encodeActivationResult(result usecase.ActivationResult) ([]byte, error) {
	return json.Marshal(activationResultJSON{
		TicketID:    result.TicketID,
		Status:      result.Status,
		OrderID:     result.OrderID,
		CheckoutURL: result.CheckoutURL,
	})
}

func decodeActivationResult(operation activationOperationRecord) (usecase.ActivationResult, error) {
	if operation.ResponseStatus == nil || *operation.ResponseStatus != activationResponseStatus {
		return usecase.ActivationResult{}, fmt.Errorf("activation operation has invalid response status")
	}

	var response activationResultJSON
	if err := json.Unmarshal(operation.ResponseBody, &response); err != nil {
		return usecase.ActivationResult{}, fmt.Errorf("decode activation response: %w", err)
	}
	if response.TicketID != operation.TicketID ||
		response.Status != domain.TicketStatusRedeemed ||
		response.OrderID == uuid.Nil ||
		response.CheckoutURL == "" {
		return usecase.ActivationResult{}, fmt.Errorf("activation operation has invalid response")
	}

	return usecase.ActivationResult{
		TicketID:    response.TicketID,
		Status:      response.Status,
		OrderID:     response.OrderID,
		CheckoutURL: response.CheckoutURL,
	}, nil
}

func encodeActivationOutbox(
	userID uuid.UUID,
	result usecase.ActivationResult,
	activatedAt time.Time,
) ([]byte, error) {
	return json.Marshal(activationOutboxJSON{
		TicketID:    result.TicketID,
		UserID:      userID,
		OrderID:     result.OrderID,
		CheckoutURL: result.CheckoutURL,
		Status:      domain.TicketStatusRedeemed,
		RedeemedAt:  activatedAt,
	})
}

func inActivationTransaction[T any](
	ctx context.Context,
	transactions activationTransactionBeginner,
	action func(activationTransaction) (T, error),
) (result T, err error) {
	transaction, err := transactions.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin activation transaction: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		rollbackContext, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), activationRollbackTimeout)
		defer cancelRollback()
		rollbackErr := transaction.Rollback(rollbackContext)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}

		wrappedRollbackErr := fmt.Errorf("rollback activation transaction: %w", rollbackErr)
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
		return result, fmt.Errorf("commit activation transaction: %w", err)
	}
	committed = true

	return result, nil
}
