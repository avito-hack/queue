package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/avito-hack/queue/services/queue/internal/domain"
	
)

func (s *itemQueueService) Dequeue(ctx context.Context, itemID, userID uuid.UUID) error {
	return s.txManager.WithinTransaction(ctx, func(ctx context.Context, queueRepository domain.ItemQueueRepository, memberRepository domain.ItemQueueMemberRepository) error {
		member, err := memberRepository.GetByUserID(ctx, itemID, userID)
		if err != nil {
			return ErrUserNotInQueue
		}

		if err := memberRepository.Delete(ctx, itemID, userID); err != nil {
			return err
		}

		return memberRepository.ShiftPositionsAfterDelete(ctx, itemID, member.Position)
	})
}