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
	GetTickets(
		ctx context.Context,
		itemID uuid.UUID,
	) ([]*domain.Ticket, error)
}