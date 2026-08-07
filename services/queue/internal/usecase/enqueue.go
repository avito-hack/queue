package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) Enqueue(ctx context.Context, itemID, userID uuid.UUID) error {
    if err := s.ensureItemAvailable(ctx, itemID); err != nil {
        return err
    }

    if err := s.ensureUserHasNoActiveTicket(ctx, itemID); err != nil {
        return err
    }

    return s.txManager.WithinTransaction(ctx, func(ctx context.Context, queueRepository domain.ItemQueueRepository, memberRepository domain.ItemQueueMemberRepository) error {
        exists, err := queueRepository.Exists(ctx, itemID)
        if err != nil {
            return err
        }

        if !exists {
            err = queueRepository.Create(ctx, &domain.ItemQueue{
                ItemID:    itemID,
                State:     domain.QueueTicketsAvailable,
                CreatedAt: time.Now(),
                UpdatedAt: time.Now(),
            })
            if err != nil {
                return err
            }
        }

        exists, err = memberRepository.Exists(ctx, itemID, userID)
        if err != nil {
            return err
        }

        if exists {
            return ErrUserAlreadyInQueue
        }

        count, err := memberRepository.Count(ctx, itemID)
        if err != nil {
            return err
        }

        return memberRepository.Create(ctx, itemID, &domain.ItemQueueMember{
            ItemID:    itemID,
            UserID:    userID,
            Position:  uint(count + 1),
            Status:    domain.UserWaitingInLine,
            CreatedAt: time.Now(),
        })
    })
}
