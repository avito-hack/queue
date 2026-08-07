package usecase

import (
	"context"
	"fmt"
	"time"
)

type TicketMaintenanceRepository interface {
	ExpireIssued(context.Context, time.Time, int) (int, error)
	RecoverStaleActivations(context.Context, time.Time, int) (int, error)
}

type TicketMaintenanceResult struct {
	Expired              int
	RecoveredActivations int
}

type MaintainTickets struct {
	repository        TicketMaintenanceRepository
	batchSize         int
	activationTimeout time.Duration
	clock             Clock
}

func NewMaintainTickets(
	repository TicketMaintenanceRepository,
	batchSize int,
	activationTimeout time.Duration,
	clocks ...Clock,
) *MaintainTickets {
	clock := Clock(time.Now)
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}

	return &MaintainTickets{
		repository:        repository,
		batchSize:         batchSize,
		activationTimeout: activationTimeout,
		clock:             clock,
	}
}

func (u *MaintainTickets) Run(ctx context.Context) (TicketMaintenanceResult, error) {
	if u.batchSize <= 0 {
		return TicketMaintenanceResult{}, fmt.Errorf("maintain tickets: batch size must be positive")
	}
	if u.activationTimeout <= 0 {
		return TicketMaintenanceResult{}, fmt.Errorf("maintain tickets: activation timeout must be positive")
	}

	now := u.clock()
	recovered, err := u.repository.RecoverStaleActivations(ctx, now.Add(-u.activationTimeout), u.batchSize)
	if err != nil {
		return TicketMaintenanceResult{}, fmt.Errorf("recover stale activations: %w", err)
	}
	expired, err := u.repository.ExpireIssued(ctx, now, u.batchSize)
	if err != nil {
		return TicketMaintenanceResult{}, fmt.Errorf("expire issued tickets: %w", err)
	}

	return TicketMaintenanceResult{Expired: expired, RecoveredActivations: recovered}, nil
}
