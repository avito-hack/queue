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

type activationRepositoryStub struct {
	prepared               PreparedActivation
	prepareErr             error
	completeResult         ActivationResult
	completeErr            error
	failErr                error
	prepareCalls           int
	completeCalls          int
	failCalls              int
	receivedPrepareCommand PrepareActivationCommand
	receivedOperationID    uuid.UUID
	receivedOrder          CreatedOrder
	receivedCompletionTime time.Time
}

func (s *activationRepositoryStub) Fail(_ context.Context, _ uuid.UUID, _ time.Time) error {
	s.failCalls++

	return s.failErr
}

func (s *activationRepositoryStub) Prepare(
	_ context.Context,
	command PrepareActivationCommand,
) (PreparedActivation, error) {
	s.prepareCalls++
	s.receivedPrepareCommand = command

	return s.prepared, s.prepareErr
}

func (s *activationRepositoryStub) Complete(
	_ context.Context,
	operationID uuid.UUID,
	order CreatedOrder,
	completedAt time.Time,
) (ActivationResult, error) {
	s.completeCalls++
	s.receivedOperationID = operationID
	s.receivedOrder = order
	s.receivedCompletionTime = completedAt

	return s.completeResult, s.completeErr
}

type orderCreatorStub struct {
	order           CreatedOrder
	err             error
	calls           int
	receivedRequest CreateOrderRequest
}

func (s *orderCreatorStub) CreateOrder(_ context.Context, request CreateOrderRequest) (CreatedOrder, error) {
	s.calls++
	s.receivedRequest = request

	return s.order, s.err
}

type orderCreatorFunc func(context.Context, CreateOrderRequest) (CreatedOrder, error)

func (f orderCreatorFunc) CreateOrder(ctx context.Context, request CreateOrderRequest) (CreatedOrder, error) {
	return f(ctx, request)
}

func Test_ActivateTicket_EligibleTicket_ReturnActivationResult(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	idempotencyKey := uuid.New()
	operationID := uuid.New()
	orderID := uuid.New()
	preparedAt := time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC)
	completedAt := preparedAt.Add(time.Second)
	clockCalls := 0
	clock := func() time.Time {
		clockCalls++
		if clockCalls == 1 {
			return preparedAt
		}

		return completedAt
	}
	orderRequest := CreateOrderRequest{
		TicketID:       ticketID,
		ListingID:      listingID,
		SKUID:          skuID,
		UserID:         userID,
		IdempotencyKey: operationID,
	}
	order := CreatedOrder{ID: orderID, CheckoutURL: "/checkout/" + orderID.String()}
	expected := ActivationResult{
		TicketID:    ticketID,
		Status:      domain.TicketStatusRedeemed,
		OrderID:     orderID,
		CheckoutURL: order.CheckoutURL,
	}
	repository := &activationRepositoryStub{
		prepared: PreparedActivation{
			OperationID: operationID,
			Order:       orderRequest,
		},
		completeResult: expected,
	}
	orderCreator := &orderCreatorStub{order: order}
	useCase := NewActivateTicket(repository, orderCreator, clock)

	// when
	result, err := useCase.Activate(context.Background(), userID, ticketID, idempotencyKey)

	// then
	require.NoError(t, err)
	require.Equal(t, expected, result)
	require.Equal(t, 1, repository.prepareCalls)
	require.Equal(t, PrepareActivationCommand{
		UserID:         userID,
		TicketID:       ticketID,
		IdempotencyKey: idempotencyKey,
		Now:            preparedAt,
	}, repository.receivedPrepareCommand)
	require.Equal(t, 1, orderCreator.calls)
	require.Equal(t, orderRequest, orderCreator.receivedRequest)
	require.Equal(t, 1, repository.completeCalls)
	require.Equal(t, operationID, repository.receivedOperationID)
	require.Equal(t, order, repository.receivedOrder)
	require.Equal(t, completedAt, repository.receivedCompletionTime)
	require.Equal(t, 2, clockCalls)
}

func Test_ActivateTicket_CompletedOperation_ReturnReplayWithoutCreatingOrder(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	idempotencyKey := uuid.New()
	replay := ActivationResult{
		TicketID:    ticketID,
		Status:      domain.TicketStatusRedeemed,
		OrderID:     uuid.New(),
		CheckoutURL: "/checkout/replayed",
	}
	repository := &activationRepositoryStub{prepared: PreparedActivation{Replay: &replay}}
	orderCreator := &orderCreatorStub{}
	clockCalls := 0
	useCase := NewActivateTicket(repository, orderCreator, func() time.Time {
		clockCalls++
		return time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC)
	})

	// when
	result, err := useCase.Activate(context.Background(), userID, ticketID, idempotencyKey)

	// then
	require.NoError(t, err)
	require.Equal(t, replay, result)
	require.Equal(t, 1, repository.prepareCalls)
	require.Zero(t, repository.completeCalls)
	require.Zero(t, orderCreator.calls)
	require.Equal(t, 1, clockCalls)
}

