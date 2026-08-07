package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	avitoclient "github.com/avito-hack/queue/services/queue/gen/clients/avito"
	ticketsclient "github.com/avito-hack/queue/services/queue/gen/clients/tickets"
	"github.com/avito-hack/queue/services/queue/infrastructure/auth"
)

func (s *itemQueueService) ensureItemAvailable(ctx context.Context, itemID uuid.UUID) error {
	response, err := s.avitoClient.GetListing(ctx, avitoclient.ListingId(itemID))
	if err != nil {
		return fmt.Errorf("get listing: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return ErrQueueNotFound
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("get listing status: %d", response.StatusCode)
	}

	var listing avitoclient.Listing
	if err := json.NewDecoder(response.Body).Decode(&listing); err != nil {
		return fmt.Errorf("decode listing response: %w", err)
	}

	if !listing.QueueEnabled {
		return ErrQueueUnavailable
	}

	if listing.Status != avitoclient.ListingStatusActive {
		return ErrQueueUnavailable
	}

	if listing.AvailableQuantity <= 0 {
		return ErrQueueUnavailable
	}

	return nil
}

func (s *itemQueueService) ensureUserHasNoActiveTicket(ctx context.Context, itemID uuid.UUID) error {
	header, ok := auth.AuthorizationHeader(ctx)
	if !ok {
		return fmt.Errorf("authorization header not found in context")
	}

	listingID := openapi_types.UUID(itemID)
	params := &ticketsclient.ListTicketsParams{
		ListingId: &listingID,
	}

	response, err := s.ticketsClient.ListTickets(
		ctx,
		params,
		func(_ context.Context, request *http.Request) error {
			request.Header.Set("Authorization", header)
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("list tickets: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("tickets unauthorized")
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("list tickets status: %d", response.StatusCode)
	}

	var list ticketsclient.V1TicketListResponse
	if err := json.NewDecoder(response.Body).Decode(&list); err != nil {
		return fmt.Errorf("decode tickets response: %w", err)
	}

	for _, ticket := range list.Ticket {
		if ticket.Status == ticketsclient.Active || ticket.Status == ticketsclient.Issued || ticket.Status == ticketsclient.Redeemed {
			return ErrUserHasActiveTicket
		}
	}

	return nil
}
