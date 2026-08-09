package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) Enqueue(ctx context.Context, itemID, userID uuid.UUID) error {
	if err := s.ensureItemAvailable(ctx, itemID); err != nil {
		s.logger.Warn(
			"enqueue failed: item unavailable",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
		)

		return err
	}

	if err := s.ensureUserHasNoActiveTicket(ctx, itemID); err != nil {
		s.logger.Warn(
			"enqueue failed: user has active ticket",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
		)

		return err
	}

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
			err = queueRepository.Create(ctx, &domain.ItemQueue{
				ItemID:    itemID,
				State:     domain.QueueTicketsAvailable,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})

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
		}

		exists, err = memberRepository.Exists(ctx, itemID, userID)
		if err != nil {
			s.logger.Error(
				"failed to check queue member existence",
				"error",
				err,
				"item_id",
				itemID,
				"user_id",
				userID,
			)

			return err
		}

		if exists {
			s.logger.Warn(
				"enqueue failed: user already in queue",
				"item_id",
				itemID,
				"user_id",
				userID,
			)

			return ErrUserAlreadyInQueue
		}

		count, err := memberRepository.Count(ctx, itemID)
		if err != nil {
			s.logger.Error(
				"failed to count queue members",
				"error",
				err,
				"item_id",
				itemID,
			)

			return err
		}

		err = memberRepository.Create(ctx, itemID, &domain.ItemQueueMember{
			ItemID:    itemID,
			UserID:    userID,
			Position:  uint(count + 1),
			Status:    domain.UserWaitingInLine,
			CreatedAt: time.Now(),
		})

		if err != nil {
			s.logger.Error(
				"failed to create queue member",
				"error",
				err,
				"item_id",
				itemID,
				"user_id",
				userID,
			)

			return err
		}

		return nil
	})
}