func Test_ActivateTicket_InvalidPreparedActivation_ReturnError(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	idempotencyKey := uuid.New()
	operationID := uuid.New()
	valid := PreparedActivation{
		OperationID: operationID,
		Order: CreateOrderRequest{
			TicketID:       ticketID,
			ListingID:      uuid.New(),
			SKUID:          uuid.New(),
			UserID:         userID,
			IdempotencyKey: operationID,
		},
	}
	tests := []struct {
		name          string
		prepared      PreparedActivation
		expectedError string
	}{
		{
			name: "empty operation id",
			prepared: func() PreparedActivation {
				result := valid
				result.OperationID = uuid.Nil

				return result
			}(),
			expectedError: "prepare ticket activation: invalid prepared activation: empty operation id",
		},
		{
			name: "scope mismatch",
			prepared: func() PreparedActivation {
				result := valid
				result.Order.UserID = uuid.New()

				return result
			}(),
			expectedError: "prepare ticket activation: invalid prepared activation: scope mismatch",
		},
		{
			name: "empty order scope",
			prepared: func() PreparedActivation {
				result := valid
				result.Order.ListingID = uuid.Nil

				return result
			}(),
			expectedError: "prepare ticket activation: invalid prepared activation: empty order scope",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			orderCreator := &orderCreatorStub{}
			repository := &activationRepositoryStub{prepared: test.prepared}
			useCase := NewActivateTicket(repository, orderCreator, time.Now)

			// when
			_, err := useCase.Activate(context.Background(), userID, ticketID, idempotencyKey)

			// then
			require.EqualError(t, err, test.expectedError)
			require.Zero(t, orderCreator.calls)
			require.Zero(t, repository.completeCalls)
		})
	}
}

func Test_ActivateTicket_InvalidReplay_ReturnError(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	idempotencyKey := uuid.New()
	replay := ActivationResult{
		TicketID:    uuid.New(),
		Status:      domain.TicketStatusRedeemed,
		OrderID:     uuid.New(),
		CheckoutURL: "/checkout/replayed",
	}
	repository := &activationRepositoryStub{prepared: PreparedActivation{Replay: &replay}}
	orderCreator := &orderCreatorStub{}
	useCase := NewActivateTicket(repository, orderCreator, time.Now)

	// when
	_, err := useCase.Activate(context.Background(), userID, ticketID, idempotencyKey)

	// then
	require.EqualError(t, err, "prepare ticket activation: invalid activation replay")
	require.Zero(t, orderCreator.calls)
	require.Zero(t, repository.completeCalls)
}

func Test_ActivateTicket_EmptyIdentifier_ReturnInvalidActivation(t *testing.T) {
	tests := []struct {
		name           string
		userID         uuid.UUID
		ticketID       uuid.UUID
		idempotencyKey uuid.UUID
		expectedError  string
	}{
		{
			name:           "empty user id",
			ticketID:       uuid.New(),
			idempotencyKey: uuid.New(),
			expectedError:  "invalid ticket activation: empty user id",
		},
		{
			name:           "empty ticket id",
			userID:         uuid.New(),
			idempotencyKey: uuid.New(),
			expectedError:  "invalid ticket activation: empty ticket id",
		},
		{
			name:          "empty idempotency key",
			userID:        uuid.New(),
			ticketID:      uuid.New(),
			expectedError: "invalid ticket activation: empty idempotency key",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			repository := &activationRepositoryStub{}
			orderCreator := &orderCreatorStub{}
			useCase := NewActivateTicket(repository, orderCreator, time.Now)

			// when
			_, err := useCase.Activate(
				context.Background(),
				test.userID,
				test.ticketID,
				test.idempotencyKey,
			)

			// then
			require.Error(t, err)
			require.ErrorIs(t, err, ErrInvalidActivation)
			require.Equal(t, test.expectedError, err.Error())
			require.Zero(t, repository.prepareCalls)
			require.Zero(t, orderCreator.calls)
		})
	}
}

