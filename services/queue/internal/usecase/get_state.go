package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) GetItemQueueState(ctx context.Context, itemID uuid.UUID) (domain.ItemQueueState, error) {
	queue, err := s.queueRepository.GetByItemID(ctx, itemID)
	if err != nil {
		return "", ErrQueueNotFound
	}

	return queue.State, nil
}