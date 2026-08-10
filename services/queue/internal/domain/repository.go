package domain

import (
	"context"

	"github.com/google/uuid"
)

type ItemQueueRepository interface {
	Create(ctx context.Context, queue *ItemQueue) error
	GetByItemID(ctx context.Context, itemID uuid.UUID) (*ItemQueue, error)
	Update(ctx context.Context, queue *ItemQueue) error
	Delete(ctx context.Context, itemID uuid.UUID) error
	Exists(ctx context.Context, itemID uuid.UUID) (bool, error)
	LockByItemID(ctx context.Context, itemID uuid.UUID) (*ItemQueue, error)
}

type ItemQueueMemberRepository interface {
	Create(ctx context.Context, itemID uuid.UUID, member *ItemQueueMember) (*ItemQueueMember, error)
	GetByUserID(ctx context.Context, itemID, userID uuid.UUID) (*ItemQueueMember, error)
	GetByTicketID(ctx context.Context, ticketID uuid.UUID) (*ItemQueueMember, error)
	GetAllByItemID(ctx context.Context, itemID uuid.UUID) ([]*ItemQueueMember, error)
	GetAllByUserID(ctx context.Context, userID uuid.UUID) ([]*ItemQueueMember, error)
	Update(ctx context.Context, itemID uuid.UUID, member *ItemQueueMember) error
	Leave(ctx context.Context, itemID, userID uuid.UUID, reason ItemQueueMemberStatus) error
	Reactivate(ctx context.Context, itemID, userID uuid.UUID, position uint) error
	DeleteAllByItemID(ctx context.Context, itemID uuid.UUID) error
	Exists(ctx context.Context, itemID, userID uuid.UUID) (bool, error)
	GetPosition(ctx context.Context, itemID, userID uuid.UUID) (uint, error)
	ShiftPositionsAfterDelete(ctx context.Context, itemID uuid.UUID, position uint) error
	GetRank(ctx context.Context, itemID, userID uuid.UUID) (uint, error)
}