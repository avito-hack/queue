package rabbitmq

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

func Test_EncodeEventPayload_TicketClosed_ReturnTypedJSON(t *testing.T) {
	// given
	ticketID := uuid.New()
	queueEntryID := uuid.New()
	userID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	event := usecase.OutboxEvent{
		Type: ticketClosedEventType,
		Payload: []byte(`{
			"ticket_id":"` + ticketID.String() + `",
			"queue_entry_id":"` + queueEntryID.String() + `",
			"user_id":"` + userID.String() + `",
			"listing_id":"` + listingID.String() + `",
			"sku_id":"` + skuID.String() + `",
			"status":"closed",
			"close_reason":"activation_timeout",
			"finished_at":"2026-08-07T10:00:00Z"
		}`),
	}

	// when
	payload, err := encodeEventPayload(event)

	// then
	require.NoError(t, err)
	assert.JSONEq(t, string(event.Payload), string(payload))
}

func Test_EncodeEventPayload_IncompleteTicketRedeemed_ReturnError(t *testing.T) {
	// given
	event := usecase.OutboxEvent{
		Type:    ticketRedeemedEventType,
		Payload: []byte(`{"status":"redeemed"}`),
	}

	// when
	_, err := encodeEventPayload(event)

	// then
	require.Error(t, err)
	assert.Equal(t, "encode ticket.redeemed: incomplete payload", err.Error())
}
