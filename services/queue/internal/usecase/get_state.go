package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) GetItemQueueState(ctx context.Context, itemID uuid.UUID) (domain.ItemQueueStateInfo, error) {
	queue, err := s.queueRepository.GetByItemID(ctx, itemID)
	if err != nil {
		s.logger.Warn(
			"failed to get item queue state",
			"error",
			err,
			"item_id",
			itemID,
		)

		return domain.ItemQueueStateInfo{}, ErrQueueNotFound
	}

	count, err := s.memberRepository.Count(ctx, itemID)
	if err != nil {
		s.logger.Error(
			"failed to count queue members",
			"error",
			err,
			"item_id",
			itemID,
		)

		return domain.ItemQueueStateInfo{}, err
	}

	return domain.ItemQueueStateInfo{
		State:        queue.State,
		WaitingCount: count,
	}, nil
}
