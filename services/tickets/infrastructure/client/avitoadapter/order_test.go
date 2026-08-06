package avitoadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	generated "github.com/avito-hack/queue/services/tickets/gen/clients/avitoadapter"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

func TestOrderCreator_CreateOrder_ReturnOrder(t *testing.T) {
	// given
	request := usecase.CreateOrderRequest{
		TicketID:       uuid.New(),
		ListingID:      uuid.New(),
		SKUID:          uuid.New(),
		UserID:         uuid.New(),
		IdempotencyKey: uuid.New(),
	}
	expectedOrderID := uuid.New()
	client := &http.Client{Transport: roundTripFunc(func(httpRequest *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, httpRequest.Method)
		assert.Equal(t, "http://avito-adapter:8080/v1/orders", httpRequest.URL.String())
		assert.Equal(t, request.IdempotencyKey.String(), httpRequest.Header.Get("Idempotency-Key"))
		assert.Equal(t, "application/json", httpRequest.Header.Get("Content-Type"))

		var body generated.CreateOrderRequest
		assert.NoError(t, json.NewDecoder(httpRequest.Body).Decode(&body))
		assert.Equal(t, request.TicketID, body.TicketId)
		assert.Equal(t, request.ListingID, body.ListingId)
		assert.Equal(t, request.SKUID, body.SkuId)
		assert.Equal(t, request.UserID, body.UserId)

		responseBody, err := json.Marshal(generated.Order{
			Id:          expectedOrderID,
			TicketId:    request.TicketID,
			ListingId:   request.ListingID,
			SkuId:       request.SKUID,
			UserId:      request.UserID,
			Status:      generated.Created,
			CheckoutUrl: "/checkout?ticket=" + request.TicketID.String(),
			CreatedAt:   time.Now(),
		})
		require.NoError(t, err)

		return newHTTPResponse(http.StatusCreated, string(responseBody)), nil
	})}
	creator, err := NewOrderCreator("http://avito-adapter:8080", client)
	require.NoError(t, err)

	// when
	order, err := creator.CreateOrder(context.Background(), request)

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedOrderID, order.ID)
	assert.Equal(t, "/checkout?ticket="+request.TicketID.String(), order.CheckoutURL)
}

func TestOrderCreator_CreateOrder_RequestReturnsError_ReturnUnavailable(t *testing.T) {
	// given
	requestError := errors.New("request failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, requestError
	})}
	creator, err := NewOrderCreator("http://avito-adapter:8080", client)
	require.NoError(t, err)

	// when
	_, err = creator.CreateOrder(context.Background(), validCreateOrderRequest())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, usecase.ErrOrderUnavailable)
	assert.ErrorIs(t, err, requestError)
}

func TestOrderCreator_CreateOrder_AdapterUnavailable_ReturnUnavailable(t *testing.T) {
	// given
	statuses := []int{http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusServiceUnavailable}

	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return newHTTPResponse(status, `{"message":"adapter unavailable"}`), nil
			})}
			creator, err := NewOrderCreator("http://avito-adapter:8080", client)
			require.NoError(t, err)

			// when
			_, err = creator.CreateOrder(context.Background(), validCreateOrderRequest())

			// then
			assert.ErrorIs(t, err, usecase.ErrOrderUnavailable)
		})
	}
}

func TestOrderCreator_CreateOrder_AdapterRejectsRequest_ReturnError(t *testing.T) {
	// given
	statuses := []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict}

	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return newHTTPResponse(status, `{"message":"request rejected"}`), nil
			})}
			creator, err := NewOrderCreator("http://avito-adapter:8080", client)
			require.NoError(t, err)

			// when
			_, err = creator.CreateOrder(context.Background(), validCreateOrderRequest())

			// then
			require.EqualError(t, err, fmt.Sprintf("create order: unexpected status %d", status))
			assert.NotErrorIs(t, err, usecase.ErrOrderUnavailable)
		})
	}
}

func TestOrderCreator_CreateOrder_InvalidResponse_ReturnError(t *testing.T) {
	// given
	request := validCreateOrderRequest()
	responseBody := func(checkoutURL string) string {
		body, err := json.Marshal(generated.Order{
			Id:          uuid.New(),
			TicketId:    request.TicketID,
			ListingId:   request.ListingID,
			SkuId:       request.SKUID,
			UserId:      request.UserID,
			Status:      generated.Created,
			CheckoutUrl: checkoutURL,
			CreatedAt:   time.Now(),
		})
		require.NoError(t, err)

		return string(body)
	}
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: "{"},
		{name: "empty order", body: `{}`},
		{name: "scope mismatch", body: `{"id":"` + uuid.NewString() + `","ticketId":"` + uuid.NewString() + `","listingId":"` + request.ListingID.String() + `","skuId":"` + request.SKUID.String() + `","userId":"` + request.UserID.String() + `","status":"created","checkoutUrl":"/checkout","createdAt":"2026-08-07T00:00:00Z"}`},
		{name: "unsafe scheme", body: responseBody("javascript:alert(1)")},
		{name: "protocol relative URL", body: responseBody("//example.com/checkout")},
		{name: "rootless path", body: responseBody("checkout/1")},
		{name: "backslash path", body: responseBody(`/\example.com/checkout`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return newHTTPResponse(http.StatusCreated, test.body), nil
			})}
			creator, err := NewOrderCreator("http://avito-adapter:8080", client)
			require.NoError(t, err)

			// when
			_, err = creator.CreateOrder(context.Background(), request)

			// then
			require.Error(t, err)
			assert.NotErrorIs(t, err, usecase.ErrOrderUnavailable)
		})
	}
}

func TestCreatedOrderFromResponse_SafeCheckoutURL_ReturnOrder(t *testing.T) {
	// given
	request := validCreateOrderRequest()
	tests := []string{"/checkout/1#payment", "https://www.avito.ru/checkout/1"}

	for _, checkoutURL := range tests {
		t.Run(checkoutURL, func(t *testing.T) {
			order := &generated.Order{
				Id:          uuid.New(),
				TicketId:    request.TicketID,
				ListingId:   request.ListingID,
				SkuId:       request.SKUID,
				UserId:      request.UserID,
				Status:      generated.Created,
				CheckoutUrl: checkoutURL,
				CreatedAt:   time.Now(),
			}

			// when
			createdOrder, err := createdOrderFromResponse(order, request)

			// then
			require.NoError(t, err)
			assert.Equal(t, order.Id, createdOrder.ID)
			assert.Equal(t, checkoutURL, createdOrder.CheckoutURL)
		})
	}
}

func validCreateOrderRequest() usecase.CreateOrderRequest {
	return usecase.CreateOrderRequest{
		TicketID:       uuid.New(),
		ListingID:      uuid.New(),
		SKUID:          uuid.New(),
		UserID:         uuid.New(),
		IdempotencyKey: uuid.New(),
	}
}
