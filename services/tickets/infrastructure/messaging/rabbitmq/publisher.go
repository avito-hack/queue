package rabbitmq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	eventmessages "github.com/avito-hack/queue/services/tickets/gen/events/messages"
	eventschemas "github.com/avito-hack/queue/services/tickets/gen/events/schemas"
	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type Publisher struct {
	channel  *amqp.Channel
	exchange string
}

func NewPublisher(connection *amqp.Connection, exchange string) (*Publisher, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("open publisher channel: %w", err)
	}
	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("declare event exchange: %w", err)
	}
	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	return &Publisher{channel: channel, exchange: exchange}, nil
}

func (p *Publisher) Publish(ctx context.Context, event usecase.OutboxEvent) error {
	if event.ID == uuid.Nil {
		return errors.New("publish event: empty event id")
	}
	body, err := encodeEventPayload(event)
	if err != nil {
		return err
	}
	confirmation, err := p.channel.PublishWithDeferredConfirmWithContext(
		ctx,
		p.exchange,
		event.Type,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.ID.String(),
			Timestamp:    time.Now(),
			Type:         event.Type,
			AppId:        "tickets",
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish %s: %w", event.Type, err)
	}
	if confirmation == nil {
		return errors.New("publish event: missing broker confirmation")
	}
	acknowledged, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("wait for publisher confirmation: %w", err)
	}
	if !acknowledged {
		return errors.New("publish event: broker rejected message")
	}

	return nil
}

func encodeEventPayload(event usecase.OutboxEvent) ([]byte, error) {
	var buffer bytes.Buffer

	switch event.Type {
	case domain.TicketEventClosed:
		var payload eventschemas.TicketClosedPayload
		if err := decodeEventPayload(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("encode %s: decode payload: %w", event.Type, err)
		}
		if err := validateTicketClosedPayload(payload); err != nil {
			return nil, fmt.Errorf("encode %s: %w", event.Type, err)
		}
		message := eventmessages.TicketClosedOut{Payload: payload}
		if err := message.MarshalAMQP(&buffer); err != nil {
			return nil, fmt.Errorf("encode %s: %w", event.Type, err)
		}
	case domain.TicketEventRedeemed:
		var payload eventschemas.TicketRedeemedPayload
		if err := decodeEventPayload(event.Payload, &payload); err != nil {
			return nil, fmt.Errorf("encode %s: decode payload: %w", event.Type, err)
		}
		if err := validateTicketRedeemedPayload(payload); err != nil {
			return nil, fmt.Errorf("encode %s: %w", event.Type, err)
		}
		message := eventmessages.TicketRedeemedOut{Payload: payload}
		if err := message.MarshalAMQP(&buffer); err != nil {
			return nil, fmt.Errorf("encode %s: %w", event.Type, err)
		}
	default:
		return nil, fmt.Errorf("unsupported event type %q", event.Type)
	}

	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

func decodeEventPayload(payload []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}

		return err
	}

	return nil
}

func validateTicketClosedPayload(payload eventschemas.TicketClosedPayload) error {
	if payload.TicketID == nil || payload.QueueEntryID == nil || payload.UserID == nil ||
		payload.ListingID == nil || payload.SkuID == nil || payload.Status == nil ||
		payload.CloseReason == nil || payload.FinishedAt == nil {
		return errors.New("incomplete payload")
	}
	if *payload.TicketID == uuid.Nil || *payload.QueueEntryID == uuid.Nil || *payload.UserID == uuid.Nil ||
		*payload.ListingID == uuid.Nil || *payload.SkuID == uuid.Nil {
		return errors.New("empty UUID")
	}
	if payload.FinishedAt.IsZero() {
		return errors.New("empty finished time")
	}
	if *payload.Status != "closed" {
		return fmt.Errorf("unexpected status %q", *payload.Status)
	}
	switch *payload.CloseReason {
	case "activation_timeout", "user_declined":
	default:
		return fmt.Errorf("unexpected close reason %q", *payload.CloseReason)
	}

	return nil
}

func validateTicketRedeemedPayload(payload eventschemas.TicketRedeemedPayload) error {
	if payload.TicketID == nil || payload.UserID == nil || payload.OrderID == nil ||
		payload.CheckoutURL == nil || payload.Status == nil || payload.RedeemedAt == nil {
		return errors.New("incomplete payload")
	}
	if *payload.TicketID == uuid.Nil || *payload.UserID == uuid.Nil || *payload.OrderID == uuid.Nil {
		return errors.New("empty UUID")
	}
	if payload.RedeemedAt.IsZero() {
		return errors.New("empty redeemed time")
	}
	if *payload.Status != "redeemed" {
		return fmt.Errorf("unexpected status %q", *payload.Status)
	}
	checkoutURL, err := domain.NormalizeCheckoutURL(*payload.CheckoutURL)
	if err != nil {
		return fmt.Errorf("invalid checkout URL: %w", err)
	}
	if checkoutURL != *payload.CheckoutURL {
		return errors.New("non-normalized checkout URL")
	}

	return nil
}

func (p *Publisher) Close() error {
	return p.channel.Close()
}
