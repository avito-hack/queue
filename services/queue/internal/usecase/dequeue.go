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

	var ticketID *uuid.UUID

	err := s.txManager.WithinTransaction(
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

			if !domain.IsActiveMemberStatus(
				member.Status,
			) {
				return ErrUserCannotLeaveQueue
			}

			if member.TicketID != nil {
				id := *member.TicketID
				ticketID = &id
			}

			position := member.Position

			err = memberRepository.Leave(
				ctx,
				itemID,
				userID,
				domain.UserVoluntarilyLeftLine,
			)

			if err != nil {
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

			return nil
		},
	)

	if err != nil {
		return err
	}

	if ticketID != nil {
		err := s.ticketsClient.DeclineTicket(
			ctx,
			*ticketID,
		)

		if err != nil {
			s.logger.Error(
				"failed to decline ticket",
				"error",
				err,
				"ticket_id",
				*ticketID,
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
		ticketID,
	)

	return nil
}