package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ValidateUserToken_ReturnsUser(t *testing.T) {
	// given
	service := NewService()
	created, err := service.CreateUser("buyer", "Bearer token-1")
	require.NoError(t, err)

	// when
	user, err := service.ValidateUserToken(context.Background(), "Bearer token-1")

	// then
	require.NoError(t, err)
	assert.Equal(t, created.ID, user.ID)
}

func Test_CreateOrder_ReturnsExistingOrderForSameIdempotencyKey(t *testing.T) {
	// given
	service := NewService()
	seller, err := service.CreateUser("seller", "seller-token")
	require.NoError(t, err)
	buyer, err := service.CreateUser("buyer", "buyer-token")
	require.NoError(t, err)
	listing, err := service.CreateListing(seller.ID, "item", 100, 1, true)
	require.NoError(t, err)
	ticketID := uuid.NewString()
	skuID := uuid.NewString()
	idempotencyKey := uuid.NewString()

	// when
	first, err := service.CreateOrder(ticketID, listing.ID, skuID, buyer.ID, idempotencyKey)
	require.NoError(t, err)
	second, err := service.CreateOrder(ticketID, listing.ID, skuID, buyer.ID, idempotencyKey)

	// then
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Contains(t, first.CheckoutURL, skuID)
}

func Test_CreateOrder_ReturnsConflictWhenStockIsReserved(t *testing.T) {
	// given
	service := NewService()
	seller, err := service.CreateUser("seller", "seller-token")
	require.NoError(t, err)
	buyer, err := service.CreateUser("buyer", "buyer-token")
	require.NoError(t, err)
	listing, err := service.CreateListing(seller.ID, "item", 100, 1, true)
	require.NoError(t, err)
	_, err = service.CreateOrder(uuid.NewString(), listing.ID, uuid.NewString(), buyer.ID, uuid.NewString())
	require.NoError(t, err)

	// when
	_, err = service.CreateOrder(uuid.NewString(), listing.ID, uuid.NewString(), buyer.ID, uuid.NewString())

	// then
	assert.ErrorIs(t, err, ErrConflict)
}

func Test_CreateOrder_ReturnsConflictForSameKeyAndDifferentBody(t *testing.T) {
	// given
	service := NewService()
	seller, err := service.CreateUser("seller", "seller-token")
	require.NoError(t, err)
	buyer, err := service.CreateUser("buyer", "buyer-token")
	require.NoError(t, err)
	listing, err := service.CreateListing(seller.ID, "item", 100, 2, true)
	require.NoError(t, err)
	idempotencyKey := uuid.NewString()
	_, err = service.CreateOrder(uuid.NewString(), listing.ID, uuid.NewString(), buyer.ID, idempotencyKey)
	require.NoError(t, err)

	// when
	_, err = service.CreateOrder(uuid.NewString(), listing.ID, uuid.NewString(), buyer.ID, idempotencyKey)

	// then
	assert.ErrorIs(t, err, ErrConflict)
}

func Test_CreateOrder_ReturnsConflictForReusedTicket(t *testing.T) {
	// given
	service := NewService()
	seller, err := service.CreateUser("seller", "seller-token")
	require.NoError(t, err)
	buyer, err := service.CreateUser("buyer", "buyer-token")
	require.NoError(t, err)
	listing, err := service.CreateListing(seller.ID, "item", 100, 2, true)
	require.NoError(t, err)
	ticketID := uuid.NewString()
	_, err = service.CreateOrder(ticketID, listing.ID, uuid.NewString(), buyer.ID, uuid.NewString())
	require.NoError(t, err)

	// when
	_, err = service.CreateOrder(ticketID, listing.ID, uuid.NewString(), buyer.ID, uuid.NewString())

	// then
	assert.ErrorIs(t, err, ErrConflict)
}
