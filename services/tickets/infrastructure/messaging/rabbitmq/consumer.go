package rabbitmq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	avitoevents "github.com/avito-hack/queue/services/tickets/gen/clients/avitoadapterevents/schemas"
	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

var errInvalidListingEventMessage = errors.New("invalid listing event message")

type ListingEventHandler interface {
	QuantityChanged(context.Context, usecase.ListingQuantityChangedCommand) (int, error)
	StatusChanged(context.Context, usecase.ListingStatusChangedCommand) (int, error)
}

type Consumer struct {
	channel *amqp.Channel
	queue   string
	handler ListingEventHandler
}

func NewConsumer(
	connection *amqp.Connection,
	exchange string,
	queue string,
	prefetch int,
	handler ListingEventHandler,
) (*Consumer, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("open consumer channel: %w", err)
	}
	closeOnError := func(err error) (*Consumer, error) {
		_ = channel.Close()

		return nil, err
	}
	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return closeOnError(fmt.Errorf("declare event exchange: %w", err))
	}
	if _, err := channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		return closeOnError(fmt.Errorf("declare listing events queue: %w", err))
	}
	for _, routingKey := range []string{domain.ListingEventQuantityChanged, domain.ListingEventStatusChanged} {
		if err := channel.QueueBind(queue, routingKey, exchange, false, nil); err != nil {
			return closeOnError(fmt.Errorf("bind listing events queue to %s: %w", routingKey, err))
		}
	}
	if err := channel.Qos(prefetch, 0, false); err != nil {
		return closeOnError(fmt.Errorf("configure listing events consumer qos: %w", err))
	}

	return &Consumer{channel: channel, queue: queue, handler: handler}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	deliveries, err := c.channel.ConsumeWithContext(ctx, c.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume listing events: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}

				return errors.New("listing events delivery channel closed")
			}
			if err := c.handle(ctx, delivery); err != nil {
				if errors.Is(err, errInvalidListingEventMessage) || errors.Is(err, usecase.ErrInvalidListingEvent) {
					slog.WarnContext(ctx, "reject invalid listing event", "error", err)
					if rejectErr := delivery.Reject(false); rejectErr != nil {
						return fmt.Errorf("reject invalid listing event: %w", rejectErr)
					}

					continue
				}
				slog.ErrorContext(ctx, "retry listing event", "error", err)
				if nackErr := delivery.Nack(false, true); nackErr != nil {
					return fmt.Errorf("requeue listing event: %w", nackErr)
				}

				continue
			}
			if err := delivery.Ack(false); err != nil {
				return fmt.Errorf("acknowledge listing event: %w", err)
			}
		}
	}
}

func (c *Consumer) handle(ctx context.Context, delivery amqp.Delivery) error {
	if delivery.ContentType != "application/json" {
		return fmt.Errorf("%w: unexpected content type %q", errInvalidListingEventMessage, delivery.ContentType)
	}
	if delivery.Type != delivery.RoutingKey {
		return fmt.Errorf(
			"%w: message type %q does not match routing key %q",
			errInvalidListingEventMessage,
			delivery.Type,
			delivery.RoutingKey,
		)
	}
	eventID, err := uuid.Parse(delivery.MessageId)
	if err != nil || eventID == uuid.Nil {
		return fmt.Errorf("%w: invalid message id %q", errInvalidListingEventMessage, delivery.MessageId)
	}
	event := usecase.IncomingListingEvent{
		ID:      eventID,
		Type:    delivery.Type,
		Source:  delivery.AppId,
		Payload: delivery.Body,
	}

	switch delivery.Type {
	case domain.ListingEventQuantityChanged:
		var payload avitoevents.ListingQuantityChangedPayload
		if err := decodeListingEvent(delivery.Body, &payload); err != nil {
			return fmt.Errorf("%w: decode quantity changed: %w", errInvalidListingEventMessage, err)
		}
		if payload.ListingID == nil || payload.SellerID == nil || payload.PreviousQuantity == nil ||
			payload.Quantity == nil || payload.ChangedAt == nil {
			return fmt.Errorf("%w: incomplete quantity changed payload", errInvalidListingEventMessage)
		}
		revoked, err := c.handler.QuantityChanged(ctx, usecase.ListingQuantityChangedCommand{
			Event:            event,
			ListingID:        *payload.ListingID,
			SellerID:         *payload.SellerID,
			PreviousQuantity: *payload.PreviousQuantity,
			Quantity:         *payload.Quantity,
			ChangedAt:        *payload.ChangedAt,
		})
		if err != nil {
			return err
		}
		slog.InfoContext(ctx, "handled listing quantity change", "listing_id", payload.ListingID, "revoked", revoked)

		return nil
	case domain.ListingEventStatusChanged:
		var payload avitoevents.ListingStatusChangedPayload
		if err := decodeListingEvent(delivery.Body, &payload); err != nil {
			return fmt.Errorf("%w: decode status changed: %w", errInvalidListingEventMessage, err)
		}
		if payload.ListingID == nil || payload.SellerID == nil || payload.PreviousStatus == nil ||
			payload.Status == nil || payload.ChangedAt == nil {
			return fmt.Errorf("%w: incomplete status changed payload", errInvalidListingEventMessage)
		}
		revoked, err := c.handler.StatusChanged(ctx, usecase.ListingStatusChangedCommand{
			Event:          event,
			ListingID:      *payload.ListingID,
			SellerID:       *payload.SellerID,
			PreviousStatus: *payload.PreviousStatus,
			Status:         *payload.Status,
			ChangedAt:      *payload.ChangedAt,
		})
		if err != nil {
			return err
		}
		slog.InfoContext(ctx, "handled listing status change", "listing_id", payload.ListingID, "revoked", revoked)

		return nil
	default:
		return fmt.Errorf("%w: unsupported event type %q", errInvalidListingEventMessage, delivery.Type)
	}
}

func decodeListingEvent(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
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

func (c *Consumer) Close() error {
	return c.channel.Close()
}
