package usecase

import "sync"

type Service struct {
	mu           sync.RWMutex
	users        map[string]User
	listings     map[string]Listing
	reservations map[string]Reservation
	orders       map[string]Order
}

func NewService() *Service {
	return &Service{
		users:        map[string]User{},
		listings:     map[string]Listing{},
		reservations: map[string]Reservation{},
		orders:       map[string]Order{},
	}
}
