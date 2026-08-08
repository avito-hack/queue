package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type maintenanceRepositoryStub struct {
	expired             int
	recovered           int
	receivedExpiration  time.Time
	receivedStaleBefore time.Time
	receivedLimit       int
}

func (s *maintenanceRepositoryStub) ExpireIssued(_ context.Context, now time.Time, limit int) (int, error) {
	s.receivedExpiration = now
	s.receivedLimit = limit

	return s.expired, nil
}

func (s *maintenanceRepositoryStub) RecoverStaleActivations(
	_ context.Context,
	staleBefore time.Time,
	limit int,
) (int, error) {
	s.receivedStaleBefore = staleBefore
	s.receivedLimit = limit

	return s.recovered, nil
}

func Test_MaintainTickets_Run_ExpiredAndStaleTickets_ProcessBatch(t *testing.T) {
	// given
	now := time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC)
	repository := &maintenanceRepositoryStub{expired: 3, recovered: 2}
	useCase := NewMaintainTickets(repository, 100, time.Minute, func() time.Time {
		return now
	})

	// when
	result, err := useCase.Run(context.Background())

	// then
	require.NoError(t, err)
	require.Equal(t, TicketMaintenanceResult{Expired: 3, RecoveredActivations: 2}, result)
	require.Equal(t, now, repository.receivedExpiration)
	require.Equal(t, now.Add(-time.Minute), repository.receivedStaleBefore)
	require.Equal(t, 100, repository.receivedLimit)
}
