package http

import (
	"context"

	"github.com/avito-hack/queue/services/tickets/gen/server"
)

const notImplementedMessage = "tickets service is not implemented"

type HealthChecker interface {
	Check(context.Context) error
}

type Handler struct {
	healthChecker HealthChecker
}

func NewHandler(healthChecker HealthChecker) *Handler {
	return &Handler{healthChecker: healthChecker}
}

func (h *Handler) GetHealth(ctx context.Context, _ server.GetHealthRequestObject) (server.GetHealthResponseObject, error) {
	if err := h.healthChecker.Check(ctx); err != nil {
		return nil, err
	}

	return server.GetHealth200Response{}, nil
}

func (h *Handler) IssueTicket(context.Context, server.IssueTicketRequestObject) (server.IssueTicketResponseObject, error) {
	return server.IssueTicket500JSONResponse{InternalErrorJSONResponse: internalError()}, nil
}

func (h *Handler) ListTickets(context.Context, server.ListTicketsRequestObject) (server.ListTicketsResponseObject, error) {
	return server.ListTickets500JSONResponse{InternalErrorJSONResponse: internalError()}, nil
}

func (h *Handler) GetTicket(context.Context, server.GetTicketRequestObject) (server.GetTicketResponseObject, error) {
	return server.GetTicket500JSONResponse{InternalErrorJSONResponse: internalError()}, nil
}

func (h *Handler) ActivateTicket(context.Context, server.ActivateTicketRequestObject) (server.ActivateTicketResponseObject, error) {
	return server.ActivateTicket500JSONResponse{InternalErrorJSONResponse: internalError()}, nil
}

func (h *Handler) DeclineTicket(context.Context, server.DeclineTicketRequestObject) (server.DeclineTicketResponseObject, error) {
	return server.DeclineTicket500JSONResponse{InternalErrorJSONResponse: internalError()}, nil
}

func internalError() server.InternalErrorJSONResponse {
	return server.InternalErrorJSONResponse{
		Error:   "not_implemented",
		Message: notImplementedMessage,
	}
}
