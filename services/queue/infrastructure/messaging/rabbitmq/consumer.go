package rabbitmq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/avito-hack/queue/services/queue/internal/usecase"
)

var errInvalidQueueEventMessage = errors.New("invalid queue event message")

type Consumer struct {
	channel *amqp.Channel
	queue   string
	handler usecase.QueueEventHandler
}

func NewConsumer(
	connection *amqp.Connection,
	exchange string,
	queue string,
	prefetch int,
	handler usecase.QueueEventHandler,
) (*Consumer, error) {

	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("open consumer channel: %w", err)
	}

	closeOnError := func(err error) (*Consumer, error) {
		_ = channel.Close()
		return nil, err
	}


	if err := channel.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return closeOnError(
			fmt.Errorf("declare exchange: %w", err),
		)
	}


	if _, err := channel.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return closeOnError(
			fmt.Errorf("declare queue: %w", err),
		)
	}


	routingKeys := []string{
		"listing.quantity.changed",
		"listing.status.changed",
		"ticket.closed",
		"ticket.redeemed",
	}


	for _, key := range routingKeys {

		if err := channel.QueueBind(
			queue,
			key,
			exchange,
			false,
			nil,
		); err != nil {
			return closeOnError(
				fmt.Errorf(
					"bind queue to %s: %w",
					key,
					err,
				),
			)
		}
	}


	if err := channel.Qos(
		prefetch,
		0,
		false,
	); err != nil {
		return closeOnError(
			fmt.Errorf("configure qos: %w", err),
		)
	}


	return &Consumer{
		channel: channel,
		queue: queue,
		handler: handler,
	}, nil
}


func (c *Consumer) Run(ctx context.Context) error {

	deliveries, err := c.channel.ConsumeWithContext(
		ctx,
		c.queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf(
			"consume events: %w",
			err,
		)
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

				return errors.New(
					"delivery channel closed",
				)
			}


			if err := c.handle(
				ctx,
				delivery,
			); err != nil {


				slog.ErrorContext(
					ctx,
					"handle rabbit event failed",
					"error",
					err,
				)


				if nackErr := delivery.Nack(
					false,
					true,
				); nackErr != nil {
					return fmt.Errorf(
						"nack message: %w",
						nackErr,
					)
				}

				continue
			}


			if err := delivery.Ack(false); err != nil {
				return fmt.Errorf(
					"ack message: %w",
					err,
				)
			}
		}
	}
}


func (c *Consumer) handle(
	ctx context.Context,
	delivery amqp.Delivery,
) error {


	if delivery.ContentType != "application/json" {
		return fmt.Errorf(
			"%w: invalid content type",
			errInvalidQueueEventMessage,
		)
	}


	switch delivery.Type {


	case "listing.quantity.changed":

		var event usecase.ListingQuantityChangedEvent

		if err := decode(
			delivery.Body,
			&event,
		); err != nil {
			return err
		}


		return c.handler.ListingQuantityChanged(
			ctx,
			event,
		)



	case "listing.status.changed":

		var event usecase.ListingStatusChangedEvent

		if err := decode(
			delivery.Body,
			&event,
		); err != nil {
			return err
		}


		return c.handler.ListingStatusChanged(
			ctx,
			event,
		)



	case "ticket.closed":

		var event usecase.TicketClosedEvent

		if err := decode(
			delivery.Body,
			&event,
		); err != nil {
			return err
		}


		return c.handler.TicketClosed(
			ctx,
			event,
		)



	case "ticket.redeemed":

		var event usecase.TicketRedeemedEvent

		if err := decode(
			delivery.Body,
			&event,
		); err != nil {
			return err
		}


		return c.handler.TicketRedeemed(
			ctx,
			event,
		)


	default:

		return fmt.Errorf(
			"%w: unknown event %s",
			errInvalidQueueEventMessage,
			delivery.Type,
		)
	}
}


func decode(
	body []byte,
	target any,
) error {

	decoder := json.NewDecoder(
		bytes.NewReader(body),
	)

	decoder.DisallowUnknownFields()


	if err := decoder.Decode(target); err != nil {
		return err
	}


	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {

		if err == nil {
			return errors.New(
				"multiple json values",
			)
		}

		return err
	}


	return nil
}


func (c *Consumer) Close() error {
	return c.channel.Close()
}