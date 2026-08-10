package avitoadapter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	generated "github.com/avito-hack/queue/services/tickets/gen/clients/avitoadapter"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type ListingReader struct {
	client generated.ClientWithResponsesInterface
}

func NewListingReader(baseURL string, client *http.Client) (*ListingReader, error) {
	generatedClient, err := newGeneratedClient(baseURL, client)
	if err != nil {
		return nil, err
	}

	return &ListingReader{client: generatedClient}, nil
}

func (r *ListingReader) Get(ctx context.Context, listingID uuid.UUID) (usecase.TicketIssueListing, error) {
	response, err := r.client.GetListingWithResponse(ctx, generated.ListingId(listingID))
	if err != nil {
		return usecase.TicketIssueListing{}, fmt.Errorf("get listing request: %w", err)
	}
	if response.StatusCode() != http.StatusOK {
		return usecase.TicketIssueListing{}, fmt.Errorf("get listing: unexpected status %d", response.StatusCode())
	}
	if response.JSON200 == nil {
		return usecase.TicketIssueListing{}, fmt.Errorf("decode get listing response: empty listing")
	}

	return usecase.TicketIssueListing{
		ID:           response.JSON200.Id,
		Quantity:     response.JSON200.Quantity,
		QueueEnabled: response.JSON200.QueueEnabled,
		Status:       string(response.JSON200.Status),
	}, nil
}
