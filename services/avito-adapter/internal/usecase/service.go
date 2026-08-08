package usecase

import (
	"context"
	"sync"
)

type Reader interface {
	ListListings(context.Context, ListingFilter) ([]Listing, int, error)
	GetListing(context.Context, string) (Listing, error)
	GetUser(context.Context, string) (User, error)
	GetUserByToken(context.Context, string) (User, error)
	GetOrder(context.Context, string) (Order, error)
}

type Service struct {
	mu                     sync.RWMutex
	users                  map[string]User
	listings               map[string]Listing
	orders                 map[string]Order
	ordersByIdempotencyKey map[string]string
	ordersByTicketID       map[string]string
	reader                 Reader
}

func NewService(readers ...Reader) *Service {
	service := &Service{
		users:                  map[string]User{},
		listings:               map[string]Listing{},
		orders:                 map[string]Order{},
		ordersByIdempotencyKey: map[string]string{},
		ordersByTicketID:       map[string]string{},
	}
	if len(readers) > 0 {
		service.reader = readers[0]
	}
	return service
}
