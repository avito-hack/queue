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
	issueOperation                = "issue_ticket"
	issueOperationStateProcessing = "processing"
	issueOperationStateComplete   = "completed"
	issueOperationTTL             = 24 * time.Hour
	issueCreatedResponseStatus    = 201
	issueExistingResponseStatus   = 200
	issueRollbackTimeout          = 5 * time.Second
	issueLiveTicketConstraint     = "uq_tickets_live_user_listing"
)

const findIssueOperationQuery = `SELECT
    id,
    actor_id,
    request_hash,
    state,
    response_status,
    response_body
FROM public.idempotency_operations
WHERE actor_id = $1 AND operation = $2 AND idempotency_key = $3`

const insertIssueOperationQuery = `INSERT INTO public.idempotency_operations (
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
) VALUES ($1, $2, $3, $4, NULL, $5, $6, NULL, NULL, $7, $8)
ON CONFLICT (actor_id, operation, idempotency_key) DO NOTHING`

const insertIssuedTicketQuery = `INSERT INTO public.tickets (
    id,
    queue_entry_id,
    user_id,
    listing_id,
    sku_id,
    status,
    activation_deadline,
    activated_at,
    order_id,
    checkout_url,
    close_reason,
    issued_at,
    finished_at,
    updated_at,
    version
) VALUES ($1, $2, $3, $4, $5, 'issued', $6, NULL, NULL, NULL, NULL, $7, NULL, $7, 1)
ON CONFLICT (queue_entry_id) DO NOTHING`

const findTicketByQueueEntryQuery = `SELECT
    id,
    queue_entry_id,
    user_id,
    listing_id,
    sku_id,
    status,
    issued_at,
    activation_deadline,
    activated_at,
    order_id,
    checkout_url,
    finished_at,
    close_reason
FROM public.tickets
WHERE queue_entry_id = $1`

const completeIssueOperationQuery = `UPDATE public.idempotency_operations
SET state = 'completed',
    response_status = $2,
    response_body = $3,
    updated_at = $4
WHERE id = $1 AND state = 'processing'`

type IssueRepository struct {
	transactions activationTransactionBeginner
	newID        func() uuid.UUID
}

func NewIssueRepository(pool *pgxpool.Pool) *IssueRepository {
	return &IssueRepository{
		transactions: pgxActivationTransactionBeginner{pool: pool},
		newID:        uuid.New,
	}
}

func (r *IssueRepository) Issue(
	ctx context.Context,
	command usecase.IssueTicketCommand,
) (usecase.IssueTicketResult, error) {
	return inIssueTransaction(ctx, r.transactions, func(
		transaction activationTransaction,
	) (usecase.IssueTicketResult, error) {
		return r.issue(ctx, transaction, command)
	})
}

func (r *IssueRepository) issue(
	ctx context.Context,
	transaction activationTransaction,
	command usecase.IssueTicketCommand,
) (usecase.IssueTicketResult, error) {
	requestHash := issueRequestHash(command)
	operation, err := findIssueOperation(ctx, transaction, command.UserID, command.IdempotencyKey)
	if err == nil {
		return resolveIssueOperation(operation, command, requestHash)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return usecase.IssueTicketResult{}, fmt.Errorf("find issue operation: %w", err)
	}

	operationID := r.newID()
	inserted, err := insertIssueOperation(ctx, transaction, operationID, command, requestHash)
	if err != nil {
		return usecase.IssueTicketResult{}, fmt.Errorf("insert issue operation: %w", err)
	}
	if !inserted {
		operation, err = findIssueOperation(ctx, transaction, command.UserID, command.IdempotencyKey)
		if err != nil {
			return usecase.IssueTicketResult{}, fmt.Errorf("find concurrent issue operation: %w", err)
		}

		return resolveIssueOperation(operation, command, requestHash)
	}

	ticketID := r.newID()
	ticketInserted, err := insertIssuedTicket(ctx, transaction, ticketID, command)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) &&
			postgresError.Code == "23505" &&
			postgresError.ConstraintName == issueLiveTicketConstraint {
			return usecase.IssueTicketResult{}, usecase.ErrTicketNotIssuable
		}

		return usecase.IssueTicketResult{}, fmt.Errorf("insert issued ticket: %w", err)
	}

	result := usecase.IssueTicketResult{
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
	}
	if !ticketInserted {
		result, err = findTicketByQueueEntry(ctx, transaction, command.QueueEntryID)
		if errors.Is(err, pgx.ErrNoRows) {
			return usecase.IssueTicketResult{}, errors.New("issued ticket conflict has no existing ticket")
		}
		if err != nil {
			return usecase.IssueTicketResult{}, fmt.Errorf("find existing issued ticket: %w", err)
		}
		if !issueScopeMatches(result, command) {
			return usecase.IssueTicketResult{}, usecase.ErrTicketNotIssuable
		}
		result.Created = false
	}

	responseBody, err := encodeIssueResult(result)
	if err != nil {
		return usecase.IssueTicketResult{}, fmt.Errorf("encode issue response: %w", err)
	}
	responseStatus := issueExistingResponseStatus
	if result.Created {
		responseStatus = issueCreatedResponseStatus
	}
	commandTag, err := transaction.Exec(
		ctx,
		completeIssueOperationQuery,
		toPGUUID(operationID),
		responseStatus,
		responseBody,
		command.IssuedAt,
	)
	if err != nil {
		return usecase.IssueTicketResult{}, fmt.Errorf("complete issue operation: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return usecase.IssueTicketResult{}, fmt.Errorf(
			"complete issue operation: unexpected affected rows %d",
			commandTag.RowsAffected(),
		)
	}

	return result, nil
}

