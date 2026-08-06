package http

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/tickets/gen/server"
	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

const notImplementedMessage = "tickets service is not implemented"

type HealthChecker interface {
	Check(context.Context) error
}

type TicketLister interface {
	List(context.Context, uuid.UUID, usecase.ListTicketsFilter) ([]domain.Ticket, error)
}

type TicketGetter interface {
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.Ticket, error)
}

type Handler struct {
	healthChecker HealthChecker
	ticketLister  TicketLister
	ticketGetter  TicketGetter
}

func NewHandler(healthChecker HealthChecker, ticketLister TicketLister, ticketGetter TicketGetter) *Handler {
	return &Handler{
		healthChecker: healthChecker,
		ticketLister:  ticketLister,
		ticketGetter:  ticketGetter,
	}
}

func (h *Handler) GetHealth(ctx context.Context, _ server.GetHealthRequestObject) (server.GetHealthResponseObject, error) {
	if err := h.healthChecker.Check(ctx); err != nil {
		return nil, err
	}

	return server.GetHealth200Response{}, nil
}

func (h *Handler) IssueTicket(context.Context, server.IssueTicketRequestObject) (server.IssueTicketResponseObject, error) {
	return server.IssueTicket500JSONResponse{InternalErrorJSONResponse: notImplementedError()}, nil
}

func (h *Handler) ListTickets(ctx context.Context, request server.ListTicketsRequestObject) (server.ListTicketsResponseObject, error) {
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return server.ListTickets401JSONResponse{UnauthorizedJSONResponse: unauthorizedError()}, nil
	}

	filter := toListTicketsFilter(request.Params)
	tickets, err := h.ticketLister.List(ctx, userID, filter)
	if errors.Is(err, usecase.ErrInvalidTicketFilter) {
		return server.ListTickets400JSONResponse{BadRequestJSONResponse: badRequestError(err.Error())}, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "list tickets", "error", err)
		return server.ListTickets500JSONResponse{InternalErrorJSONResponse: internalServerError()}, nil
	}

	responseTickets := make([]server.V1Ticket, len(tickets))
	for index := range tickets {
		responseTickets[index] = toV1Ticket(tickets[index])
	}

	return server.ListTickets200JSONResponse{Ticket: responseTickets}, nil
}

func (h *Handler) GetTicket(ctx context.Context, request server.GetTicketRequestObject) (server.GetTicketResponseObject, error) {
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return server.GetTicket401JSONResponse{UnauthorizedJSONResponse: unauthorizedError()}, nil
	}

	ticket, err := h.ticketGetter.Get(ctx, userID, request.TicketId)
	if errors.Is(err, usecase.ErrTicketNotFound) {
		return server.GetTicket404JSONResponse{TicketNotFoundJSONResponse: ticketNotFoundError()}, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "get ticket", "error", err)
		return server.GetTicket500JSONResponse{InternalErrorJSONResponse: internalServerError()}, nil
	}

	return server.GetTicket200JSONResponse(toV1Ticket(ticket)), nil
}

func (h *Handler) ActivateTicket(context.Context, server.ActivateTicketRequestObject) (server.ActivateTicketResponseObject, error) {
	return server.ActivateTicket500JSONResponse{InternalErrorJSONResponse: notImplementedError()}, nil
}

func (h *Handler) DeclineTicket(context.Context, server.DeclineTicketRequestObject) (server.DeclineTicketResponseObject, error) {
	return server.DeclineTicket500JSONResponse{InternalErrorJSONResponse: notImplementedError()}, nil
}

func toListTicketsFilter(params server.ListTicketsParams) usecase.ListTicketsFilter {
	filter := usecase.ListTicketsFilter{
		ListingID: params.ListingId,
		SKUID:     params.SkuId,
	}
	if params.Status != nil {
		status := domain.TicketStatus(*params.Status)
		filter.Status = &status
	}

	return filter
}

func toV1Ticket(ticket domain.Ticket) server.V1Ticket {
	actions := make([]server.V1TicketAvailableAction, len(ticket.AvailableActions))
	for index := range ticket.AvailableActions {
		actions[index] = server.V1TicketAvailableAction(ticket.AvailableActions[index])
	}

	var closeReason *server.V1TicketCloseReason
	if ticket.CloseReason != nil {
		value := server.V1TicketCloseReason(*ticket.CloseReason)
		closeReason = &value
	}

	return server.V1Ticket{
		Id:                 ticket.ID,
		ListingId:          ticket.ListingID,
		SkuId:              ticket.SKUID,
		Status:             server.V1TicketStatus(ticket.Status),
		IssuedAt:           ticket.IssuedAt,
		ActivationDeadline: ticket.ActivationDeadline,
		ActivatedAt:        ticket.ActivatedAt,
		OrderId:            ticket.OrderID,
		CheckoutUrl:        ticket.CheckoutURL,
		FinishedAt:         ticket.FinishedAt,
		FinishReason:       closeReason,
		AvailableActions:   actions,
	}
}

func badRequestError(message string) server.BadRequestJSONResponse {
	return server.BadRequestJSONResponse{
		Error:   "bad_request",
		Message: message,
	}
}

func unauthorizedError() server.UnauthorizedJSONResponse {
	return server.UnauthorizedJSONResponse{
		Error:   "unauthorized",
		Message: errBearerTokenInvalid.Error(),
	}
}

func ticketNotFoundError() server.TicketNotFoundJSONResponse {
	return server.TicketNotFoundJSONResponse{
		Error:   "ticket_not_found",
		Message: "ticket not found",
	}
}

func internalServerError() server.InternalErrorJSONResponse {
	return server.InternalErrorJSONResponse{
		Error:   "internal_error",
		Message: "internal server error",
	}
}

func notImplementedError() server.InternalErrorJSONResponse {
	return server.InternalErrorJSONResponse{
		Error:   "not_implemented",
		Message: notImplementedMessage,
	}
}
