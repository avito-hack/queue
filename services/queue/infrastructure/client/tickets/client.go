package tickets

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"encoding/json"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	ticketsgen "github.com/avito-hack/queue/services/queue/gen/clients/tickets"
	"github.com/avito-hack/queue/services/queue/infrastructure/auth"
	"github.com/avito-hack/queue/services/queue/internal/domain"
	"github.com/avito-hack/queue/services/queue/internal/usecase"
)

type client struct {
	api    ticketsgen.ClientInterface
	logger *slog.Logger
}

func NewClient(
	api ticketsgen.ClientInterface,
	logger *slog.Logger,
) usecase.TicketsClient {
	return &client{
		api:    api,
		logger: logger,
	}
}

func (c *client) authEditor(ctx context.Context) func(context.Context, *http.Request) error {
	return func(_ context.Context, req *http.Request) error {
		header, ok := ctx.Value(auth.AuthorizationHeaderKey).(string)

		if !ok || header == "" {
			return fmt.Errorf("authorization header missing")
		}

		req.Header.Set("Authorization", header)

		return nil
	}
}

func (c *client) IssueTicket(
	ctx context.Context,
	listingID uuid.UUID,
	queueEntryID uuid.UUID,
	skuID uuid.UUID,
	userID uuid.UUID,
) (*domain.Ticket, error) {

	response, err := c.api.IssueTicket(
		ctx,
		&ticketsgen.IssueTicketParams{
			IdempotencyKey: ticketsgen.IdempotencyKey(
				openapi_types.UUID(queueEntryID),
			),
		},
		ticketsgen.IssueTicketJSONRequestBody{
			ListingId:    openapi_types.UUID(listingID),
			QueueEntryId: openapi_types.UUID(queueEntryID),
			SkuId:        openapi_types.UUID(skuID),
			UserId:       openapi_types.UUID(userID),
		},
		c.authEditor(ctx),
	)

	if err != nil {
		return nil, fmt.Errorf("issue ticket request: %w", err)
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusCreated, http.StatusOK:

	default:
		return nil, fmt.Errorf(
			"issue ticket failed with status %d",
			response.StatusCode,
		)
	}

	var ticket ticketsgen.V1Ticket

	if err := json.NewDecoder(response.Body).Decode(&ticket); err != nil {
		return nil, fmt.Errorf(
			"decode ticket response: %w",
			err,
		)
	}

	return &domain.Ticket{
		ID: ticket.Id.String(),
		Status: domain.TicketStatus(
			ticket.Status,
		),
	}, nil
}

func (c *client) DeclineTicket(
	ctx context.Context,
	ticketID uuid.UUID,
) error {

	response, err := c.api.DeclineTicket(
		ctx,
		openapi_types.UUID(ticketID),
		&ticketsgen.DeclineTicketParams{
			IdempotencyKey: ticketsgen.IdempotencyKey(
				openapi_types.UUID(uuid.New()),
			),
		},
		c.authEditor(ctx),
	)

	if err != nil {
		return fmt.Errorf(
			"decline ticket request: %w",
			err,
		)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"decline ticket failed: %d",
			response.StatusCode,
		)
	}

	return nil
}