type issueOperationRecord struct {
	ID             uuid.UUID
	ActorID        uuid.UUID
	RequestHash    string
	State          string
	ResponseStatus *int32
	ResponseBody   []byte
}

type issueResultJSON struct {
	ID                 uuid.UUID                 `json:"id"`
	QueueEntryID       uuid.UUID                 `json:"queue_entry_id"`
	UserID             uuid.UUID                 `json:"user_id"`
	ListingID          uuid.UUID                 `json:"listing_id"`
	SKUID              uuid.UUID                 `json:"sku_id"`
	Status             domain.TicketStatus       `json:"status"`
	IssuedAt           time.Time                 `json:"issued_at"`
	ActivationDeadline time.Time                 `json:"activation_deadline"`
	ActivatedAt        *time.Time                `json:"activated_at"`
	OrderID            *uuid.UUID                `json:"order_id"`
	CheckoutURL        *string                   `json:"checkout_url"`
	FinishedAt         *time.Time                `json:"finished_at"`
	FinishReason       *domain.TicketCloseReason `json:"finish_reason"`
}

func findIssueOperation(
	ctx context.Context,
	transaction activationTransaction,
	userID uuid.UUID,
	idempotencyKey uuid.UUID,
) (issueOperationRecord, error) {
	return scanIssueOperation(transaction.QueryRow(
		ctx,
		findIssueOperationQuery,
		toPGUUID(userID),
		issueOperation,
		toPGUUID(idempotencyKey),
	))
}

func scanIssueOperation(row activationRow) (issueOperationRecord, error) {
	var record issueOperationRecord
	var id pgtype.UUID
	var actorID pgtype.UUID
	var responseStatus pgtype.Int4

	err := row.Scan(
		&id,
		&actorID,
		&record.RequestHash,
		&record.State,
		&responseStatus,
		&record.ResponseBody,
	)
	if err != nil {
		return issueOperationRecord{}, err
	}
	if !id.Valid || !actorID.Valid {
		return issueOperationRecord{}, fmt.Errorf("issue operation has null required UUID")
	}

	record.ID = uuid.UUID(id.Bytes)
	record.ActorID = uuid.UUID(actorID.Bytes)
	if responseStatus.Valid {
		value := responseStatus.Int32
		record.ResponseStatus = &value
	}

	return record, nil
}

func insertIssueOperation(
	ctx context.Context,
	transaction activationTransaction,
	operationID uuid.UUID,
	command usecase.IssueTicketCommand,
	requestHash string,
) (bool, error) {
	commandTag, err := transaction.Exec(
		ctx,
		insertIssueOperationQuery,
		toPGUUID(operationID),
		toPGUUID(command.IdempotencyKey),
		issueOperation,
		toPGUUID(command.UserID),
		requestHash,
		issueOperationStateProcessing,
		command.IssuedAt.Add(issueOperationTTL),
		command.IssuedAt,
	)
	if err != nil {
		return false, err
	}

	return commandTag.RowsAffected() == 1, nil
}

