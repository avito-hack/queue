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

			if _, err := queueRepository.LockByItemID(
				ctx,
				itemID,
			); err != nil {
				return err
			}

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

			if !domain.IsActiveMemberStatus(member.Status) {
				s.logger.Warn(
					"user cannot leave queue",
					"item_id",
					itemID,
					"user_id",
					userID,
					"status",
					member.Status,
				)

				return ErrUserCannotLeaveQueue
			}

			position := member.Position

			err = memberRepository.Leave(
				ctx,
				itemID,
				userID,
				domain.UserVoluntarilyLeftLine,
			)

			if err != nil {
				s.logger.Error(
					"failed to leave queue",
					"error",
					err,
					"item_id",
					itemID,
					"user_id",
					userID,
				)

				return err
			}

			if position != nil {
				err = memberRepository.ShiftPositionsAfterDelete(
					ctx,
					itemID,
					*position,
				)

				if err != nil {
					s.logger.Error(
						"failed to shift queue positions",
						"error",
						err,
						"item_id",
						itemID,
						"position",
						*position,
					)

					return err
				}
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