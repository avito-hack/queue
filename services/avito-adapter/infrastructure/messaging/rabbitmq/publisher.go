package rabbitmq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
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
		return nil, fmt.Errorf("declare exchange: %w", err)
	}
	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("enable confirms: %w", err)
	}
	return &Publisher{channel: channel, exchange: exchange}, nil
}

func (p *Publisher) Publish(ctx context.Context, id, eventType string, payload []byte) error {
	confirmation, err := p.channel.PublishWithDeferredConfirmWithContext(ctx, p.exchange, eventType, false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: id, Type: eventType, AppId: "avito-adapter", Timestamp: time.Now(), Body: payload})
	if err != nil {
		return fmt.Errorf("publish %s: %w", eventType, err)
	}
	acknowledged, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("wait confirmation: %w", err)
	}
	if !acknowledged {
		return fmt.Errorf("publish %s: rejected", eventType)
	}
	return nil
}

func (p *Publisher) Close() error { return p.channel.Close() }
