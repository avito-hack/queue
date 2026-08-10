package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) ClearItemQueue(ctx context.Context, itemID uuid.UUID) error {
	return s.txManager.WithinTransaction(ctx, func(ctx context.Context, queueRepository domain.ItemQueueRepository, memberRepository domain.ItemQueueMemberRepository) error {
		exists, err := queueRepository.Exists(ctx, itemID)
		if err != nil {
			s.logger.Error(
				"failed to check queue existence",
				"error",
				err,
				"item_id",
				itemID,
			)

			return err
		}

		if !exists {
			s.logger.Warn(
				"failed to clear queue: queue not found",
				"item_id",
				itemID,
			)

			return ErrQueueNotFound
		}

		if err := memberRepository.DeleteAllByItemID(ctx, itemID); err != nil {
			s.logger.Error(
				"failed to delete queue members",
				"error",
				err,
				"item_id",
				itemID,
			)

			return err
		}

		return nil
	})
}
