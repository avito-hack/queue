package tickets

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

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

func (c *client) GetTickets(
	ctx context.Context,
	itemID uuid.UUID,
) ([]*domain.Ticket, error) {

	listingID := openapi_types.UUID(itemID)

	response, err := c.api.ListTickets(
		ctx,
		&ticketsgen.ListTicketsParams{
			ListingId: &listingID,
		},
		func(
			_ context.Context,
			req *http.Request,
		) error {

			header, ok := ctx.Value(
				auth.AuthorizationHeaderKey,
			).(string)

			if !ok || header == "" {
				err := fmt.Errorf(
					"authorization header missing",
				)

				c.logger.Error(
					"failed to authorize tickets request",
					"error",
					err,
					"item_id",
					itemID,
				)

				return err
			}

			req.Header.Set(
				"Authorization",
				header,
			)

			return nil
		},
	)

	if err != nil {
		c.logger.Error(
			"tickets request failed",
			"error",
			err,
			"item_id",
			itemID,
		)

		return nil, fmt.Errorf(
			"list tickets request: %w",
			err,
		)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {

		err := fmt.Errorf(
			"unexpected tickets status: %d",
			response.StatusCode,
		)

		c.logger.Error(
			"tickets service returned error",
			"error",
			err,
			"item_id",
			itemID,
			"status_code",
			response.StatusCode,
		)

		return nil, err
	}

	var responseBody ticketsgen.V1TicketListResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&responseBody); err != nil {

		c.logger.Error(
			"failed to decode tickets response",
			"error",
			err,
			"item_id",
			itemID,
		)

		return nil, fmt.Errorf(
			"decode tickets response: %w",
			err,
		)
	}

	result := make(
		[]*domain.Ticket,
		0,
		len(responseBody.Ticket),
	)

	for _, ticket := range responseBody.Ticket {

		result = append(
			result,
			&domain.Ticket{
				ID: ticket.Id.String(),
				Status: domain.TicketStatus(
					ticket.Status,
				),
			},
		)
	}

	c.logger.Info(
		"tickets fetched successfully",
		"item_id",
		itemID,
		"tickets_count",
		len(result),
	)

	return result, nil
}
