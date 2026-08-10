package usecase

import "context"

type QueueEventHandler interface {
	ListingQuantityChanged(
		ctx context.Context,
		event ListingQuantityChangedEvent,
	) error

	ListingStatusChanged(
		ctx context.Context,
		event ListingStatusChangedEvent,
	) error

	TicketClosed(
		ctx context.Context,
		event TicketClosedEvent,
	) error

	TicketRedeemed(
		ctx context.Context,
		event TicketRedeemedEvent,
	) error
}

type queueEventHandler struct {
	service ItemQueueService
}

func NewQueueEventHandler(
	service ItemQueueService,
) QueueEventHandler {
	return &queueEventHandler{
		service: service,
	}
}

func (h *queueEventHandler) ListingQuantityChanged(
	ctx context.Context,
	event ListingQuantityChangedEvent,
) error {
	return h.service.HandleListingQuantityChanged(
		ctx,
		event,
	)
}

func (h *queueEventHandler) ListingStatusChanged(
	ctx context.Context,
	event ListingStatusChangedEvent,
) error {
	return h.service.HandleListingStatusChanged(
		ctx,
		event,
	)
}

func (h *queueEventHandler) TicketClosed(
	ctx context.Context,
	event TicketClosedEvent,
) error {
	return h.service.HandleTicketClosed(
		ctx,
		event,
	)
}

func (h *queueEventHandler) TicketRedeemed(
	ctx context.Context,
	event TicketRedeemedEvent,
) error {
	return h.service.HandleTicketRedeemed(
		ctx,
		event,
	)
}