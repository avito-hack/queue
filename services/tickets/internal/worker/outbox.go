package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/avito-hack/queue/services/tickets/internal/usecase"
	"github.com/avito-hack/queue/services/tickets/pkg/workerpool"
)

type Outbox struct {
	repository  usecase.OutboxRepository
	publisher   usecase.EventPublisher
	interval    time.Duration
	lease       time.Duration
	retryDelay  time.Duration
	concurrency int
	batchSize   int
	clock       usecase.Clock
}

func NewOutbox(
	repository usecase.OutboxRepository,
	publisher usecase.EventPublisher,
	interval time.Duration,
	lease time.Duration,
	retryDelay time.Duration,
	concurrency int,
	batchSize int,
	clocks ...usecase.Clock,
) *Outbox {
	clock := usecase.Clock(time.Now)
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}

	return &Outbox{
		repository:  repository,
		publisher:   publisher,
		interval:    interval,
		lease:       lease,
		retryDelay:  retryDelay,
		concurrency: concurrency,
		batchSize:   batchSize,
		clock:       clock,
	}
}

func (w *Outbox) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		if err := w.RunOnce(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "publish outbox", "error", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (w *Outbox) RunOnce(ctx context.Context) error {
	if w.concurrency <= 0 || w.batchSize <= 0 || w.lease <= 0 || w.retryDelay <= 0 {
		return fmt.Errorf("invalid outbox worker configuration")
	}
	events, err := w.repository.Claim(ctx, w.clock(), w.batchSize, w.lease)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	pool := workerpool.New[usecase.OutboxEvent, usecase.OutboxEvent](w.concurrency, w.deliver)
	submission := make(chan error, 1)
	go func() {
		var submitErr error
		for _, event := range events {
			if err := pool.Submit(ctx, event); err != nil {
				submitErr = fmt.Errorf("submit outbox event: %w", err)
				break
			}
		}
		shutdownContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := pool.Shutdown(shutdownContext); err != nil {
			submitErr = errors.Join(submitErr, fmt.Errorf("shutdown outbox worker pool: %w", err))
		}
		submission <- submitErr
	}()

	var resultErr error
	for result := range pool.Results() {
		resultErr = errors.Join(resultErr, result.Err)
	}

	return errors.Join(resultErr, <-submission)
}

func (w *Outbox) deliver(ctx context.Context, event usecase.OutboxEvent) (usecase.OutboxEvent, error) {
	if err := w.publisher.Publish(ctx, event); err != nil {
		retryContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if retryErr := w.repository.Retry(retryContext, event.ID, w.clock().Add(w.retryDelay)); retryErr != nil {
			return event, errors.Join(err, retryErr)
		}

		return event, err
	}
	if err := w.repository.MarkPublished(ctx, event.ID, w.clock()); err != nil {
		return event, err
	}

	return event, nil
}
