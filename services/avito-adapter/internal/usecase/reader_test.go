package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type readerStub struct {
	listings []Listing
	user     User
}

func (s *readerStub) ListListings(context.Context, ListingFilter) ([]Listing, int, error) {
	return s.listings, len(s.listings), nil
}

func (s *readerStub) GetListing(context.Context, string) (Listing, error) {
	return s.listings[0], nil
}

func (s *readerStub) GetUser(context.Context, string) (User, error) {
	return s.user, nil
}

func (s *readerStub) GetUserByToken(context.Context, string) (User, error) {
	return s.user, nil
}

func (s *readerStub) GetOrder(context.Context, string) (Order, error) {
	return Order{}, ErrNotFound
}

func (s *readerStub) CreateUser(_ context.Context, user User) error {
	s.user = user
	return nil
}

func (s *readerStub) CreateListing(context.Context, Listing) error {
	return nil
}

func (s *readerStub) SaveListing(context.Context, Listing) error {
	return nil
}

func (s *readerStub) CreateOrder(context.Context, Order) (Order, error) {
	return Order{}, nil
}

func Test_ListListings_ReturnRepositoryData(t *testing.T) {
	// given
	expected := Listing{ID: "11111111-1111-4111-8111-111111111101"}
	service := NewService(&readerStub{listings: []Listing{expected}})

	// when
	listings, total, err := service.ListListings(context.Background(), ListingFilter{Limit: 20})

	// then
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, []Listing{expected}, listings)
}

func Test_ValidateUserToken_ReturnRepositoryUser(t *testing.T) {
	// given
	expected := User{ID: "00000000-0000-4000-8000-000000000001"}
	service := NewService(&readerStub{user: expected})

	// when
	user, err := service.ValidateUserToken(context.Background(), "Bearer demo-token")

	// then
	require.NoError(t, err)
	assert.Equal(t, expected, user)
}

func Test_CreateUser_ValidateUserToken_ReturnCreatedUser(t *testing.T) {
	// given
	repository := &readerStub{}
	service := NewService(repository)
	created, err := service.CreateUser(context.Background(), "buyer", "demo-token")
	require.NoError(t, err)

	// when
	validated, err := service.ValidateUserToken(context.Background(), "Bearer demo-token")

	// then
	require.NoError(t, err)
	assert.Equal(t, created, validated)
}
