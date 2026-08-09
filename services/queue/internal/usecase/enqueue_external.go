package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

func (s *itemQueueService) ensureItemAvailable(
	ctx context.Context,
	itemID uuid.UUID,
) error {

	listing, err := s.avitoClient.GetListing(
		ctx,
		itemID,
	)

	if err != nil {
		s.logger.Error(
			"failed to get listing from avito",
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

	if !listing.QueueEnabled {
		s.logger.Warn(
			"queue disabled for listing",
			"item_id",
			itemID,
		)

		return ErrQueueUnavailable
	}

	if listing.Status != "active" {
		s.logger.Warn(
			"listing inactive",
			"item_id",
			itemID,
			"status",
			listing.Status,
		)

		return ErrQueueUnavailable
	}

	if listing.Quantity <= 0 {
		s.logger.Warn(
			"listing has no available quantity",
			"item_id",
			itemID,
			"quantity",
			listing.Quantity,
		)

		return ErrQueueUnavailable
	}

	return nil
}

func (s *itemQueueService) ensureUserHasNoActiveTicket(
	ctx context.Context,
	itemID uuid.UUID,
) error {

	tickets, err := s.ticketsClient.GetTickets(
		ctx,
		itemID,
	)

	if err != nil {
		s.logger.Error(
			"failed to get tickets",
			"error",
			err,
			"item_id",
			itemID,
		)

		return fmt.Errorf(
			"get tickets: %w",
			err,
		)
	}

	for _, ticket := range tickets {

		switch ticket.Status {

		case domain.TicketIssued,
			domain.TicketRedeemed:

			s.logger.Warn(
				"user has active ticket",
				"item_id",
				itemID,
				"ticket_id",
				ticket.ID,
				"ticket_status",
				ticket.Status,
			)

			return ErrUserHasActiveTicket
		}
	}

	return nil
}
