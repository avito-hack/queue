package http

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/avito-hack/queue/services/queue/gen/server"
	"github.com/avito-hack/queue/services/queue/internal/usecase"
)

var errUserIDNotFound = errors.New("user id not found in context")

type HealthChecker interface {
	Check(context.Context) error
}

type Handler struct {
	healthChecker HealthChecker
	queueService  usecase.ItemQueueService
	logger        *slog.Logger
}

func NewHandler(healthChecker HealthChecker, queueService usecase.ItemQueueService, logger *slog.Logger) *Handler {
	return &Handler{
		healthChecker: healthChecker,
		queueService:  queueService,
		logger:        logger,
	}
}

func requestContext(ctx context.Context) context.Context {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		return ginCtx.Request.Context()
	}

	return ctx
}

func getUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		if v, exists := ginCtx.Get(authenticatedUserIDKey); exists {
			switch val := v.(type) {
			case uuid.UUID:
				if val != uuid.Nil {
					return val, true
				}
			case string:
				if id, err := uuid.Parse(val); err == nil && id != uuid.Nil {
					return id, true
				}
			}
		}

		if s, ok := ginCtx.Request.Context().Value("user_id").(string); ok {
			if id, err := uuid.Parse(s); err == nil && id != uuid.Nil {
				return id, true
			}
		}
	}

	return uuid.Nil, false
}

func getAuthorizationHeaderFromContext(ctx context.Context) (string, bool) {
	if ginCtx, ok := ctx.(*gin.Context); ok {
		h := ginCtx.GetHeader("Authorization")

		if h != "" {
			return h, true
		}

		if v := ginCtx.Request.Context().Value("authorization_header"); v != nil {
			if hs, ok := v.(string); ok && hs != "" {
				return hs, true
			}
		}
	}

	return "", false
}

func (h *Handler) GetHealth(ctx context.Context, _ server.GetHealthRequestObject) (server.GetHealthResponseObject, error) {
	rc := requestContext(ctx)

	if h.healthChecker != nil {
		if err := h.healthChecker.Check(rc); err != nil {
			h.logger.Error("health check failed", "error", err)
			return server.GetHealth500JSONResponse(internalError(err)), nil
		}
	}

	return server.GetHealth200JSONResponse{
		Status: "ok",
	}, nil
}

func (h *Handler) Enqueue(ctx context.Context, request server.EnqueueRequestObject) (server.EnqueueResponseObject, error) {
	rc := requestContext(ctx)

	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		h.logger.Warn("enqueue user id missing", "item_id", request.ItemID)
		return server.Enqueue401JSONResponse(unauthorized(errUserIDNotFound)), nil
	}

	h.logger.Info("enqueue started", "item_id", request.ItemID, "user_id", userID)

	if _, ok := getAuthorizationHeaderFromContext(ctx); !ok {
		err := errors.New("authorization header not found in context")

		h.logger.Error("authorization header missing", "error", err, "item_id", request.ItemID, "user_id", userID)

		return server.Enqueue500JSONResponse(internalError(err)), nil
	}

	err := h.queueService.Enqueue(rc, request.ItemID, userID)

	switch {
	case errors.Is(err, usecase.ErrQueueNotFound):
		h.logger.Warn("enqueue queue not found", "item_id", request.ItemID)
		return server.Enqueue404JSONResponse(notFound(err)), nil

	case errors.Is(err, usecase.ErrUserAlreadyInQueue):
		h.logger.Warn("enqueue user already in queue", "item_id", request.ItemID, "user_id", userID)
		return server.Enqueue409JSONResponse(conflict(err)), nil

	case errors.Is(err, usecase.ErrUserHasActiveTicket):
		h.logger.Warn("enqueue user has active ticket", "item_id", request.ItemID, "user_id", userID)
		return server.Enqueue409JSONResponse(conflict(err)), nil

	case errors.Is(err, usecase.ErrQueueUnavailable):
		h.logger.Warn("enqueue queue unavailable", "item_id", request.ItemID)
		return server.Enqueue422JSONResponse(unavailable(err)), nil

	case err != nil:
		h.logger.Error("enqueue internal error", "error", err, "item_id", request.ItemID, "user_id", userID)
		return server.Enqueue500JSONResponse(internalError(err)), nil
	}

	h.logger.Info("enqueue completed", "item_id", request.ItemID, "user_id", userID)

	return server.Enqueue201Response{}, nil
}

func (h *Handler) Dequeue(ctx context.Context, request server.DequeueRequestObject) (server.DequeueResponseObject, error) {
	rc := requestContext(ctx)

	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		h.logger.Warn("dequeue user id missing", "item_id", request.ItemID)
		return server.Dequeue401JSONResponse(unauthorized(errUserIDNotFound)), nil
	}

	h.logger.Info("dequeue started", "item_id", request.ItemID, "user_id", userID)

	err := h.queueService.Dequeue(rc, request.ItemID, userID)

	switch {
	case errors.Is(err, usecase.ErrUserNotInQueue):
		h.logger.Warn("dequeue user not in queue", "item_id", request.ItemID, "user_id", userID)
		return server.Dequeue404JSONResponse(notFound(err)), nil

	case errors.Is(err, usecase.ErrUserCannotLeaveQueue):
		h.logger.Warn("dequeue user cannot leave queue", "item_id", request.ItemID, "user_id", userID)
		return server.Dequeue409JSONResponse(conflict(err)), nil

	case err != nil:
		h.logger.Error("dequeue internal error", "error", err, "item_id", request.ItemID, "user_id", userID)
		return server.Dequeue500JSONResponse(internalError(err)), nil
	}

	h.logger.Info("dequeue completed", "item_id", request.ItemID, "user_id", userID)

	return server.Dequeue204Response{}, nil
}

