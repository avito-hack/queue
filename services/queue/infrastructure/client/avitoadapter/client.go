package avito

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	avitogen "github.com/avito-hack/queue/services/queue/gen/clients/avito"
	"github.com/avito-hack/queue/services/queue/internal/domain"
	"github.com/avito-hack/queue/services/queue/internal/usecase"
)

type client struct {
	api    avitogen.ClientInterface
	logger *slog.Logger
}

func NewClient(
	api avitogen.ClientInterface,
	logger *slog.Logger,
) usecase.AvitoClient {
	return &client{
		api:    api,
		logger: logger,
	}
}

func (c *client) GetListing(
	ctx context.Context,
	itemID uuid.UUID,
) (*domain.Listing, error) {

	response, err := c.api.GetListing(
		ctx,
		avitogen.ListingId(itemID),
	)

	if err != nil {
		c.logger.Error(
			"failed to get listing from avito",
			"error",
			err,
			"item_id",
			itemID,
		)

		return nil, fmt.Errorf(
			"get listing request: %w",
			err,
		)
	}

	defer response.Body.Close()

	switch response.StatusCode {

	case http.StatusNotFound:
		return nil, fmt.Errorf(
			"listing not found",
		)

	case http.StatusOK:

	default:
		return nil, fmt.Errorf(
			"unexpected avito status: %d",
			response.StatusCode,
			)
	}

	var listing avitogen.Listing

	if err := json.NewDecoder(
		response.Body,
	).Decode(&listing); err != nil {

		return nil, fmt.Errorf(
			"decode listing response: %w",
			err,
		)
	}


	result := &domain.Listing{
	    ID: uuid.UUID(listing.Id),
	    SkuID: uuid.UUID(listing.Id),
		
	    QueueEnabled: listing.QueueEnabled,
	    Status:       string(listing.Status),
	    Quantity:     listing.Quantity,
	}


	c.logger.Info(
		"listing fetched successfully",
		"item_id",
		itemID,
		"sku_id",
		result.SkuID,
		"status",
		result.Status,
		"quantity",
		result.Quantity,
		"queue_enabled",
		result.QueueEnabled,
	)


	return result, nil
}