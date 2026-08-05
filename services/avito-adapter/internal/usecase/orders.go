package usecase

import (
	"time"

	"github.com/google/uuid"
)

func (s *Service) CreateOrder(reservationID, userID string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return Order{}, ErrNotFound
	}
	reservation, ok := s.reservations[reservationID]
	if !ok {
		return Order{}, ErrNotFound
	}
	if reservation.UserID != userID || reservation.Status != ReservationActive {
		return Order{}, ErrConflict
	}
	for _, order := range s.orders {
		if order.ReservationID == reservationID {
			return Order{}, ErrConflict
		}
	}
	order := Order{ID: uuid.NewString(), ReservationID: reservationID, ListingID: reservation.ListingID, UserID: userID, CreatedAt: time.Now().UTC()}
	s.orders[order.ID] = order
	return order, nil
}

func (s *Service) GetOrder(id string) (Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	return order, nil
}
