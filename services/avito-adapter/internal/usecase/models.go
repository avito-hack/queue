package usecase

import "time"

type ListingStatus string

const (
	ListingActive  ListingStatus = "active"
	ListingPaused  ListingStatus = "paused"
	ListingRemoved ListingStatus = "removed"
)

type ReservationStatus string

const (
	ReservationActive    ReservationStatus = "active"
	ReservationCancelled ReservationStatus = "cancelled"
)

type User struct {
	ID, Name  string
	CreatedAt time.Time
}

type Listing struct {
	ID, SellerID, Title        string
	Price                      int64
	Quantity, ReservedQuantity int
	QueueEnabled               bool
	Status                     ListingStatus
	CreatedAt, UpdatedAt       time.Time
}

func (l Listing) AvailableQuantity() int {
	return l.Quantity - l.ReservedQuantity
}

type Reservation struct {
	ID, ListingID, UserID string
	Quantity              int
	Status                ReservationStatus
	CreatedAt             time.Time
}

type Order struct {
	ID, ReservationID, ListingID, UserID string
	CreatedAt                            time.Time
}
