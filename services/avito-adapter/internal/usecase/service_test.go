package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CreateReservation_DecreasesAvailableQuantity(t *testing.T) {
	// given
	service := NewService()
	seller, err := service.CreateUser("seller")
	require.NoError(t, err)
	buyer, err := service.CreateUser("buyer")
	require.NoError(t, err)
	listing, err := service.CreateListing(seller.ID, "item", 100, 2, true)
	require.NoError(t, err)

	// when
	reservation, err := service.CreateReservation(listing.ID, buyer.ID, 1)
	require.NoError(t, err)
	updated, err := service.GetListing(listing.ID)
	require.NoError(t, err)

	// then
	assert.Equal(t, ReservationActive, reservation.Status)
	assert.Equal(t, 1, updated.ReservedQuantity)
	assert.Equal(t, 1, updated.Quantity-updated.ReservedQuantity)
}

func Test_CancelReservation_RestoresAvailabilityAndOrderCannotBeDuplicated(t *testing.T) {
	// given
	service := NewService()
	seller, err := service.CreateUser("seller")
	require.NoError(t, err)
	buyer, err := service.CreateUser("buyer")
	require.NoError(t, err)
	listing, err := service.CreateListing(seller.ID, "item", 100, 1, true)
	require.NoError(t, err)
	reservation, err := service.CreateReservation(listing.ID, buyer.ID, 1)
	require.NoError(t, err)
	_, err = service.CreateOrder(reservation.ID, buyer.ID)
	require.NoError(t, err)

	// when
	_, duplicateOrderErr := service.CreateOrder(reservation.ID, buyer.ID)
	cancelErr := service.CancelReservation(reservation.ID)
	updated, getErr := service.GetListing(listing.ID)

	// then
	assert.ErrorIs(t, duplicateOrderErr, ErrConflict)
	assert.NoError(t, cancelErr)
	require.NoError(t, getErr)
	assert.Equal(t, 0, updated.ReservedQuantity)
}
