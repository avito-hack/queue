package avitoadapter

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListingReader_Get_ReturnListing(t *testing.T) {
	// given
	listingID := uuid.New()
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "http://avito-adapter:8080/v1/listings/"+listingID.String(), request.URL.String())

		return newHTTPResponse(http.StatusOK, `{
			"id":"`+listingID.String()+`",
			"sellerId":"`+uuid.NewString()+`",
			"title":"limited",
			"price":1000,
			"quantity":3,
			"queueEnabled":true,
			"status":"active",
			"createdAt":"2026-08-09T10:00:00Z",
			"updatedAt":"2026-08-09T10:00:00Z"
		}`), nil
	})}
	reader, err := NewListingReader("http://avito-adapter:8080", client)
	require.NoError(t, err)

	// when
	listing, err := reader.Get(context.Background(), listingID)

	// then
	require.NoError(t, err)
	assert.Equal(t, listingID, listing.ID)
	assert.Equal(t, 3, listing.Quantity)
	assert.True(t, listing.QueueEnabled)
	assert.Equal(t, "active", listing.Status)
}

func TestListingReader_Get_UnexpectedStatus_ReturnError(t *testing.T) {
	// given
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return newHTTPResponse(http.StatusNotFound, `{"message":"not found"}`), nil
	})}
	reader, err := NewListingReader("http://avito-adapter:8080", client)
	require.NoError(t, err)

	// when
	_, err = reader.Get(context.Background(), uuid.New())

	// then
	require.EqualError(t, err, "get listing: unexpected status 404")
}

func TestListingReader_Get_RequestFailure_ReturnError(t *testing.T) {
	// given
	requestError := errors.New("request failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, requestError
	})}
	reader, err := NewListingReader("http://avito-adapter:8080", client)
	require.NoError(t, err)

	// when
	_, err = reader.Get(context.Background(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, requestError)
}
