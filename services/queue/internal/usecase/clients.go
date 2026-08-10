package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

type AvitoClient interface {
	GetListing(
		ctx context.Context,
		itemID uuid.UUID,
	) (*domain.Listing, error)
}

type TicketsClient interface {
	IssueTicket(
		ctx context.Context,
		listingID uuid.UUID,
		queueEntryID uuid.UUID,
		skuID uuid.UUID,
		userID uuid.UUID,
	) (*domain.Ticket, error)

	DeclineTicket(
		ctx context.Context,
		ticketID uuid.UUID,
	) error	
}