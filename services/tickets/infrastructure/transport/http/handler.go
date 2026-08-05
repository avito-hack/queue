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

func (h *Handler) GetHealthz(ctx context.Context, _ server.GetHealthzRequestObject) (server.GetHealthzResponseObject, error) {
	if err := h.healthChecker.Check(ctx); err != nil {
		return nil, err
	}

	return server.GetHealthz200Response{}, nil
}

func (h *Handler) PostInternalV1TicketIssue(context.Context, server.PostInternalV1TicketIssueRequestObject) (server.PostInternalV1TicketIssueResponseObject, error) {
	return server.PostInternalV1TicketIssue500JSONResponse{InternalErrorJSONResponse: internalError()}, nil
}

func (h *Handler) GetV1TicketList(context.Context, server.GetV1TicketListRequestObject) (server.GetV1TicketListResponseObject, error) {
	return server.GetV1TicketList500JSONResponse{InternalErrorJSONResponse: internalError()}, nil
}

func (h *Handler) GetV1TicketTicketId(context.Context, server.GetV1TicketTicketIdRequestObject) (server.GetV1TicketTicketIdResponseObject, error) {
	return server.GetV1TicketTicketId500JSONResponse{InternalErrorJSONResponse: internalError()}, nil
}

func (h *Handler) PostV1TicketTicketIdActivate(context.Context, server.PostV1TicketTicketIdActivateRequestObject) (server.PostV1TicketTicketIdActivateResponseObject, error) {
	return server.PostV1TicketTicketIdActivate500JSONResponse{InternalErrorJSONResponse: internalError()}, nil
}

func (h *Handler) PostV1TicketTicketIdDecline(context.Context, server.PostV1TicketTicketIdDeclineRequestObject) (server.PostV1TicketTicketIdDeclineResponseObject, error) {
	return server.PostV1TicketTicketIdDecline500JSONResponse{InternalErrorJSONResponse: internalError()}, nil
}

func internalError() server.InternalErrorJSONResponse {
	return server.InternalErrorJSONResponse{
		Error:   "not_implemented",
		Message: notImplementedMessage,
	}
}
