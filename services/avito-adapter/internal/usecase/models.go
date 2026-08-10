package usecase

import "time"

type ListingStatus string

const (
	ListingActive  ListingStatus = "active"
	ListingPaused  ListingStatus = "paused"
	ListingRemoved ListingStatus = "removed"
)

type OrderStatus string

const (
	OrderCreated OrderStatus = "created"
)

type User struct {
	ID, Name, Token string
	CreatedAt       time.Time
}

type Listing struct {
	ID, SellerID, Title  string
	Price                int64
	Quantity             int
	QueueEnabled         bool
	Status               ListingStatus
	CreatedAt, UpdatedAt time.Time
}

type Order struct {
	ID, TicketID, ListingID, SkuID, UserID, IdempotencyKey, CheckoutURL string
	Status                                                              OrderStatus
	CreatedAt                                                           time.Time
}
