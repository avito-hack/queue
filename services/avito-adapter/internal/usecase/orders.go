package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *Service) CreateOrder(ticketID, listingID, skuID, userID, idempotencyKey string) (Order, error) {
	if _, err := uuid.Parse(ticketID); err != nil {
		return Order{}, ErrInvalid
	}
	if _, err := uuid.Parse(idempotencyKey); err != nil {
		return Order{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existingID, ok := s.ordersByIdempotencyKey[idempotencyKey]; ok {
		order := s.orders[existingID]
		if order.TicketID != ticketID || order.ListingID != listingID || order.SkuID != skuID || order.UserID != userID {
			return Order{}, ErrConflict
		}
		return order, nil
	}
	if _, ok := s.ordersByTicketID[ticketID]; ok {
		return Order{}, ErrConflict
	}
	if _, ok := s.users[userID]; !ok {
		return Order{}, ErrNotFound
	}
	listing, ok := s.listings[listingID]
	if !ok {
		return Order{}, ErrNotFound
	}
	if listing.Status != ListingActive {
		return Order{}, ErrUnavailable
	}
	reservedQuantity := 0
	for _, order := range s.orders {
		if order.ListingID == listingID {
			reservedQuantity++
		}
	}
	if reservedQuantity >= listing.Quantity {
		return Order{}, ErrConflict
	}
	orderID := uuid.NewString()
	order := Order{
		ID: orderID, TicketID: ticketID, ListingID: listingID, SkuID: skuID, UserID: userID, IdempotencyKey: idempotencyKey,
		CheckoutURL: fmt.Sprintf("https://checkout.local/orders/%s?sku_id=%s", orderID, skuID),
		Status:      OrderCreated, CreatedAt: time.Now().UTC(),
	}
	s.orders[order.ID] = order
	s.ordersByIdempotencyKey[idempotencyKey] = order.ID
	s.ordersByTicketID[ticketID] = order.ID
	return order, nil
}

func (s *Service) GetOrder(ctx context.Context, id string) (Order, error) {
	if s.reader != nil {
		return s.reader.GetOrder(ctx, id)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	order, ok := s.orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	return order, nil
}