func insertIssuedTicket(
	ctx context.Context,
	transaction activationTransaction,
	ticketID uuid.UUID,
	command usecase.IssueTicketCommand,
) (bool, error) {
	commandTag, err := transaction.Exec(
		ctx,
		insertIssuedTicketQuery,
		toPGUUID(ticketID),
		toPGUUID(command.QueueEntryID),
		toPGUUID(command.UserID),
		toPGUUID(command.ListingID),
		toPGUUID(command.SKUID),
		command.ActivationDeadline,
		command.IssuedAt,
	)
	if err != nil {
		return false, err
	}

	return commandTag.RowsAffected() == 1, nil
}

func findTicketByQueueEntry(
	ctx context.Context,
	transaction activationTransaction,
	queueEntryID uuid.UUID,
) (usecase.IssueTicketResult, error) {
	return scanIssuedTicket(transaction.QueryRow(
		ctx,
		findTicketByQueueEntryQuery,
		toPGUUID(queueEntryID),
	))
}

func scanIssuedTicket(row activationRow) (usecase.IssueTicketResult, error) {
	var result usecase.IssueTicketResult
	var ticketID pgtype.UUID
	var queueEntryID pgtype.UUID
	var userID pgtype.UUID
	var listingID pgtype.UUID
	var skuID pgtype.UUID
	var status string
	var activatedAt pgtype.Timestamptz
	var orderID pgtype.UUID
	var checkoutURL pgtype.Text
	var finishedAt pgtype.Timestamptz
	var closeReason pgtype.Text

	err := row.Scan(
		&ticketID,
		&queueEntryID,
		&userID,
		&listingID,
		&skuID,
		&status,
		&result.Ticket.IssuedAt,
		&result.Ticket.ActivationDeadline,
		&activatedAt,
		&orderID,
		&checkoutURL,
		&finishedAt,
		&closeReason,
	)
	if err != nil {
		return usecase.IssueTicketResult{}, err
	}
	if !ticketID.Valid || !queueEntryID.Valid || !userID.Valid || !listingID.Valid || !skuID.Valid {
		return usecase.IssueTicketResult{}, fmt.Errorf("issued ticket has null required UUID")
	}

	result.Ticket.ID = uuid.UUID(ticketID.Bytes)
	result.QueueEntryID = uuid.UUID(queueEntryID.Bytes)
	result.UserID = uuid.UUID(userID.Bytes)
	result.Ticket.ListingID = uuid.UUID(listingID.Bytes)
	result.Ticket.SKUID = uuid.UUID(skuID.Bytes)
	result.Ticket.Status = domain.TicketStatus(status)
	if !result.Ticket.Status.Valid() {
		return usecase.IssueTicketResult{}, fmt.Errorf("issued ticket has unknown status %q", status)
	}
	if activatedAt.Valid {
		value := activatedAt.Time
		result.Ticket.ActivatedAt = &value
	}
	if orderID.Valid {
		value := uuid.UUID(orderID.Bytes)
		result.Ticket.OrderID = &value
	}
	if checkoutURL.Valid {
		value := checkoutURL.String
		result.Ticket.CheckoutURL = &value
	}
	if finishedAt.Valid {
		value := finishedAt.Time
		result.Ticket.FinishedAt = &value
	}
	if closeReason.Valid {
		value := domain.TicketCloseReason(closeReason.String)
		if !value.Valid() {
			return usecase.IssueTicketResult{}, fmt.Errorf("issued ticket has unknown close reason %q", closeReason.String)
		}
		result.Ticket.CloseReason = &value
	}
	if !validIssueSnapshot(result) {
		return usecase.IssueTicketResult{}, fmt.Errorf("issued ticket has invalid snapshot")
	}

	return result, nil
}

func resolveIssueOperation(
	operation issueOperationRecord,
	command usecase.IssueTicketCommand,
	requestHash string,
) (usecase.IssueTicketResult, error) {
	if operation.ActorID != command.UserID || operation.RequestHash != requestHash {
		return usecase.IssueTicketResult{}, usecase.ErrIdempotencyConflict
	}

	switch operation.State {
	case issueOperationStateComplete:
		result, err := decodeIssueResult(operation)
		if err != nil {
			return usecase.IssueTicketResult{}, err
		}
		if !issueScopeMatches(result, command) {
			return usecase.IssueTicketResult{}, errors.New("issue operation has invalid response scope")
		}
		result.Created = false

		return result, nil
	case issueOperationStateProcessing:
		return usecase.IssueTicketResult{}, errors.New("issue operation is unexpectedly processing")
	default:
		return usecase.IssueTicketResult{}, fmt.Errorf("issue operation has unknown state %q", operation.State)
	}
}

