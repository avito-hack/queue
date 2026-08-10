package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/internal/domain"
)

type ItemQueueService interface {
	Enqueue(ctx context.Context, itemID, userID uuid.UUID) error
	Dequeue(ctx context.Context, itemID, userID uuid.UUID) error
	ClearItemQueue(ctx context.Context, itemID uuid.UUID) error
	
	
	GetUserPosition(ctx context.Context, itemID, userID uuid.UUID) (uint, error)
	GetItemQueueState(ctx context.Context, itemID uuid.UUID) (domain.ItemQueueState, error)
	GetUserQueues(ctx context.Context, userID uuid.UUID) ([]*domain.UserQueueInfo, error)

	HandleListingQuantityChanged(
        ctx context.Context,
        event ListingQuantityChangedEvent,
    ) error

    HandleListingStatusChanged(
        ctx context.Context,
        event ListingStatusChangedEvent,
    ) error

    HandleTicketClosed(
        ctx context.Context,
        event TicketClosedEvent,
    ) error

    HandleTicketRedeemed(
        ctx context.Context,
        event TicketRedeemedEvent,
    ) error

}