func (h *Handler) ClearItemQueue(ctx context.Context, request server.ClearItemQueueRequestObject) (server.ClearItemQueueResponseObject, error) {
	rc := requestContext(ctx)

	h.logger.Info("clear queue started", "item_id", request.ItemID)

	err := h.queueService.ClearItemQueue(rc, request.ItemID)

	switch {
	case errors.Is(err, usecase.ErrQueueNotFound):
		h.logger.Warn("clear queue not found", "item_id", request.ItemID)
		return server.ClearItemQueue404JSONResponse(notFound(err)), nil

	case err != nil:
		h.logger.Error("clear queue internal error", "error", err, "item_id", request.ItemID)
		return server.ClearItemQueue500JSONResponse(internalError(err)), nil
	}

	h.logger.Info("clear queue completed", "item_id", request.ItemID)

	return server.ClearItemQueue204Response{}, nil
}

func (h *Handler) GetUserPosition(ctx context.Context, request server.GetUserPositionRequestObject) (server.GetUserPositionResponseObject, error) {
	rc := requestContext(ctx)

	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		h.logger.Warn("get user position user id missing", "item_id", request.ItemID)
		return server.GetUserPosition401JSONResponse(unauthorized(errUserIDNotFound)), nil
	}

	h.logger.Info("get user position started", "item_id", request.ItemID, "user_id", userID)

	position, err := h.queueService.GetUserPosition(rc, request.ItemID, userID)

	switch {
	case errors.Is(err, usecase.ErrUserNotInQueue):
		h.logger.Warn("get user position user not in queue", "item_id", request.ItemID, "user_id", userID)
		return server.GetUserPosition404JSONResponse(notFound(err)), nil

	case err != nil:
		h.logger.Error("get user position internal error", "error", err, "item_id", request.ItemID, "user_id", userID)
		return server.GetUserPosition500JSONResponse(internalError(err)), nil
	}

	h.logger.Info("get user position completed", "item_id", request.ItemID, "user_id", userID, "position", position)

	return server.GetUserPosition200JSONResponse{
		Position: int(position),
	}, nil
}

func (h *Handler) GetItemQueueState(ctx context.Context, request server.GetItemQueueStateRequestObject) (server.GetItemQueueStateResponseObject, error) {
	rc := requestContext(ctx)

	h.logger.Info("get queue state started", "item_id", request.ItemID)

	info, err := h.queueService.GetItemQueueState(rc, request.ItemID)

	switch {
	case errors.Is(err, usecase.ErrQueueNotFound):
		h.logger.Warn("get queue state not found", "item_id", request.ItemID)
		return server.GetItemQueueState404JSONResponse(notFound(err)), nil

	case err != nil:
		h.logger.Error("get queue state internal error", "error", err, "item_id", request.ItemID)
		return server.GetItemQueueState500JSONResponse(internalError(err)), nil
	}

	h.logger.Info(
		"get queue state completed",
		"item_id",
		request.ItemID,
		"state",
		info.State,
		"waiting_count",
		info.WaitingCount,
	)

	return server.GetItemQueueState200JSONResponse{
		State:        server.ItemQueueState(info.State),
		WaitingCount: info.WaitingCount,
	}, nil
}

func (h *Handler) GetUserQueues(ctx context.Context, _ server.GetUserQueuesRequestObject) (server.GetUserQueuesResponseObject, error) {
	rc := requestContext(ctx)

	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		h.logger.Warn("get user queues user id missing")
		return server.GetUserQueues401JSONResponse(unauthorized(errUserIDNotFound)), nil
	}

	h.logger.Info("get user queues started", "user_id", userID)

	queues, err := h.queueService.GetUserQueues(rc, userID)

	if err != nil {
		h.logger.Error("get user queues internal error", "error", err, "user_id", userID)
		return server.GetUserQueues500JSONResponse(internalError(err)), nil
	}

	response := make(server.GetUserQueues200JSONResponse, 0, len(queues))

	for _, queue := range queues {
		response = append(response, server.ItemQueueInfo{
			ItemId:   queue.ItemID,
			Position: queue.Position,
			Status:   server.ItemQueueMemberStatus(queue.Status),
		})
	}

	h.logger.Info("get user queues completed", "user_id", userID, "count", len(response))

	return response, nil
}

func unauthorized(err error) server.ErrorResponse {
	return server.ErrorResponse{
		Code:    "unauthorized",
		Message: err.Error(),
	}
}

func notFound(err error) server.ErrorResponse {
	return server.ErrorResponse{
		Code:    "not_found",
		Message: err.Error(),
	}
}

func conflict(err error) server.ErrorResponse {
	return server.ErrorResponse{
		Code:    "conflict",
		Message: err.Error(),
	}
}

func unavailable(err error) server.ErrorResponse {
	return server.ErrorResponse{
		Code:    "queue_unavailable",
		Message: err.Error(),
	}
}

func internalError(err error) server.ErrorResponse {
	return server.ErrorResponse{
		Code:    "internal_error",
		Message: err.Error(),
	}
}
