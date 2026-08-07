package usecase

import "sync"

type Service struct {
	mu                     sync.RWMutex
	users                  map[string]User
	listings               map[string]Listing
	orders                 map[string]Order
	ordersByIdempotencyKey map[string]string
	ordersByTicketID       map[string]string
}

func NewService() *Service {
	return &Service{
		users:                  map[string]User{},
		listings:               map[string]Listing{},
		orders:                 map[string]Order{},
		ordersByIdempotencyKey: map[string]string{},
		ordersByTicketID:       map[string]string{},
	}
}