func Test_ActivateTicket_PrepareFails_ReturnDomainError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "ticket not found", err: ErrTicketNotFound},
		{name: "ticket not activatable", err: ErrTicketNotActivatable},
		{name: "ticket activation expired", err: ErrTicketActivationExpired},
		{name: "idempotency conflict", err: ErrIdempotencyConflict},
		{name: "activation in progress", err: ErrActivationInProgress},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			repository := &activationRepositoryStub{prepareErr: test.err}
			orderCreator := &orderCreatorStub{}
			useCase := NewActivateTicket(repository, orderCreator, time.Now)

			// when
			_, err := useCase.Activate(context.Background(), uuid.New(), uuid.New(), uuid.New())

			// then
			require.Error(t, err)
			require.ErrorIs(t, err, test.err)
			require.Equal(t, "prepare ticket activation: "+test.err.Error(), err.Error())
			require.Equal(t, 1, repository.prepareCalls)
			require.Zero(t, repository.completeCalls)
			require.Zero(t, orderCreator.calls)
		})
	}
}

func Test_ActivateTicket_OrderUnavailable_ReturnOrderUnavailable(t *testing.T) {
	// given
	operationID := uuid.New()
	idempotencyKey := uuid.New()
	repository := &activationRepositoryStub{prepared: PreparedActivation{
		OperationID: operationID,
		Order: CreateOrderRequest{
			TicketID:       uuid.New(),
			ListingID:      uuid.New(),
			SKUID:          uuid.New(),
			UserID:         uuid.New(),
			IdempotencyKey: operationID,
		},
	}}
	orderCreator := &orderCreatorStub{err: ErrOrderUnavailable}
	useCase := NewActivateTicket(repository, orderCreator, time.Now)

	// when
	_, err := useCase.Activate(
		context.Background(),
		repository.prepared.Order.UserID,
		repository.prepared.Order.TicketID,
		idempotencyKey,
	)

	// then
	require.Error(t, err)
	require.ErrorIs(t, err, ErrOrderUnavailable)
	require.Equal(t, "create order: order service unavailable", err.Error())
	require.Equal(t, 1, orderCreator.calls)
	require.Zero(t, repository.completeCalls)
	require.Equal(t, 1, repository.failCalls)
}

func Test_ActivateTicket_OrderUnavailable_RetrySameOperation(t *testing.T) {
	// given
	operationID := uuid.New()
	idempotencyKey := uuid.New()
	prepared := PreparedActivation{
		OperationID: operationID,
		Order: CreateOrderRequest{
			TicketID:       uuid.New(),
			ListingID:      uuid.New(),
			SKUID:          uuid.New(),
			UserID:         uuid.New(),
			IdempotencyKey: operationID,
		},
	}
	order := CreatedOrder{ID: uuid.New(), CheckoutURL: "/checkout/retried"}
	expected := ActivationResult{
		TicketID:    prepared.Order.TicketID,
		Status:      domain.TicketStatusRedeemed,
		OrderID:     order.ID,
		CheckoutURL: order.CheckoutURL,
	}
	repository := &activationRepositoryStub{prepared: prepared, completeResult: expected}
	orderCalls := 0
	orderCreator := orderCreatorFunc(func(context.Context, CreateOrderRequest) (CreatedOrder, error) {
		orderCalls++
		if orderCalls == 1 {
			return CreatedOrder{}, ErrOrderUnavailable
		}

		return order, nil
	})
	useCase := NewActivateTicket(repository, orderCreator, time.Now)

	// when
	_, firstErr := useCase.Activate(
		context.Background(),
		prepared.Order.UserID,
		prepared.Order.TicketID,
		idempotencyKey,
	)
	result, secondErr := useCase.Activate(
		context.Background(),
		prepared.Order.UserID,
		prepared.Order.TicketID,
		idempotencyKey,
	)

	// then
	require.ErrorIs(t, firstErr, ErrOrderUnavailable)
	require.NoError(t, secondErr)
	require.Equal(t, expected, result)
	require.Equal(t, 2, repository.prepareCalls)
	require.Equal(t, 2, orderCalls)
	require.Equal(t, 1, repository.completeCalls)
	require.Equal(t, 1, repository.failCalls)
}

