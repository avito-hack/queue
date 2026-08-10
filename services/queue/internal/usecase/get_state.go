package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) GetItemQueueState(
	ctx context.Context,
	itemID uuid.UUID,
) (domain.ItemQueueState, error) {

	s.logger.Info(
		"get queue state started",
		"item_id",
		itemID,
	)

	queue, err := s.queueRepository.GetByItemID(
		ctx,
		itemID,
	)

	if err != nil {
		s.logger.Warn(
			"queue not found",
			"error",
			err,
			"item_id",
			itemID,
		)

		return "", ErrQueueNotFound
	}

	s.logger.Info(
		"get queue state completed",
		"item_id",
		itemID,
		"state",
		queue.State,
	)

	return queue.State, nil
}