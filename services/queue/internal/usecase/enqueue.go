package usecase

import (
	"context"
	"errors"
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


	var (
		memberID uuid.UUID
		position uint
		needTicket bool
	)


	err = s.txManager.WithinTransaction(
		ctx,
		func(
			ctx context.Context,
			queueRepository domain.ItemQueueRepository,
			memberRepository domain.ItemQueueMemberRepository,
		) error {


			_, err := queueRepository.LockByItemID(
				ctx,
				itemID,
			)

			if err != nil {

				if !errors.Is(
					err,
					domain.ErrQueueNotFound,
				) {
					return err
				}


				err = queueRepository.Create(
					ctx,
					&domain.ItemQueue{
						ItemID: itemID,
						State: domain.QueueTicketsAvailable,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				)

				if err != nil &&
					!errors.Is(
						err,
						domain.ErrAlreadyExists,
					) {

					return err
				}


				_, err = queueRepository.LockByItemID(
					ctx,
					itemID,
				)

				if err != nil {
					return err
				}
			}


			existing, err := memberRepository.GetByUserID(
				ctx,
				itemID,
				userID,
			)

			if err != nil &&
				!errors.Is(
					err,
					domain.ErrMemberNotFound,
				) {
				return err
			}


			if existing != nil &&
				domain.IsActiveMemberStatus(existing.Status) {

				return domain.ErrAlreadyExists
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


			needTicket = activeTickets < listing.Quantity



			position = 1

			for _, member := range members {

				if member.Position != nil &&
					*member.Position >= position {

					position = *member.Position + 1
				}
			}



			var member *domain.ItemQueueMember


			if existing == nil {

				member, err = memberRepository.Create(
					ctx,
					itemID,
					&domain.ItemQueueMember{
						ItemID: itemID,
						UserID: userID,
						Position: &position,
						Status: domain.UserWaitingInLine,
						CreatedAt: time.Now(),
					},
				)

				if err != nil {
					return err
				}

			} else {

				err = memberRepository.Reactivate(
					ctx,
					itemID,
					userID,
					position,
				)

				if err != nil {
					return err
				}

				member = existing
				member.Position = &position
				member.Status = domain.UserWaitingInLine
				member.TicketID = nil
			}


			memberID = member.ID


			s.logger.Info(
				"queue member created",
				"member_id",
				memberID,
				"item_id",
				itemID,
				"user_id",
				userID,
				"position",
				position,
				"need_ticket",
				needTicket,
				"active_tickets",
				activeTickets,
				"quantity",
				listing.Quantity,
			)


			return nil
		},
	)


	if err != nil {
		return err
	}



	if !needTicket {

		s.logger.Info(
			"user added to queue without ticket",
			"item_id",
			itemID,
			"user_id",
			userID,
			"queue_entry_id",
			memberID,
		)

		return nil
	}



	s.logger.Info(
		"requesting ticket",
		"item_id",
		itemID,
		"user_id",
		userID,
		"queue_entry_id",
		memberID,
	)



	ticket, err := s.ticketsClient.IssueTicket(
		ctx,
		itemID,
		memberID,
		itemID,
		userID,
	)


	if err != nil {

		s.logger.Warn(
			"ticket issue failed",
			"error",
			err,
			"item_id",
			itemID,
			"user_id",
			userID,
		)

		return nil
	}



	ticketID, err := uuid.Parse(
		ticket.ID,
	)


	if err != nil {

		s.logger.Error(
			"failed to parse ticket uuid",
			"ticket_id",
			ticket.ID,
			"error",
			err,
		)

		return nil
	}



	err = s.txManager.WithinTransaction(
		ctx,
		func(
			ctx context.Context,
			_ domain.ItemQueueRepository,
			memberRepository domain.ItemQueueMemberRepository,
		) error {


			member, err := memberRepository.GetByUserID(
				ctx,
				itemID,
				userID,
			)

			if err != nil {
				return err
			}


			member.TicketID = &ticketID
			member.Status = domain.UserAcquiredPurchaseRights


			return memberRepository.Update(
				ctx,
				itemID,
				member,
			)
		},
	)


	if err != nil {
		return err
	}


	s.logger.Info(
		"ticket issued successfully",
		"item_id",
		itemID,
		"user_id",
		userID,
		"ticket_id",
		ticketID,
	)


	return nil
}