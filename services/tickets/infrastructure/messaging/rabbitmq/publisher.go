package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

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
			Body:         event.Payload,
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

func (p *Publisher) Close() error {
	return p.channel.Close()
}
