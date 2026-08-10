package usecase

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ListingFilter struct {
	SellerID *string
	Status   *ListingStatus
	Limit    int
	Offset   int
}

func (s *Service) ListListings(ctx context.Context, filter ListingFilter) ([]Listing, int, error) {
	if s.reader != nil {
		return s.reader.ListListings(ctx, filter)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	listings := make([]Listing, 0, len(s.listings))
	for _, listing := range s.listings {
		if filter.SellerID != nil && listing.SellerID != *filter.SellerID {
			continue
		}
		if filter.Status != nil && listing.Status != *filter.Status {
			continue
		}
		listings = append(listings, listing)
	}

	sort.Slice(listings, func(i, j int) bool {
		if listings[i].CreatedAt.Equal(listings[j].CreatedAt) {
			return listings[i].ID < listings[j].ID
		}
		return listings[i].CreatedAt.After(listings[j].CreatedAt)
	})

	total := len(listings)
	if filter.Offset >= total {
		return []Listing{}, total, nil
	}
	end := min(filter.Offset+filter.Limit, total)
	return listings[filter.Offset:end], total, nil
}

func (s *Service) CreateListing(ctx context.Context, sellerID, title string, price int64, quantity int, queueEnabled bool) (Listing, error) {
	if strings.TrimSpace(title) == "" || price < 0 || quantity < 0 {
		return Listing{}, ErrInvalid
	}
	now := time.Now().UTC()
	listing := Listing{ID: uuid.NewString(), SellerID: sellerID, Title: title, Price: price, Quantity: quantity, QueueEnabled: queueEnabled, Status: ListingActive, CreatedAt: now, UpdatedAt: now}
	if s.writer != nil {
		if _, err := s.reader.GetUser(ctx, sellerID); err != nil {
			return Listing{}, err
		}
		if err := s.writer.CreateListing(ctx, listing); err != nil {
			return Listing{}, err
		}
		return listing, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[sellerID]; !ok {
		return Listing{}, ErrNotFound
	}
	s.listings[listing.ID] = listing
	return listing, nil
}

func (s *Service) GetListing(ctx context.Context, id string) (Listing, error) {
	if s.reader != nil {
		return s.reader.GetListing(ctx, id)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	listing, ok := s.listings[id]
	if !ok {
		return Listing{}, ErrNotFound
	}
	return listing, nil
}

func (s *Service) UpdateListing(ctx context.Context, id string, title *string, price *int64) (Listing, error) {
	listing, err := s.GetListing(ctx, id)
	if err != nil {
		return Listing{}, err
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
	if s.writer != nil {
		if err := s.writer.SaveListing(ctx, listing); err != nil {
			return Listing{}, err
		}
		return listing, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listings[id] = listing
	return listing, nil
}

func (s *Service) ChangeQuantity(ctx context.Context, id string, quantity int) (Listing, error) {
	if quantity < 0 {
		return Listing{}, ErrInvalid
	}
	listing, err := s.GetListing(ctx, id)
	if err != nil {
		return Listing{}, err
	}
	if listing.Status == ListingRemoved {
		return Listing{}, ErrConflict
	}
	listing.Quantity = quantity
	listing.UpdatedAt = time.Now().UTC()
	if s.writer != nil {
		if err := s.writer.SaveListing(ctx, listing); err != nil {
			return Listing{}, err
		}
		return listing, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listings[id] = listing
	return listing, nil
}

func (s *Service) SetQueueEnabled(ctx context.Context, id string, enabled bool) (Listing, error) {
	return s.changeListing(ctx, id, func(l *Listing) { l.QueueEnabled = enabled })
}
func (s *Service) PauseListing(ctx context.Context, id string) (Listing, error) {
	return s.changeListing(ctx, id, func(l *Listing) { l.Status = ListingPaused })
}
func (s *Service) ActivateListing(ctx context.Context, id string) (Listing, error) {
	listing, err := s.GetListing(ctx, id)
	if err != nil {
		return Listing{}, err
	}
	if listing.Status == ListingRemoved {
		return Listing{}, ErrConflict
	}
	listing.Status = ListingActive
	listing.UpdatedAt = time.Now().UTC()
	if s.writer != nil {
		if err := s.writer.SaveListing(ctx, listing); err != nil {
			return Listing{}, err
		}
		return listing, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listings[id] = listing
	return listing, nil
}
func (s *Service) RemoveListing(ctx context.Context, id string) error {
	listing, err := s.GetListing(ctx, id)
	if err != nil {
		return err
	}
	listing.Status = ListingRemoved
	listing.UpdatedAt = time.Now().UTC()
	if s.writer != nil {
		return s.writer.SaveListing(ctx, listing)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listings[id] = listing
	return nil
}
func (s *Service) changeListing(ctx context.Context, id string, change func(*Listing)) (Listing, error) {
	listing, err := s.GetListing(ctx, id)
	if err != nil {
		return Listing{}, err
	}
	if listing.Status == ListingRemoved {
		return Listing{}, ErrConflict
	}
	change(&listing)
	listing.UpdatedAt = time.Now().UTC()
	if s.writer != nil {
		if err := s.writer.SaveListing(ctx, listing); err != nil {
			return Listing{}, err
		}
		return listing, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listings[id] = listing
	return listing, nil
}