func Test_ActivateTicket_CreateOrderFails_ReturnWrappedError(t *testing.T) {
	// given
	orderError := errors.New("create order failed")
	operationID := uuid.New()
	prepared := PreparedActivation{
		OperationID: operationID,
		Order: CreateOrderRequest{
			TicketID:       uuid.New(),
			ListingID:      uuid.New(),
			SKUID:          uuid.New(),
			UserID:         uuid.New(),
			IdempotencyKey: operationID,
		},
	}
	repository := &activationRepositoryStub{prepared: prepared}
	orderCreator := &orderCreatorStub{err: orderError}
	useCase := NewActivateTicket(repository, orderCreator, time.Now)

	// when
	_, err := useCase.Activate(
		context.Background(),
		prepared.Order.UserID,
		prepared.Order.TicketID,
		uuid.New(),
	)

	// then
	require.Error(t, err)
	require.ErrorIs(t, err, orderError)
	require.Equal(t, "create order: create order failed", err.Error())
	require.Zero(t, repository.completeCalls)
	require.Equal(t, 1, repository.failCalls)
}

func Test_ActivateTicket_InvalidCreatedOrder_ReturnError(t *testing.T) {
	// given
	operationID := uuid.New()
	prepared := PreparedActivation{
		OperationID: operationID,
		Order: CreateOrderRequest{
			TicketID:       uuid.New(),
			ListingID:      uuid.New(),
			SKUID:          uuid.New(),
			UserID:         uuid.New(),
			IdempotencyKey: operationID,
		},
	}
	tests := []struct {
		name  string
		order CreatedOrder
	}{
		{name: "empty order id", order: CreatedOrder{CheckoutURL: "/checkout/1"}},
		{name: "unsafe checkout URL", order: CreatedOrder{ID: uuid.New(), CheckoutURL: "javascript:alert(1)"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &activationRepositoryStub{prepared: prepared}
			orderCreator := &orderCreatorStub{order: test.order}
			useCase := NewActivateTicket(repository, orderCreator, time.Now)

			// when
			_, err := useCase.Activate(
				context.Background(),
				prepared.Order.UserID,
				prepared.Order.TicketID,
				uuid.New(),
			)

			// then
			require.Error(t, err)
			require.Contains(t, err.Error(), "create order: invalid created order")
			require.Zero(t, repository.completeCalls)
			require.Equal(t, 1, repository.failCalls)
		})
	}
}

func Test_ActivateTicket_CompleteFails_ReturnWrappedError(t *testing.T) {
	// given
	completeError := errors.New("complete activation failed")
	operationID := uuid.New()
	prepared := PreparedActivation{
		OperationID: operationID,
		Order: CreateOrderRequest{
			TicketID:       uuid.New(),
			ListingID:      uuid.New(),
			SKUID:          uuid.New(),
			UserID:         uuid.New(),
			IdempotencyKey: operationID,
		},
	}
	repository := &activationRepositoryStub{prepared: prepared, completeErr: completeError}
	orderCreator := &orderCreatorStub{order: CreatedOrder{ID: uuid.New(), CheckoutURL: "/checkout/1"}}
	useCase := NewActivateTicket(repository, orderCreator, time.Now)

	// when
	_, err := useCase.Activate(
		context.Background(),
		prepared.Order.UserID,
		prepared.Order.TicketID,
		uuid.New(),
	)

	// then
	require.Error(t, err)
	require.ErrorIs(t, err, completeError)
	require.Equal(t, "complete ticket activation: complete activation failed", err.Error())
	require.Equal(t, 1, orderCreator.calls)
	require.Equal(t, 1, repository.completeCalls)
}

func Test_ActivateTicket_CompleteReturnsInvalidResult_ReturnError(t *testing.T) {
	// given
	operationID := uuid.New()
	prepared := PreparedActivation{
		OperationID: operationID,
		Order: CreateOrderRequest{
			TicketID:       uuid.New(),
			ListingID:      uuid.New(),
			SKUID:          uuid.New(),
			UserID:         uuid.New(),
			IdempotencyKey: operationID,
		},
	}
	order := CreatedOrder{ID: uuid.New(), CheckoutURL: "/checkout/1"}
	repository := &activationRepositoryStub{
		prepared: prepared,
		completeResult: ActivationResult{
			TicketID:    prepared.Order.TicketID,
			Status:      domain.TicketStatusRedeemed,
			OrderID:     uuid.New(),
			CheckoutURL: order.CheckoutURL,
		},
	}
	orderCreator := &orderCreatorStub{order: order}
	useCase := NewActivateTicket(repository, orderCreator, time.Now)

	// when
	_, err := useCase.Activate(
		context.Background(),
		prepared.Order.UserID,
		prepared.Order.TicketID,
		uuid.New(),
	)

	// then
	require.EqualError(t, err, "complete ticket activation: invalid activation result")
	require.Equal(t, 1, repository.completeCalls)
}
