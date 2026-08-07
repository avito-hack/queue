package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/avito-hack/queue/services/tickets/internal/usecase"
	"github.com/avito-hack/queue/services/tickets/pkg/workerpool"
)

var lifecycleBindings = []string{
	string(usecase.LifecycleEventPaymentSucceeded),
	string(usecase.LifecycleEventReservationReleased),
	string(usecase.LifecycleEventListingClosed),
	string(usecase.LifecycleEventSKUClosed),
	string(usecase.LifecycleEventSystemCancelled),
}

type LifecycleConsumer struct {
	channel     *amqp.Channel
	queue       string
	concurrency int
	processor   *usecase.ProcessLifecycleEvent
}

func NewLifecycleConsumer(
	connection *amqp.Connection,
	exchange string,
	queueName string,
	concurrency int,
	processor *usecase.ProcessLifecycleEvent,
) (*LifecycleConsumer, error) {
	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("open consumer channel: %w", err)
	}
	closeWithError := func(err error) (*LifecycleConsumer, error) {
		_ = channel.Close()
		return nil, err
	}
	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return closeWithError(fmt.Errorf("declare event exchange: %w", err))
	}
	queue, err := channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return closeWithError(fmt.Errorf("declare lifecycle queue: %w", err))
	}
	for _, binding := range lifecycleBindings {
		if err := channel.QueueBind(queue.Name, binding, exchange, false, nil); err != nil {
			return closeWithError(fmt.Errorf("bind lifecycle event %s: %w", binding, err))
		}
	}
	if err := channel.Qos(concurrency, 0, false); err != nil {
		return closeWithError(fmt.Errorf("configure lifecycle consumer QoS: %w", err))
	}

	return &LifecycleConsumer{
		channel:     channel,
		queue:       queue.Name,
		concurrency: concurrency,
		processor:   processor,
	}, nil
}

func (c *LifecycleConsumer) Run(ctx context.Context) error {
	deliveries, err := c.channel.ConsumeWithContext(ctx, c.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume lifecycle events: %w", err)
	}
	pool := workerpool.New[amqp.Delivery, struct{}](c.concurrency, c.handle)
	done := make(chan error, 1)
	go func() {
		var submitErr error
		for delivery := range deliveries {
			if err := pool.Submit(ctx, delivery); err != nil {
				submitErr = fmt.Errorf("submit lifecycle event: %w", err)
				break
			}
		}
		shutdownContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := pool.Shutdown(shutdownContext); err != nil {
			submitErr = errors.Join(submitErr, fmt.Errorf("shutdown lifecycle worker pool: %w", err))
		}
		done <- submitErr
	}()

	for result := range pool.Results() {
		if result.Err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "process lifecycle message", "error", result.Err)
		}
	}

	return <-done
}

func (c *LifecycleConsumer) Close() error {
	return c.channel.Close()
}

func (c *LifecycleConsumer) handle(ctx context.Context, delivery amqp.Delivery) (struct{}, error) {
	event, err := lifecycleEventFromDelivery(delivery)
	if err == nil {
		err = c.processor.Process(ctx, event)
	}
	if err == nil {
		return struct{}{}, delivery.Ack(false)
	}
	if errors.Is(err, usecase.ErrInvalidLifecycleEvent) {
		return struct{}{}, errors.Join(err, delivery.Reject(false))
	}

	return struct{}{}, errors.Join(err, delivery.Nack(false, true))
}

type lifecyclePayload struct {
	TicketID   uuid.UUID `json:"ticket_id"`
	OrderID    uuid.UUID `json:"order_id"`
	ListingID  uuid.UUID `json:"listing_id"`
	SKUID      uuid.UUID `json:"sku_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

func lifecycleEventFromDelivery(delivery amqp.Delivery) (usecase.LifecycleEvent, error) {
	eventID, err := uuid.Parse(delivery.MessageId)
	if err != nil {
		return usecase.LifecycleEvent{}, fmt.Errorf("%w: parse event id: %v", usecase.ErrInvalidLifecycleEvent, err)
	}
	var payload lifecyclePayload
	if err := json.Unmarshal(delivery.Body, &payload); err != nil {
		return usecase.LifecycleEvent{}, fmt.Errorf("%w: decode payload: %v", usecase.ErrInvalidLifecycleEvent, err)
	}
	occurredAt := delivery.Timestamp
	if occurredAt.IsZero() {
		occurredAt = payload.OccurredAt
	}

	return usecase.LifecycleEvent{
		ID:         eventID,
		Type:       usecase.LifecycleEventType(delivery.Type),
		Source:     delivery.AppId,
		OccurredAt: occurredAt,
		TicketID:   payload.TicketID,
		OrderID:    payload.OrderID,
		ListingID:  payload.ListingID,
		SKUID:      payload.SKUID,
		Payload:    delivery.Body,
	}, nil
}
