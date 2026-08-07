package domain

import (
	"time"

	"github.com/google/uuid"
)

type ItemQueueMemberStatus string

const (
	UserWaitingInLine          ItemQueueMemberStatus = "waiting_in_line"
	UserAcquiredPurchaseRights ItemQueueMemberStatus = "acquired_purchase_rights"

	UserPlacedAnOrder   ItemQueueMemberStatus = "placed_an_order"
	UserPurchasedAnItem ItemQueueMemberStatus = "purchased_an_item"

	UserVoluntarilyLeftLine ItemQueueMemberStatus = "voluntarily_left_the_line"
	UserGivenUpRights       ItemQueueMemberStatus = "given_up_purchase_rights"
	UserLostRights          ItemQueueMemberStatus = "lost_purchase_rights"
	UserItemOutOfStock      ItemQueueMemberStatus = "item_out_of_stock"
)

type ItemQueueState string

const (
	QueueTicketsAvailable       ItemQueueState = "tickets_available"
	QueueTicketsPartiallyIssued ItemQueueState = "tickets_partially_issued"
	QueueTicketsExhausted       ItemQueueState = "tickets_exhausted"
)

type ItemQueue struct {
	ItemID    uuid.UUID
	State     ItemQueueState
	CreatedAt time.Time
	UpdatedAt time.Time
	Members   []*ItemQueueMember
}

type ItemQueueMember struct {
	ItemID  uuid.UUID	
	UserID    uuid.UUID
	Position  uint	
	Status    ItemQueueMemberStatus
	CreatedAt time.Time
}

type UserQueueInfo struct {
    ItemID   uuid.UUID
    Position int
    Status   ItemQueueMemberStatus
}