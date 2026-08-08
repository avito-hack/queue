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
			return err
		}

		if !exists {
			return ErrQueueNotFound
		}

		return memberRepository.DeleteAllByItemID(ctx, itemID)
	})
}