func issueRequestHash(command usecase.IssueTicketCommand) string {
	var input [64]byte
	copy(input[:16], command.QueueEntryID[:])
	copy(input[16:32], command.UserID[:])
	copy(input[32:48], command.ListingID[:])
	copy(input[48:], command.SKUID[:])
	hash := sha256.Sum256(input[:])

	return hex.EncodeToString(hash[:])
}

func issueScopeMatches(result usecase.IssueTicketResult, command usecase.IssueTicketCommand) bool {
	return result.QueueEntryID == command.QueueEntryID &&
		result.UserID == command.UserID &&
		result.Ticket.ListingID == command.ListingID &&
		result.Ticket.SKUID == command.SKUID
}

func encodeIssueResult(result usecase.IssueTicketResult) ([]byte, error) {
	return json.Marshal(issueResultJSON{
		ID:                 result.Ticket.ID,
		QueueEntryID:       result.QueueEntryID,
		UserID:             result.UserID,
		ListingID:          result.Ticket.ListingID,
		SKUID:              result.Ticket.SKUID,
		Status:             result.Ticket.Status,
		IssuedAt:           result.Ticket.IssuedAt,
		ActivationDeadline: result.Ticket.ActivationDeadline,
		ActivatedAt:        result.Ticket.ActivatedAt,
		OrderID:            result.Ticket.OrderID,
		CheckoutURL:        result.Ticket.CheckoutURL,
		FinishedAt:         result.Ticket.FinishedAt,
		FinishReason:       result.Ticket.CloseReason,
	})
}

func decodeIssueResult(operation issueOperationRecord) (usecase.IssueTicketResult, error) {
	if operation.ResponseStatus == nil ||
		(*operation.ResponseStatus != issueCreatedResponseStatus &&
			*operation.ResponseStatus != issueExistingResponseStatus) {
		return usecase.IssueTicketResult{}, fmt.Errorf("issue operation has invalid response status")
	}

	var response issueResultJSON
	if err := json.Unmarshal(operation.ResponseBody, &response); err != nil {
		return usecase.IssueTicketResult{}, fmt.Errorf("decode issue response: %w", err)
	}
	result := usecase.IssueTicketResult{
		Ticket: domain.Ticket{
			ID:                 response.ID,
			ListingID:          response.ListingID,
			SKUID:              response.SKUID,
			Status:             response.Status,
			IssuedAt:           response.IssuedAt,
			ActivationDeadline: response.ActivationDeadline,
			ActivatedAt:        response.ActivatedAt,
			OrderID:            response.OrderID,
			CheckoutURL:        response.CheckoutURL,
			FinishedAt:         response.FinishedAt,
			CloseReason:        response.FinishReason,
		},
		QueueEntryID: response.QueueEntryID,
		UserID:       response.UserID,
	}
	if !validIssueSnapshot(result) {
		return usecase.IssueTicketResult{}, fmt.Errorf("issue operation has invalid response")
	}

	return result, nil
}

func validIssueSnapshot(result usecase.IssueTicketResult) bool {
	if result.Ticket.ID == uuid.Nil ||
		result.QueueEntryID == uuid.Nil ||
		result.UserID == uuid.Nil ||
		result.Ticket.ListingID == uuid.Nil ||
		result.Ticket.SKUID == uuid.Nil ||
		!result.Ticket.Status.Valid() ||
		result.Ticket.IssuedAt.IsZero() ||
		!result.Ticket.ActivationDeadline.After(result.Ticket.IssuedAt) ||
		(result.Ticket.OrderID != nil && *result.Ticket.OrderID == uuid.Nil) {
		return false
	}

	return result.Ticket.CloseReason == nil || result.Ticket.CloseReason.Valid()
}

func inIssueTransaction[T any](
	ctx context.Context,
	transactions activationTransactionBeginner,
	action func(activationTransaction) (T, error),
) (result T, err error) {
	transaction, err := transactions.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin issue transaction: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		rollbackContext, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), issueRollbackTimeout)
		defer cancelRollback()
		rollbackErr := transaction.Rollback(rollbackContext)
		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}

		wrappedRollbackErr := fmt.Errorf("rollback issue transaction: %w", rollbackErr)
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
		return result, fmt.Errorf("commit issue transaction: %w", err)
	}
	committed = true

	return result, nil
}
