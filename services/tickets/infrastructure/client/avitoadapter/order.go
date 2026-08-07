package avitoadapter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	generated "github.com/avito-hack/queue/services/tickets/gen/clients/avitoadapter"
	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type OrderCreator struct {
	client generated.ClientInterface
}

func NewOrderCreator(baseURL string, client *http.Client) (*OrderCreator, error) {
	generatedClient, err := newGeneratedClient(baseURL, client)
	if err != nil {
		return nil, err
	}

	return &OrderCreator{client: generatedClient}, nil
}

func (c *OrderCreator) CreateOrder(ctx context.Context, request usecase.CreateOrderRequest) (usecase.CreatedOrder, error) {
	httpResponse, err := c.client.CreateOrder(
		ctx,
		&generated.CreateOrderParams{IdempotencyKey: request.IdempotencyKey},
		generated.CreateOrderJSONRequestBody{
			TicketId:  request.TicketID,
			ListingId: request.ListingID,
			SkuId:     request.SKUID,
			UserId:    request.UserID,
		},
	)
	if err != nil {
		return usecase.CreatedOrder{}, fmt.Errorf("%w: create order request: %w", usecase.ErrOrderUnavailable, err)
	}
	if httpResponse == nil || httpResponse.Body == nil {
		return usecase.CreatedOrder{}, fmt.Errorf("create order request: empty HTTP response")
	}
	defer func() { _ = httpResponse.Body.Close() }()

	response, err := generated.ParseCreateOrderResponse(httpResponse)
	if err != nil {
		if orderUnavailableStatus(httpResponse.StatusCode) {
			return usecase.CreatedOrder{}, fmt.Errorf("%w: create order status %d", usecase.ErrOrderUnavailable, httpResponse.StatusCode)
		}

		return usecase.CreatedOrder{}, fmt.Errorf("decode create order response: %w", err)
	}

	if response.StatusCode() == http.StatusCreated {
		return createdOrderFromResponse(response.JSON201, request)
	}
	if orderUnavailableStatus(response.StatusCode()) {
		return usecase.CreatedOrder{}, fmt.Errorf("%w: create order status %d", usecase.ErrOrderUnavailable, response.StatusCode())
	}
	if response.StatusCode() >= 400 && response.StatusCode() < 500 {
		return usecase.CreatedOrder{}, fmt.Errorf("%w: create order status %d", usecase.ErrOrderRejected, response.StatusCode())
	}

	return usecase.CreatedOrder{}, fmt.Errorf("create order: unexpected status %d", response.StatusCode())
}

func orderUnavailableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func createdOrderFromResponse(order *generated.Order, request usecase.CreateOrderRequest) (usecase.CreatedOrder, error) {
	if order == nil || order.Id == uuid.Nil {
		return usecase.CreatedOrder{}, fmt.Errorf("decode create order response: empty order id")
	}
	if order.TicketId != request.TicketID || order.ListingId != request.ListingID || order.SkuId != request.SKUID || order.UserId != request.UserID {
		return usecase.CreatedOrder{}, fmt.Errorf("decode create order response: order scope mismatch")
	}
	if order.Status != generated.Created {
		return usecase.CreatedOrder{}, fmt.Errorf("decode create order response: unexpected order status %q", order.Status)
	}
	checkoutURL, err := domain.NormalizeCheckoutURL(order.CheckoutUrl)
	if err != nil {
		return usecase.CreatedOrder{}, fmt.Errorf("decode create order response: %w", err)
	}

	return usecase.CreatedOrder{ID: order.Id, CheckoutURL: checkoutURL}, nil
}
