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
	"github.com/jackc/pgx/v5/pgxpool"

	sqlgen "github.com/avito-hack/queue/services/tickets/gen/sql"
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

	ticketInserted, err := insertIssuedTicket(
		ctx,
		transaction,
		ticketID,
		command,
	)
	if err != nil {
		var postgresError *pgconn.PgError

		if errors.As(err, &postgresError) &&
			postgresError.Code == "23505" &&
			postgresError.ConstraintName == issueLiveTicketConstraint {

			return usecase.IssueTicketResult{}, usecase.ErrTicketNotIssuable
		}

		return usecase.IssueTicketResult{}, fmt.Errorf(
			"insert issued ticket: %w",
			err,
		)
	}

	var result usecase.IssueTicketResult

	if ticketInserted {
		result = usecase.IssueTicketResult{
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
	} else {
		// rowsAffected == 0 значит либо квота исчерпана (WHERE ticket_count.total < quantity
		// не прошло), либо сработал ON CONFLICT (queue_entry_id) DO NOTHING на повторной
		// попытке для того же queue_entry_id. Различаем по наличию существующей строки.
		existing, findErr := findTicketByQueueEntry(ctx, transaction, command.QueueEntryID)
		if errors.Is(findErr, pgx.ErrNoRows) {
			return usecase.IssueTicketResult{}, usecase.ErrTicketNotIssuable
		}
		if findErr != nil {
			return usecase.IssueTicketResult{}, fmt.Errorf("find existing issued ticket: %w", findErr)
		}
		if !issueScopeMatches(existing, command) {
			return usecase.IssueTicketResult{}, usecase.ErrTicketNotIssuable
		}

		existing.Created = false
		result = existing
	}

	responseBody, err := encodeIssueResult(result)
	if err != nil {
		return usecase.IssueTicketResult{}, fmt.Errorf("encode issue response: %w", err)
	}

	responseStatus := issueExistingResponseStatus
	if result.Created {
		responseStatus = issueCreatedResponseStatus
	}

	rowsAffected, err := sqlgen.New(transaction).CompleteIssueOperation(ctx, sqlgen.CompleteIssueOperationParams{
		ResponseStatus: toPGInt4(responseStatus),
		ResponseBody:   responseBody,
		UpdatedAt:      toPGTimestamptz(command.IssuedAt),
		ID:             toPGUUID(operationID),
	})
	if err != nil {
		return usecase.IssueTicketResult{}, fmt.Errorf("complete issue operation: %w", err)
	}
	if rowsAffected != 1 {
		return usecase.IssueTicketResult{}, fmt.Errorf(
			"complete issue operation: unexpected affected rows %d",
			rowsAffected,
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
	row, err := sqlgen.New(transaction).FindIssueOperation(ctx, sqlgen.FindIssueOperationParams{
		ActorID:        toPGUUID(userID),
		Operation:      issueOperation,
		IdempotencyKey: toPGUUID(idempotencyKey),
	})
	if err != nil {
		return issueOperationRecord{}, err
	}
	if !row.ID.Valid || !row.ActorID.Valid {
		return issueOperationRecord{}, fmt.Errorf("issue operation has null required UUID")
	}

	record := issueOperationRecord{
		ID:           uuid.UUID(row.ID.Bytes),
		ActorID:      uuid.UUID(row.ActorID.Bytes),
		RequestHash:  row.RequestHash,
		State:        row.State,
		ResponseBody: row.ResponseBody,
	}
	if row.ResponseStatus.Valid {
		value := row.ResponseStatus.Int32
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
	rowsAffected, err := sqlgen.New(transaction).InsertIssueOperation(ctx, sqlgen.InsertIssueOperationParams{
		ID:             toPGUUID(operationID),
		IdempotencyKey: toPGUUID(command.IdempotencyKey),
		Operation:      issueOperation,
		ActorID:        toPGUUID(command.UserID),
		RequestHash:    requestHash,
		State:          issueOperationStateProcessing,
		ExpiresAt:      toPGTimestamptz(command.IssuedAt.Add(issueOperationTTL)),
		UpdatedAt:      toPGTimestamptz(command.IssuedAt),
	})
	if err != nil {
		return false, err
	}

	return rowsAffected == 1, nil
}

func insertIssuedTicket(
	ctx context.Context,
	transaction activationTransaction,
	ticketID uuid.UUID,
	command usecase.IssueTicketCommand,
) (bool, error) {
	if err := sqlgen.New(transaction).LockListingForTicketIssue(
		ctx,
		toPGUUID(command.ListingID),
	); err != nil {
		return false, fmt.Errorf("lock listing for ticket issue: %w", err)
	}

	rowsAffected, err := sqlgen.New(transaction).InsertIssuedTicket(
		ctx,
		sqlgen.InsertIssuedTicketParams{
			ID:                 toPGUUID(ticketID),
			QueueEntryID:       toPGUUID(command.QueueEntryID),
			UserID:             toPGUUID(command.UserID),
			ListingID:          toPGUUID(command.ListingID),
			SkuID:              toPGUUID(command.SKUID),
			ListingQuantity:    int32(command.ListingQuantity),
			ActivationDeadline: toPGTimestamptz(command.ActivationDeadline),
			IssuedAt:           toPGTimestamptz(command.IssuedAt),
		},
	)
	if err != nil {
		return false, err
	}

	return rowsAffected == 1, nil
}

func findTicketByQueueEntry(
	ctx context.Context,
	transaction activationTransaction,
	queueEntryID uuid.UUID,
) (usecase.IssueTicketResult, error) {
	row, err := sqlgen.New(transaction).FindTicketByQueueEntry(ctx, toPGUUID(queueEntryID))
	if err != nil {
		return usecase.IssueTicketResult{}, err
	}
	if !row.ID.Valid || !row.QueueEntryID.Valid || !row.UserID.Valid || !row.ListingID.Valid || !row.SkuID.Valid ||
		!row.IssuedAt.Valid || !row.ActivationDeadline.Valid {
		return usecase.IssueTicketResult{}, fmt.Errorf("issued ticket has null required UUID")
	}

	result := usecase.IssueTicketResult{
		Ticket: domain.Ticket{
			ID:                 uuid.UUID(row.ID.Bytes),
			ListingID:          uuid.UUID(row.ListingID.Bytes),
			SKUID:              uuid.UUID(row.SkuID.Bytes),
			Status:             domain.TicketStatus(row.Status),
			IssuedAt:           row.IssuedAt.Time,
			ActivationDeadline: row.ActivationDeadline.Time,
		},
		QueueEntryID: uuid.UUID(row.QueueEntryID.Bytes),
		UserID:       uuid.UUID(row.UserID.Bytes),
	}
	if !result.Ticket.Status.Valid() {
		return usecase.IssueTicketResult{}, fmt.Errorf("issued ticket has unknown status %q", row.Status)
	}
	if row.ActivatedAt.Valid {
		value := row.ActivatedAt.Time
		result.Ticket.ActivatedAt = &value
	}
	if row.OrderID.Valid {
		value := uuid.UUID(row.OrderID.Bytes)
		result.Ticket.OrderID = &value
	}
	if row.CheckoutUrl.Valid {
		value := row.CheckoutUrl.String
		result.Ticket.CheckoutURL = &value
	}
	if row.FinishedAt.Valid {
		value := row.FinishedAt.Time
		result.Ticket.FinishedAt = &value
	}
	if row.CloseReason.Valid {
		value := domain.TicketCloseReason(row.CloseReason.String)
		if !value.Valid() {
			return usecase.IssueTicketResult{}, fmt.Errorf("issued ticket has unknown close reason %q", row.CloseReason.String)
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