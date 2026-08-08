package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type outboxRepositoryStub struct {
	events    []usecase.OutboxEvent
	claimErr  error
	mutex     sync.Mutex
	published []uuid.UUID
	retried   []uuid.UUID
}

func (s *outboxRepositoryStub) Claim(
	context.Context,
	time.Time,
	int,
	time.Duration,
) ([]usecase.OutboxEvent, error) {
	return s.events, s.claimErr
}

func (s *outboxRepositoryStub) MarkPublished(_ context.Context, eventID uuid.UUID, _ time.Time) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.published = append(s.published, eventID)

	return nil
}

func (s *outboxRepositoryStub) Retry(_ context.Context, eventID uuid.UUID, _ time.Time) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.retried = append(s.retried, eventID)

	return nil
}

type eventPublisherStub struct {
	failingID uuid.UUID
	mutex     sync.Mutex
	published []uuid.UUID
}

func (s *eventPublisherStub) Publish(_ context.Context, event usecase.OutboxEvent) error {
	if event.ID == s.failingID {
		return errors.New("publish failed")
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.published = append(s.published, event.ID)

	return nil
}

func Test_Outbox_RunOnce_PendingEvents_PublishConcurrently(t *testing.T) {
	// given
	events := []usecase.OutboxEvent{{ID: uuid.New()}, {ID: uuid.New()}, {ID: uuid.New()}}
	repository := &outboxRepositoryStub{events: events}
	publisher := &eventPublisherStub{}
	now := time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC)
	worker := NewOutbox(repository, publisher, time.Second, time.Minute, time.Second, 2, 10, func() time.Time {
		return now
	})

	// when
	err := worker.RunOnce(context.Background())

	// then
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{events[0].ID, events[1].ID, events[2].ID}, publisher.published)
	require.ElementsMatch(t, publisher.published, repository.published)
	require.Empty(t, repository.retried)
}

func Test_Outbox_RunOnce_PublishFails_ScheduleRetry(t *testing.T) {
	// given
	event := usecase.OutboxEvent{ID: uuid.New()}
	repository := &outboxRepositoryStub{events: []usecase.OutboxEvent{event}}
	publisher := &eventPublisherStub{failingID: event.ID}
	worker := NewOutbox(repository, publisher, time.Second, time.Minute, time.Second, 1, 10, time.Now)

	// when
	err := worker.RunOnce(context.Background())

	// then
	require.EqualError(t, err, "publish failed")
	require.Empty(t, repository.published)
	require.Equal(t, []uuid.UUID{event.ID}, repository.retried)
}
