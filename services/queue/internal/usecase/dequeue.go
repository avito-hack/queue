package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) Dequeue(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
) error {

	return s.txManager.WithinTransaction(
		ctx,
		func(
			ctx context.Context,
			queueRepository domain.ItemQueueRepository,
			memberRepository domain.ItemQueueMemberRepository,
		) error {

			member, err := memberRepository.GetByUserID(
				ctx,
				itemID,
				userID,
			)

			if err != nil {
				s.logger.Warn(
					"user not found in queue",
					"error",
					err,
					"item_id",
					itemID,
					"user_id",
					userID,
				)

				return ErrUserNotInQueue
			}

			err = queueRepository.Lock(
				ctx,
				itemID,
			)

			if err != nil {
				return err
			}

			if err := memberRepository.Delete(
				ctx,
				itemID,
				userID,
			); err != nil {
				s.logger.Error(
					"failed to delete queue member",
					"error",
					err,
					"item_id",
					itemID,
					"user_id",
					userID,
				)

				return err
			}

			if err := memberRepository.ShiftPositionsAfterDelete(
				ctx,
				itemID,
				member.Position,
			); err != nil {
				s.logger.Error(
					"failed to shift queue positions",
					"error",
					err,
					"item_id",
					itemID,
					"position",
					member.Position,
				)

				return err
			}

			s.logger.Info(
				"user left queue",
				"item_id",
				itemID,
				"user_id",
				userID,
				"ticket_id",
				member.TicketID,
			)

			return nil
		},
	)
}