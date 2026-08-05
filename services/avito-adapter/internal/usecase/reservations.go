package usecase

import (
	"time"

	"github.com/google/uuid"
)

func (s *Service) CreateReservation(listingID, userID string, quantity int) (Reservation, error) {
	if quantity < 1 {
		return Reservation{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[userID]; !ok {
		return Reservation{}, ErrNotFound
	}
	listing, ok := s.listings[listingID]
	if !ok {
		return Reservation{}, ErrNotFound
	}

	if listing.Status != ListingActive {
		return Reservation{}, ErrUnavailable
	}
	if listing.AvailableQuantity() < quantity {
		return Reservation{}, ErrInsufficientStock
	}

	listing.ReservedQuantity += quantity
	listing.UpdatedAt = time.Now().UTC()
	s.listings[listing.ID] = listing
	reservation := Reservation{ID: uuid.NewString(), ListingID: listingID, UserID: userID, Quantity: quantity, Status: ReservationActive, CreatedAt: time.Now().UTC()}
	s.reservations[reservation.ID] = reservation
	return reservation, nil
}

func (s *Service) GetReservation(id string) (Reservation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	reservation, ok := s.reservations[id]
	if !ok {
		return Reservation{}, ErrNotFound
	}
	return reservation, nil
}

func (s *Service) CancelReservation(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	reservation, ok := s.reservations[id]
	if !ok {
		return ErrNotFound
	}
	if reservation.Status == ReservationCancelled {
		return nil
	}
	reservation.Status = ReservationCancelled
	s.reservations[id] = reservation
	listing := s.listings[reservation.ListingID]
	listing.ReservedQuantity -= reservation.Quantity
	listing.UpdatedAt = time.Now().UTC()
	s.listings[listing.ID] = listing
	return nil
}
