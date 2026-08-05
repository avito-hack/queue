package usecase

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Service) CreateListing(sellerID, title string, price int64, quantity int, queueEnabled bool) (Listing, error) {
	if strings.TrimSpace(title) == "" || price < 0 || quantity < 0 {
		return Listing{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[sellerID]; !ok {
		return Listing{}, ErrNotFound
	}
	now := time.Now().UTC()
	listing := Listing{ID: uuid.NewString(), SellerID: sellerID, Title: title, Price: price, Quantity: quantity, QueueEnabled: queueEnabled, Status: ListingActive, CreatedAt: now, UpdatedAt: now}
	s.listings[listing.ID] = listing
	return listing, nil
}

func (s *Service) GetListing(id string) (Listing, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	listing, ok := s.listings[id]
	if !ok {
		return Listing{}, ErrNotFound
	}
	return listing, nil
}

func (s *Service) UpdateListing(id string, title *string, price *int64) (Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	listing, ok := s.listings[id]
	if !ok {
		return Listing{}, ErrNotFound
	}
	if listing.Status == ListingRemoved {
		return Listing{}, ErrConflict
	}
	if title != nil {
		if strings.TrimSpace(*title) == "" {
			return Listing{}, ErrInvalid
		}
		listing.Title = *title
	}
	if price != nil {
		if *price < 0 {
			return Listing{}, ErrInvalid
		}
		listing.Price = *price
	}
	listing.UpdatedAt = time.Now().UTC()
	s.listings[id] = listing
	return listing, nil
}

func (s *Service) ChangeQuantity(id string, quantity int) (Listing, error) {
	if quantity < 0 {
		return Listing{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	listing, ok := s.listings[id]
	if !ok {
		return Listing{}, ErrNotFound
	}
	if listing.Status == ListingRemoved || quantity < listing.ReservedQuantity {
		return Listing{}, ErrConflict
	}
	listing.Quantity = quantity
	listing.UpdatedAt = time.Now().UTC()
	s.listings[id] = listing
	return listing, nil
}

func (s *Service) SetQueueEnabled(id string, enabled bool) (Listing, error) {
	return s.changeListing(id, func(l *Listing) { l.QueueEnabled = enabled })
}
func (s *Service) PauseListing(id string) (Listing, error) {
	return s.changeListing(id, func(l *Listing) { l.Status = ListingPaused })
}
func (s *Service) ActivateListing(id string) (Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	listing, ok := s.listings[id]
	if !ok {
		return Listing{}, ErrNotFound
	}
	if listing.Status == ListingRemoved {
		return Listing{}, ErrConflict
	}
	listing.Status = ListingActive
	listing.UpdatedAt = time.Now().UTC()
	s.listings[id] = listing
	return listing, nil
}
func (s *Service) RemoveListing(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	listing, ok := s.listings[id]
	if !ok {
		return ErrNotFound
	}
	listing.Status = ListingRemoved
	listing.UpdatedAt = time.Now().UTC()
	s.listings[id] = listing
	return nil
}
func (s *Service) changeListing(id string, change func(*Listing)) (Listing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	listing, ok := s.listings[id]
	if !ok {
		return Listing{}, ErrNotFound
	}
	if listing.Status == ListingRemoved {
		return Listing{}, ErrConflict
	}
	change(&listing)
	listing.UpdatedAt = time.Now().UTC()
	s.listings[id] = listing
	return listing, nil
}
