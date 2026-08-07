package rabbitmq

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
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
		Type: domain.TicketEventClosed,
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
		Type:    domain.TicketEventRedeemed,
		Payload: []byte(`{"status":"redeemed"}`),
	}

	// when
	_, err := encodeEventPayload(event)

	// then
	require.Error(t, err)
	assert.Equal(t, "encode ticket.redeemed: incomplete payload", err.Error())
}

func Test_EncodeEventPayload_TicketRedeemed_ReturnTypedJSON(t *testing.T) {
	// given
	event := usecase.OutboxEvent{
		Type: domain.TicketEventRedeemed,
		Payload: []byte(`{
			"ticket_id":"` + uuid.NewString() + `",
			"user_id":"` + uuid.NewString() + `",
			"order_id":"` + uuid.NewString() + `",
			"checkout_url":"/checkout/1",
			"status":"redeemed",
			"redeemed_at":"2026-08-07T10:00:00Z"
		}`),
	}

	// when
	payload, err := encodeEventPayload(event)

	// then
	require.NoError(t, err)
	assert.JSONEq(t, string(event.Payload), string(payload))
}

func Test_EncodeEventPayload_UnknownEvent_ReturnError(t *testing.T) {
	// given
	event := usecase.OutboxEvent{Type: "ticket.unknown", Payload: []byte(`{}`)}

	// when
	_, err := encodeEventPayload(event)

	// then
	require.Error(t, err)
	assert.Equal(t, `unsupported event type "ticket.unknown"`, err.Error())
}

func Test_EncodeEventPayload_ExtraField_ReturnError(t *testing.T) {
	// given
	event := usecase.OutboxEvent{
		Type: domain.TicketEventClosed,
		Payload: []byte(`{
			"ticket_id":"` + uuid.NewString() + `",
			"queue_entry_id":"` + uuid.NewString() + `",
			"user_id":"` + uuid.NewString() + `",
			"listing_id":"` + uuid.NewString() + `",
			"sku_id":"` + uuid.NewString() + `",
			"status":"closed",
			"close_reason":"activation_timeout",
			"finished_at":"2026-08-07T10:00:00Z",
			"extra":true
		}`),
	}

	// when
	_, err := encodeEventPayload(event)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown field "extra"`)
}

func Test_PublisherPublish_EmptyEventID_ReturnError(t *testing.T) {
	// given
	publisher := &Publisher{}

	// when
	err := publisher.Publish(t.Context(), usecase.OutboxEvent{Type: domain.TicketEventClosed})

	// then
	require.Error(t, err)
	assert.Equal(t, "publish event: empty event id", err.Error())
}
