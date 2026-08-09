package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) Enqueue(
	ctx context.Context,
	itemID uuid.UUID,
	userID uuid.UUID,
) error {
	listing, err := s.avitoClient.GetListing(
		ctx,
		itemID,
	)

	if err != nil {
		s.logger.Error(
			"failed to get listing",
			"error",
			err,
			"item_id",
			itemID,
		)

		return fmt.Errorf(
			"get listing: %w",
			err,
		)
	}

	if !listing.QueueEnabled ||
		listing.Status != "active" ||
		listing.Quantity <= 0 {
		return ErrQueueUnavailable
	}

	return s.txManager.WithinTransaction(
		ctx,
		func(
			ctx context.Context,
			queueRepository domain.ItemQueueRepository,
			memberRepository domain.ItemQueueMemberRepository,
		) error {

			exists, err := memberRepository.Exists(
				ctx,
				itemID,
				userID,
			)

			if err != nil {
				return err
			}

			if exists {
				return ErrUserAlreadyInQueue
			}

			err = queueRepository.Create(
				ctx,
				&domain.ItemQueue{
					ItemID:    itemID,
					State:     domain.QueueTicketsAvailable,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			)

			if err != nil {
				return err
			}

			err = queueRepository.Lock(
				ctx,
				itemID,
			)

			if err != nil {
				return err
			}

			members, err := memberRepository.GetAllByItemID(
				ctx,
				itemID,
			)

			if err != nil {
				return err
			}

			activeTickets := 0

			for _, member := range members {
				if member.Status == domain.UserAcquiredPurchaseRights {
					activeTickets++
				}
			}

			count, err := memberRepository.Count(
				ctx,
				itemID,
			)

			if err != nil {
				return err
			}

			member, err := memberRepository.Create(
				ctx,
				itemID,
				&domain.ItemQueueMember{
					ItemID:    itemID,
					UserID:    userID,
					Position:  uint(count + 1),
					Status:    domain.UserWaitingInLine,
					CreatedAt: time.Now(),
				},
			)

			if err != nil {
				return err
			}

			if activeTickets >= listing.Quantity {
				s.logger.Info(
					"user added without ticket",
					"item_id",
					itemID,
					"user_id",
					userID,
					"position",
					member.Position,
				)

				return nil
			}

			ticket, err := s.ticketsClient.IssueTicket(
				ctx,
				itemID,
				member.ID,
				itemID,
				userID,
			)

			if err != nil {
				return fmt.Errorf(
					"issue ticket: %w",
					err,
				)
			}

			ticketID, err := uuid.Parse(ticket.ID)

			if err != nil {
				return fmt.Errorf(
					"parse ticket id: %w",
					err,
				)
			}

			member.TicketID = ticketID
			member.Status = domain.UserAcquiredPurchaseRights

			if err := memberRepository.Update(
				ctx,
				itemID,
				member,
			); err != nil {
				return err
			}

			s.logger.Info(
				"ticket issued",
				"item_id",
				itemID,
				"user_id",
				userID,
				"ticket_id",
				member.TicketID,
				"queue_entry_id",
				member.ID,
			)

			return nil
		},
	)
}