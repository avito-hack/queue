package rabbitmq

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type listingEventHandlerStub struct {
	quantityCommand usecase.ListingQuantityChangedCommand
	statusCommand   usecase.ListingStatusChangedCommand
	revoked         int
	err             error
}

func (s *listingEventHandlerStub) QuantityChanged(
	_ context.Context,
	command usecase.ListingQuantityChangedCommand,
) (int, error) {
	s.quantityCommand = command

	return s.revoked, s.err
}

func (s *listingEventHandlerStub) StatusChanged(
	_ context.Context,
	command usecase.ListingStatusChangedCommand,
) (int, error) {
	s.statusCommand = command

	return s.revoked, s.err
}

func Test_Consumer_QuantityChanged_DispatchGeneratedPayload(t *testing.T) {
	// given
	eventID := uuid.New()
	listingID := uuid.New()
	sellerID := uuid.New()
	changedAt := time.Date(2026, time.August, 9, 10, 0, 0, 0, time.UTC)
	handler := &listingEventHandlerStub{revoked: 2}
	consumer := &Consumer{handler: handler}
	body := []byte(`{"listing_id":"` + listingID.String() + `","seller_id":"` + sellerID.String() + `","previous_quantity":5,"quantity":3,"changed_at":"` + changedAt.Format(time.RFC3339) + `"}`)
	delivery := validListingEventDelivery(eventID, domain.ListingEventQuantityChanged, body)

	// when
	err := consumer.handle(context.Background(), delivery)

	// then
	require.NoError(t, err)
	require.Equal(t, eventID, handler.quantityCommand.Event.ID)
	require.Equal(t, body, handler.quantityCommand.Event.Payload)
	require.Equal(t, listingID, handler.quantityCommand.ListingID)
	require.Equal(t, sellerID, handler.quantityCommand.SellerID)
	require.Equal(t, 5, handler.quantityCommand.PreviousQuantity)
	require.Equal(t, 3, handler.quantityCommand.Quantity)
	require.Equal(t, changedAt, handler.quantityCommand.ChangedAt)
}

func Test_Consumer_StatusChanged_DispatchGeneratedPayload(t *testing.T) {
	// given
	eventID := uuid.New()
	listingID := uuid.New()
	sellerID := uuid.New()
	changedAt := time.Date(2026, time.August, 9, 10, 0, 0, 0, time.UTC)
	handler := &listingEventHandlerStub{revoked: 3}
	consumer := &Consumer{handler: handler}
	body := []byte(`{"listing_id":"` + listingID.String() + `","seller_id":"` + sellerID.String() + `","previous_status":"active","status":"removed","changed_at":"` + changedAt.Format(time.RFC3339) + `"}`)
	delivery := validListingEventDelivery(eventID, domain.ListingEventStatusChanged, body)

	// when
	err := consumer.handle(context.Background(), delivery)

	// then
	require.NoError(t, err)
	require.Equal(t, listingID, handler.statusCommand.ListingID)
	require.Equal(t, sellerID, handler.statusCommand.SellerID)
	require.Equal(t, "active", handler.statusCommand.PreviousStatus)
	require.Equal(t, "removed", handler.statusCommand.Status)
	require.Equal(t, changedAt, handler.statusCommand.ChangedAt)
}

func Test_Consumer_InvalidMessage_ReturnPermanentError(t *testing.T) {
	// given
	tests := []struct {
		name     string
		delivery amqp.Delivery
	}{
		{
			name: "invalid message id",
			delivery: validListingEventDelivery(
				uuid.Nil,
				domain.ListingEventQuantityChanged,
				[]byte(`{}`),
			),
		},
		{
			name: "unknown payload field",
			delivery: validListingEventDelivery(
				uuid.New(),
				domain.ListingEventQuantityChanged,
				[]byte(`{"unknown":true}`),
			),
		},
		{
			name: "unsupported type",
			delivery: validListingEventDelivery(
				uuid.New(),
				"listing.unknown",
				[]byte(`{}`),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			consumer := &Consumer{handler: &listingEventHandlerStub{}}

			// when
			err := consumer.handle(context.Background(), test.delivery)

			// then
			require.ErrorIs(t, err, errInvalidListingEventMessage)
		})
	}
}

func Test_Consumer_HandlerFailed_ReturnTemporaryError(t *testing.T) {
	// given
	handlerError := errors.New("database unavailable")
	handler := &listingEventHandlerStub{err: handlerError}
	consumer := &Consumer{handler: handler}
	body := []byte(`{"listing_id":"` + uuid.NewString() + `","seller_id":"` + uuid.NewString() + `","previous_quantity":5,"quantity":3,"changed_at":"2026-08-09T10:00:00Z"}`)
	delivery := validListingEventDelivery(uuid.New(), domain.ListingEventQuantityChanged, body)

	// when
	err := consumer.handle(context.Background(), delivery)

	// then
	require.ErrorIs(t, err, handlerError)
	require.NotErrorIs(t, err, errInvalidListingEventMessage)
}

func validListingEventDelivery(eventID uuid.UUID, eventType string, body []byte) amqp.Delivery {
	messageID := eventID.String()
	if eventID == uuid.Nil {
		messageID = "invalid"
	}

	return amqp.Delivery{
		ContentType: "application/json",
		MessageId:   messageID,
		Type:        eventType,
		AppId:       "avito-adapter",
		RoutingKey:  eventType,
		Body:        body,
	}
}
