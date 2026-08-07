package rabbitmq

import (
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

func Test_LifecycleEventFromDelivery_ValidPaymentMessage_ReturnEvent(t *testing.T) {
	// given
	eventID := uuid.New()
	orderID := uuid.New()
	occurredAt := time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC)
	delivery := amqp.Delivery{
		MessageId: eventID.String(),
		Type:      string(usecase.LifecycleEventPaymentSucceeded),
		AppId:     "avito-adapter",
		Timestamp: occurredAt,
		Body:      []byte(`{"order_id":"` + orderID.String() + `"}`),
	}

	// when
	event, err := lifecycleEventFromDelivery(delivery)

	// then
	require.NoError(t, err)
	assert.Equal(t, eventID, event.ID)
	assert.Equal(t, orderID, event.OrderID)
	assert.Equal(t, occurredAt, event.OccurredAt)
	assert.Equal(t, usecase.LifecycleEventPaymentSucceeded, event.Type)
	assert.Equal(t, "avito-adapter", event.Source)
}
