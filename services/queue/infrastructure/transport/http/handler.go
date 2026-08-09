package http

import (
    "context"
    "errors"
    "log"

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
}

func NewHandler(healthChecker HealthChecker, queueService usecase.ItemQueueService) *Handler {
    return &Handler{
        healthChecker: healthChecker,
        queueService:  queueService,
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
            return server.GetHealth500JSONResponse(internalError(err)), nil
        }
    }
    return server.GetHealth200JSONResponse{
        Status: "ok",
    }, nil
}

func (h *Handler) Enqueue(ctx context.Context, request server.EnqueueRequestObject) (server.EnqueueResponseObject, error) {
    rc := requestContext(ctx)
    log.Printf("Handler.Enqueue: incoming ctx=%T %p requestCtx=%T %p", ctx, ctx, rc, rc)

    userID, ok := getUserIDFromContext(ctx)
    if !ok {
        return server.Enqueue401JSONResponse(unauthorized(errUserIDNotFound)), nil
    }

    if _, ok := getAuthorizationHeaderFromContext(ctx); !ok {
        return server.Enqueue500JSONResponse(internalError(errors.New("authorization header not found in context"))), nil
    }

    err := h.queueService.Enqueue(rc, request.ItemID, userID)
    switch {
    case errors.Is(err, usecase.ErrQueueNotFound):
        return server.Enqueue404JSONResponse(notFound(err)), nil
    case errors.Is(err, usecase.ErrUserAlreadyInQueue):
        return server.Enqueue409JSONResponse(conflict(err)), nil
    case errors.Is(err, usecase.ErrUserHasActiveTicket):
        return server.Enqueue409JSONResponse(conflict(err)), nil
    case errors.Is(err, usecase.ErrQueueUnavailable):
        return server.Enqueue422JSONResponse(unavailable(err)), nil
    case err != nil:
        return server.Enqueue500JSONResponse(internalError(err)), nil
    }
    return server.Enqueue201Response{}, nil
}

func (h *Handler) Dequeue(ctx context.Context, request server.DequeueRequestObject) (server.DequeueResponseObject, error) {
    rc := requestContext(ctx)
    log.Printf("Handler.Dequeue: incoming ctx=%T %p requestCtx=%T %p", ctx, ctx, rc, rc)

    userID, ok := getUserIDFromContext(ctx)
    if !ok {
        return server.Dequeue401JSONResponse(unauthorized(errUserIDNotFound)), nil
    }

    err := h.queueService.Dequeue(rc, request.ItemID, userID)
    switch {
    case errors.Is(err, usecase.ErrUserNotInQueue):
        return server.Dequeue404JSONResponse(notFound(err)), nil
    case errors.Is(err, usecase.ErrUserCannotLeaveQueue):
        return server.Dequeue409JSONResponse(conflict(err)), nil
    case err != nil:
        return server.Dequeue500JSONResponse(internalError(err)), nil
    }
    return server.Dequeue204Response{}, nil
}

func (h *Handler) ClearItemQueue(ctx context.Context, request server.ClearItemQueueRequestObject) (server.ClearItemQueueResponseObject, error) {
    rc := requestContext(ctx)
    log.Printf("Handler.ClearItemQueue: incoming ctx=%T %p requestCtx=%T %p", ctx, ctx, rc, rc)
    err := h.queueService.ClearItemQueue(rc, request.ItemID)
    switch {
    case errors.Is(err, usecase.ErrQueueNotFound):
        return server.ClearItemQueue404JSONResponse(notFound(err)), nil
    case err != nil:
        return server.ClearItemQueue500JSONResponse(internalError(err)), nil
    }
    return server.ClearItemQueue204Response{}, nil
}

func (h *Handler) GetUserPosition(ctx context.Context, request server.GetUserPositionRequestObject) (server.GetUserPositionResponseObject, error) {
    rc := requestContext(ctx)
    log.Printf("Handler.GetUserPosition: incoming ctx=%T %p requestCtx=%T %p", ctx, ctx, rc, rc)

    userID, ok := getUserIDFromContext(ctx)
    if !ok {
        return server.GetUserPosition401JSONResponse(unauthorized(errUserIDNotFound)), nil
    }

    position, err := h.queueService.GetUserPosition(rc, request.ItemID, userID)
    switch {
    case errors.Is(err, usecase.ErrUserNotInQueue):
        return server.GetUserPosition404JSONResponse(notFound(err)), nil
    case err != nil:
        return server.GetUserPosition500JSONResponse(internalError(err)), nil
    }
    return server.GetUserPosition200JSONResponse{
        Position: int(position),
    }, nil
}

func (h *Handler) GetItemQueueState(ctx context.Context, request server.GetItemQueueStateRequestObject) (server.GetItemQueueStateResponseObject, error) {
    rc := requestContext(ctx)
    log.Printf("Handler.GetItemQueueState: incoming ctx=%T %p requestCtx=%T %p", ctx, ctx, rc, rc)
    state, err := h.queueService.GetItemQueueState(rc, request.ItemID)
    switch {
    case errors.Is(err, usecase.ErrQueueNotFound):
        return server.GetItemQueueState404JSONResponse(notFound(err)), nil
    case err != nil:
        return server.GetItemQueueState500JSONResponse(internalError(err)), nil
    }
    return server.GetItemQueueState200JSONResponse{
        State: server.ItemQueueState(state),
    }, nil
}

func (h *Handler) GetUserQueues(ctx context.Context, _ server.GetUserQueuesRequestObject) (server.GetUserQueuesResponseObject, error) {
    rc := requestContext(ctx)
    log.Printf("Handler.GetUserQueues: incoming ctx=%T %p requestCtx=%T %p", ctx, ctx, rc, rc)

    userID, ok := getUserIDFromContext(ctx)
    if !ok {
        return server.GetUserQueues401JSONResponse(unauthorized(errUserIDNotFound)), nil
    }

    queues, err := h.queueService.GetUserQueues(rc, userID)
    if err != nil {
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
