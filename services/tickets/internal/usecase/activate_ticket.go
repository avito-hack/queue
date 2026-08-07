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
	ErrInvalidActivation       = errors.New("invalid ticket activation")
	ErrTicketNotActivatable    = errors.New("ticket is not activatable")
	ErrTicketActivationExpired = errors.New("ticket activation expired")
	ErrIdempotencyConflict     = errors.New("idempotency conflict")
	ErrActivationInProgress    = errors.New("ticket activation is in progress")
	ErrOrderUnavailable        = errors.New("order service unavailable")
	ErrOrderRejected           = errors.New("order creation rejected")
)

type PrepareActivationCommand struct {
	UserID         uuid.UUID
	TicketID       uuid.UUID
	IdempotencyKey uuid.UUID
	Now            time.Time
}

type CreateOrderRequest struct {
	TicketID       uuid.UUID
	ListingID      uuid.UUID
	SKUID          uuid.UUID
	UserID         uuid.UUID
	IdempotencyKey uuid.UUID
}

type CreatedOrder struct {
	ID          uuid.UUID
	CheckoutURL string
}

type ActivationResult struct {
	TicketID    uuid.UUID
	Status      domain.TicketStatus
	OrderID     uuid.UUID
	CheckoutURL string
}

type PreparedActivation struct {
	OperationID uuid.UUID
	Order       CreateOrderRequest
	Replay      *ActivationResult
}

type ActivationRepository interface {
	Prepare(context.Context, PrepareActivationCommand) (PreparedActivation, error)
	Complete(context.Context, uuid.UUID, CreatedOrder, time.Time) (ActivationResult, error)
	Fail(context.Context, uuid.UUID, time.Time) error
}

type OrderCreator interface {
	CreateOrder(context.Context, CreateOrderRequest) (CreatedOrder, error)
}

type ActivateTicket struct {
	repository   ActivationRepository
	orderCreator OrderCreator
	clock        Clock
}

func NewActivateTicket(repository ActivationRepository, orderCreator OrderCreator, clocks ...Clock) *ActivateTicket {
	clock := Clock(time.Now)
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}

	return &ActivateTicket{
		repository:   repository,
		orderCreator: orderCreator,
		clock:        clock,
	}
}

func (u *ActivateTicket) Activate(
	ctx context.Context,
	userID uuid.UUID,
	ticketID uuid.UUID,
	idempotencyKey uuid.UUID,
) (ActivationResult, error) {
	if userID == uuid.Nil {
		return ActivationResult{}, fmt.Errorf("%w: empty user id", ErrInvalidActivation)
	}
	if ticketID == uuid.Nil {
		return ActivationResult{}, fmt.Errorf("%w: empty ticket id", ErrInvalidActivation)
	}
	if idempotencyKey == uuid.Nil {
		return ActivationResult{}, fmt.Errorf("%w: empty idempotency key", ErrInvalidActivation)
	}

	prepared, err := u.repository.Prepare(ctx, PrepareActivationCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: idempotencyKey,
		Now:            u.clock(),
	})
	if err != nil {
		return ActivationResult{}, fmt.Errorf("prepare ticket activation: %w", err)
	}
	if prepared.Replay != nil {
		if err := validateActivationReplay(*prepared.Replay, ticketID); err != nil {
			return ActivationResult{}, fmt.Errorf("prepare ticket activation: %w", err)
		}

		return *prepared.Replay, nil
	}
	if err := validatePreparedActivation(prepared, userID, ticketID); err != nil {
		return ActivationResult{}, fmt.Errorf("prepare ticket activation: %w", err)
	}

	order, err := u.orderCreator.CreateOrder(ctx, prepared.Order)
	if err != nil {
		return ActivationResult{}, u.failPreparedActivation(ctx, prepared.OperationID, fmt.Errorf("create order: %w", err))
	}
	order, err = normalizeCreatedOrder(order)
	if err != nil {
		return ActivationResult{}, u.failPreparedActivation(ctx, prepared.OperationID, fmt.Errorf("create order: %w", err))
	}

	result, err := u.repository.Complete(ctx, prepared.OperationID, order, u.clock())
	if err != nil {
		return ActivationResult{}, fmt.Errorf("complete ticket activation: %w", err)
	}
	if !validActivationResult(result, ticketID) ||
		result.OrderID != order.ID ||
		result.CheckoutURL != order.CheckoutURL {
		return ActivationResult{}, errors.New("complete ticket activation: invalid activation result")
	}

	return result, nil
}

func (u *ActivateTicket) failPreparedActivation(ctx context.Context, operationID uuid.UUID, cause error) error {
	failContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := u.repository.Fail(failContext, operationID, u.clock()); err != nil {
		return errors.Join(cause, fmt.Errorf("fail ticket activation: %w", err))
	}

	return cause
}

func validatePreparedActivation(
	prepared PreparedActivation,
	userID uuid.UUID,
	ticketID uuid.UUID,
) error {
	if prepared.OperationID == uuid.Nil {
		return errors.New("invalid prepared activation: empty operation id")
	}
	if prepared.Order.UserID != userID ||
		prepared.Order.TicketID != ticketID ||
		prepared.Order.IdempotencyKey != prepared.OperationID {
		return errors.New("invalid prepared activation: scope mismatch")
	}
	if prepared.Order.ListingID == uuid.Nil || prepared.Order.SKUID == uuid.Nil {
		return errors.New("invalid prepared activation: empty order scope")
	}

	return nil
}

func validateActivationReplay(result ActivationResult, ticketID uuid.UUID) error {
	if !validActivationResult(result, ticketID) {
		return errors.New("invalid activation replay")
	}

	return nil
}

func normalizeCreatedOrder(order CreatedOrder) (CreatedOrder, error) {
	if order.ID == uuid.Nil {
		return CreatedOrder{}, errors.New("invalid created order: empty order id")
	}

	checkoutURL, err := domain.NormalizeCheckoutURL(order.CheckoutURL)
	if err != nil {
		return CreatedOrder{}, fmt.Errorf("invalid created order: %w", err)
	}
	order.CheckoutURL = checkoutURL

	return order, nil
}

func validActivationResult(result ActivationResult, ticketID uuid.UUID) bool {
	checkoutURL, err := domain.NormalizeCheckoutURL(result.CheckoutURL)

	return err == nil &&
		checkoutURL == result.CheckoutURL &&
		result.TicketID == ticketID &&
		result.Status == domain.TicketStatusActive &&
		result.OrderID != uuid.Nil
}
