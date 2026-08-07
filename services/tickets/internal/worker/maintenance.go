package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type Maintenance struct {
	useCase  *usecase.MaintainTickets
	interval time.Duration
}

func NewMaintenance(useCase *usecase.MaintainTickets, interval time.Duration) *Maintenance {
	return &Maintenance{useCase: useCase, interval: interval}
}

func (w *Maintenance) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		result, err := w.useCase.Run(ctx)
		if err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "maintain tickets", "error", err)
		} else if result.Expired > 0 || result.RecoveredActivations > 0 {
			slog.InfoContext(
				ctx,
				"tickets maintained",
				"expired",
				result.Expired,
				"recovered_activations",
				result.RecoveredActivations,
			)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
