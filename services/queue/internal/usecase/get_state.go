package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) GetItemQueueState(ctx context.Context, itemID uuid.UUID) (domain.ItemQueueState, error) {
	queue, err := s.queueRepository.GetByItemID(ctx, itemID)
	if err != nil {
		s.logger.Warn(
			"failed to get item queue state",
			"error",
			err,
			"item_id",
			itemID,
		)

		return "", ErrQueueNotFound
	}

	return queue.State, nil
}