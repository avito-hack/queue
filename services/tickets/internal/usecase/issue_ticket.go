package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
)

var (
	ErrInvalidTicketIssue = errors.New("invalid ticket issue")
	ErrTicketNotIssuable  = errors.New("ticket is not issuable")
)

type IssueTicketRequest struct {
	QueueEntryID uuid.UUID
	UserID       uuid.UUID
	ListingID    uuid.UUID
	SKUID        uuid.UUID
}

type IssueTicketCommand struct {
	QueueEntryID       uuid.UUID
	UserID             uuid.UUID
	ListingID          uuid.UUID
	SKUID              uuid.UUID
	IdempotencyKey     uuid.UUID
	IssuedAt           time.Time
	ActivationDeadline time.Time
}

type IssueTicketResult struct {
	Ticket       domain.Ticket
	QueueEntryID uuid.UUID
	UserID       uuid.UUID
	Created      bool
}

type TicketIssueRepository interface {
	Issue(context.Context, IssueTicketCommand) (IssueTicketResult, error)
}

type IssueTicket struct {
	repository    TicketIssueRepository
	activationTTL time.Duration
	clock         Clock
}

func NewIssueTicket(
	repository TicketIssueRepository,
	activationTTL time.Duration,
	clocks ...Clock,
) *IssueTicket {
	clock := Clock(time.Now)
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}

	return &IssueTicket{
		repository:    repository,
		activationTTL: activationTTL,
		clock:         clock,
	}
}

func (u *IssueTicket) Issue(
	ctx context.Context,
	request IssueTicketRequest,
	idempotencyKey uuid.UUID,
) (IssueTicketResult, error) {
	if request.QueueEntryID == uuid.Nil {
		return IssueTicketResult{}, fmt.Errorf("%w: empty queue entry id", ErrInvalidTicketIssue)
	}
	if request.UserID == uuid.Nil {
		return IssueTicketResult{}, fmt.Errorf("%w: empty user id", ErrInvalidTicketIssue)
	}
	if request.ListingID == uuid.Nil {
		return IssueTicketResult{}, fmt.Errorf("%w: empty listing id", ErrInvalidTicketIssue)
	}
	if request.SKUID == uuid.Nil {
		return IssueTicketResult{}, fmt.Errorf("%w: empty SKU id", ErrInvalidTicketIssue)
	}
	if idempotencyKey == uuid.Nil {
		return IssueTicketResult{}, fmt.Errorf("%w: empty idempotency key", ErrInvalidTicketIssue)
	}
	if u.activationTTL <= 0 {
		return IssueTicketResult{}, fmt.Errorf("%w: activation TTL must be positive", ErrInvalidTicketIssue)
	}

	now := u.clock()
	command := IssueTicketCommand{
		QueueEntryID:       request.QueueEntryID,
		UserID:             request.UserID,
		ListingID:          request.ListingID,
		SKUID:              request.SKUID,
		IdempotencyKey:     idempotencyKey,
		IssuedAt:           now,
		ActivationDeadline: now.Add(u.activationTTL),
	}
	result, err := u.repository.Issue(ctx, command)
	if err != nil {
		return IssueTicketResult{}, fmt.Errorf("issue ticket: %w", err)
	}
	if !validIssueTicketResult(result, request, command) {
		return IssueTicketResult{}, errors.New("issue ticket: invalid issue result")
	}

	result.Ticket.AvailableActions = result.Ticket.ActionsAt(now)

	return result, nil
}

func validIssueTicketResult(
	result IssueTicketResult,
	request IssueTicketRequest,
	command IssueTicketCommand,
) bool {
	if result.Ticket.ID == uuid.Nil ||
		result.QueueEntryID != request.QueueEntryID ||
		result.UserID != request.UserID ||
		result.Ticket.ListingID != request.ListingID ||
		result.Ticket.SKUID != request.SKUID ||
		!result.Ticket.Status.Valid() ||
		result.Ticket.IssuedAt.IsZero() ||
		!result.Ticket.ActivationDeadline.After(result.Ticket.IssuedAt) {
		return false
	}
	if !result.Created {
		return true
	}

	return result.Ticket.Status == domain.TicketStatusIssued &&
		result.Ticket.IssuedAt.Equal(command.IssuedAt) &&
		result.Ticket.ActivationDeadline.Equal(command.ActivationDeadline) &&
		result.Ticket.ActivatedAt == nil &&
		result.Ticket.OrderID == nil &&
		result.Ticket.CheckoutURL == nil &&
		result.Ticket.FinishedAt == nil &&
		result.Ticket.CloseReason == nil
}
