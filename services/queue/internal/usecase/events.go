package usecase

import (
	"time"

	"github.com/google/uuid"
)

type ListingQuantityChangedEvent struct {
	ListingID        uuid.UUID `json:"listing_id"`
	SellerID         uuid.UUID `json:"seller_id"`
	PreviousQuantity int       `json:"previous_quantity"`
	Quantity         int       `json:"quantity"`
	ChangedAt        time.Time `json:"changed_at"`
}

type ListingStatusChangedEvent struct {
	ListingID      uuid.UUID `json:"listing_id"`
	SellerID       uuid.UUID `json:"seller_id"`
	PreviousStatus string    `json:"previous_status"`
	Status         string    `json:"status"`
	ChangedAt      time.Time `json:"changed_at"`
}

type TicketClosedEvent struct {
	TicketID     uuid.UUID `json:"ticket_id"`
	QueueEntryID uuid.UUID `json:"queue_entry_id"`
	UserID       uuid.UUID `json:"user_id"`
	ListingID    uuid.UUID `json:"listing_id"`
	SKUId        uuid.UUID `json:"sku_id"`
	Status       string    `json:"status"`
	CloseReason  string    `json:"close_reason"`
	FinishedAt   time.Time `json:"finished_at"`
}

type TicketRedeemedEvent struct {
	TicketID    uuid.UUID `json:"ticket_id"`
	UserID      uuid.UUID `json:"user_id"`
	OrderID     uuid.UUID `json:"order_id"`
	CheckoutURL string    `json:"checkout_url"`
	Status      string    `json:"status"`
	RedeemedAt  time.Time `json:"redeemed_at"`
}