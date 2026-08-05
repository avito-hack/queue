package http

import (
	"context"
	"errors"

	"github.com/avito-hack/queue/services/avito-adapter/gen/server"
	"github.com/avito-hack/queue/services/avito-adapter/internal/usecase"
	"github.com/google/uuid"
)

type HealthChecker interface{ Check(context.Context) error }
type Handler struct {
	healthChecker HealthChecker
	service       *usecase.Service
}

func NewHandler(healthChecker HealthChecker, service *usecase.Service) *Handler {
	return &Handler{healthChecker: healthChecker, service: service}
}

func (h *Handler) GetHealth(ctx context.Context, _ server.GetHealthRequestObject) (server.GetHealthResponseObject, error) {
	if err := h.healthChecker.Check(ctx); err != nil {
		return nil, err
	}
	return server.GetHealth200JSONResponse{Status: "ok"}, nil
}
func (h *Handler) CreateUser(_ context.Context, r server.CreateUserRequestObject) (server.CreateUserResponseObject, error) {
	if r.Body == nil {
		return server.CreateUser400JSONResponse{BadRequestJSONResponse: badRequest(usecase.ErrInvalid)}, nil
	}
	user, err := h.service.CreateUser(r.Body.Name)
	if err != nil {
		return server.CreateUser400JSONResponse{BadRequestJSONResponse: badRequest(err)}, nil
	}
	return server.CreateUser201JSONResponse(toUser(user)), nil
}
func (h *Handler) GetUser(_ context.Context, r server.GetUserRequestObject) (server.GetUserResponseObject, error) {
	user, err := h.service.GetUser(r.UserId.String())
	if err != nil {
		return server.GetUser404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	return server.GetUser200JSONResponse(toUser(user)), nil
}

func (h *Handler) CreateListing(_ context.Context, r server.CreateListingRequestObject) (server.CreateListingResponseObject, error) {
	if r.Body == nil {
		return server.CreateListing400JSONResponse{BadRequestJSONResponse: badRequest(usecase.ErrInvalid)}, nil
	}
	enabled := r.Body.QueueEnabled != nil && *r.Body.QueueEnabled
	listing, err := h.service.CreateListing(r.Body.SellerId.String(), r.Body.Title, r.Body.Price, r.Body.Quantity, enabled)
	if errors.Is(err, usecase.ErrNotFound) {
		return server.CreateListing404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	if err != nil {
		return server.CreateListing400JSONResponse{BadRequestJSONResponse: badRequest(err)}, nil
	}
	return server.CreateListing201JSONResponse(toListing(listing)), nil
}
func (h *Handler) GetListing(_ context.Context, r server.GetListingRequestObject) (server.GetListingResponseObject, error) {
	listing, err := h.service.GetListing(r.ListingId.String())
	if err != nil {
		return server.GetListing404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	return server.GetListing200JSONResponse(toListing(listing)), nil
}
func (h *Handler) UpdateListing(_ context.Context, r server.UpdateListingRequestObject) (server.UpdateListingResponseObject, error) {
	if r.Body == nil {
		return server.UpdateListing400JSONResponse{BadRequestJSONResponse: badRequest(usecase.ErrInvalid)}, nil
	}
	listing, err := h.service.UpdateListing(r.ListingId.String(), r.Body.Title, r.Body.Price)
	if errors.Is(err, usecase.ErrNotFound) {
		return server.UpdateListing404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	if err != nil {
		return server.UpdateListing400JSONResponse{BadRequestJSONResponse: badRequest(err)}, nil
	}
	return server.UpdateListing200JSONResponse(toListing(listing)), nil
}
func (h *Handler) RemoveListing(_ context.Context, r server.RemoveListingRequestObject) (server.RemoveListingResponseObject, error) {
	if err := h.service.RemoveListing(r.ListingId.String()); err != nil {
		return server.RemoveListing404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	return server.RemoveListing204Response{}, nil
}
func (h *Handler) ChangeListingQuantity(_ context.Context, r server.ChangeListingQuantityRequestObject) (server.ChangeListingQuantityResponseObject, error) {
	if r.Body == nil {
		return server.ChangeListingQuantity400JSONResponse{BadRequestJSONResponse: badRequest(usecase.ErrInvalid)}, nil
	}
	listing, err := h.service.ChangeQuantity(r.ListingId.String(), r.Body.Quantity)
	if errors.Is(err, usecase.ErrNotFound) {
		return server.ChangeListingQuantity404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	if err != nil {
		return server.ChangeListingQuantity400JSONResponse{BadRequestJSONResponse: badRequest(err)}, nil
	}
	return server.ChangeListingQuantity200JSONResponse(toListing(listing)), nil
}
func (h *Handler) SetListingQueueEnabled(_ context.Context, r server.SetListingQueueEnabledRequestObject) (server.SetListingQueueEnabledResponseObject, error) {
	if r.Body == nil {
		return server.SetListingQueueEnabled404JSONResponse{NotFoundJSONResponse: notFound(usecase.ErrInvalid)}, nil
	}
	listing, err := h.service.SetQueueEnabled(r.ListingId.String(), r.Body.Enabled)
	if err != nil {
		return server.SetListingQueueEnabled404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	return server.SetListingQueueEnabled200JSONResponse(toListing(listing)), nil
}
func (h *Handler) PauseListing(_ context.Context, r server.PauseListingRequestObject) (server.PauseListingResponseObject, error) {
	listing, err := h.service.PauseListing(r.ListingId.String())
	if err != nil {
		return server.PauseListing404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	return server.PauseListing200JSONResponse(toListing(listing)), nil
}
func (h *Handler) ActivateListing(_ context.Context, r server.ActivateListingRequestObject) (server.ActivateListingResponseObject, error) {
	listing, err := h.service.ActivateListing(r.ListingId.String())
	if errors.Is(err, usecase.ErrNotFound) {
		return server.ActivateListing404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	if err != nil {
		return server.ActivateListing400JSONResponse{BadRequestJSONResponse: badRequest(err)}, nil
	}
	return server.ActivateListing200JSONResponse(toListing(listing)), nil
}

func (h *Handler) CreateReservation(_ context.Context, r server.CreateReservationRequestObject) (server.CreateReservationResponseObject, error) {
	if r.Body == nil {
		return server.CreateReservation400JSONResponse{BadRequestJSONResponse: badRequest(usecase.ErrInvalid)}, nil
	}
	quantity := 1
	if r.Body.Quantity != nil {
		quantity = *r.Body.Quantity
	}
	reservation, err := h.service.CreateReservation(r.Body.ListingId.String(), r.Body.UserId.String(), quantity)
	if errors.Is(err, usecase.ErrNotFound) {
		return server.CreateReservation404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	if errors.Is(err, usecase.ErrUnavailable) || errors.Is(err, usecase.ErrInsufficientStock) {
		return server.CreateReservation409JSONResponse{ConflictJSONResponse: conflict(err)}, nil
	}
	if err != nil {
		return server.CreateReservation400JSONResponse{BadRequestJSONResponse: badRequest(err)}, nil
	}
	return server.CreateReservation201JSONResponse(toReservation(reservation)), nil
}
func (h *Handler) GetReservation(_ context.Context, r server.GetReservationRequestObject) (server.GetReservationResponseObject, error) {
	reservation, err := h.service.GetReservation(r.ReservationId.String())
	if err != nil {
		return server.GetReservation404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	return server.GetReservation200JSONResponse(toReservation(reservation)), nil
}
func (h *Handler) CancelReservation(_ context.Context, r server.CancelReservationRequestObject) (server.CancelReservationResponseObject, error) {
	if err := h.service.CancelReservation(r.ReservationId.String()); err != nil {
		return server.CancelReservation404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	return server.CancelReservation204Response{}, nil
}

func (h *Handler) CreateOrder(_ context.Context, r server.CreateOrderRequestObject) (server.CreateOrderResponseObject, error) {
	if r.Body == nil {
		return server.CreateOrder400JSONResponse{BadRequestJSONResponse: badRequest(usecase.ErrInvalid)}, nil
	}
	order, err := h.service.CreateOrder(r.Body.ReservationId.String(), r.Body.UserId.String())
	if errors.Is(err, usecase.ErrNotFound) {
		return server.CreateOrder404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	if err != nil {
		return server.CreateOrder409JSONResponse{ConflictJSONResponse: conflict(err)}, nil
	}
	return server.CreateOrder201JSONResponse(toOrder(order)), nil
}
func (h *Handler) GetOrder(_ context.Context, r server.GetOrderRequestObject) (server.GetOrderResponseObject, error) {
	order, err := h.service.GetOrder(r.OrderId.String())
	if err != nil {
		return server.GetOrder404JSONResponse{NotFoundJSONResponse: notFound(err)}, nil
	}
	return server.GetOrder200JSONResponse(toOrder(order)), nil
}

func badRequest(err error) server.BadRequestJSONResponse {
	return server.BadRequestJSONResponse{Message: err.Error()}
}
func notFound(err error) server.NotFoundJSONResponse {
	return server.NotFoundJSONResponse{Message: err.Error()}
}
func conflict(err error) server.ConflictJSONResponse {
	return server.ConflictJSONResponse{Message: err.Error()}
}
func toUser(value usecase.User) server.User {
	return server.User{Id: uuid.MustParse(value.ID), Name: value.Name, CreatedAt: value.CreatedAt}
}
func toListing(value usecase.Listing) server.Listing {
	return server.Listing{Id: uuid.MustParse(value.ID), SellerId: uuid.MustParse(value.SellerID), Title: value.Title, Price: value.Price, Quantity: value.Quantity, ReservedQuantity: value.ReservedQuantity, AvailableQuantity: value.AvailableQuantity(), QueueEnabled: value.QueueEnabled, Status: server.ListingStatus(value.Status), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func toReservation(value usecase.Reservation) server.Reservation {
	return server.Reservation{Id: uuid.MustParse(value.ID), ListingId: uuid.MustParse(value.ListingID), UserId: uuid.MustParse(value.UserID), Quantity: value.Quantity, Status: server.ReservationStatus(value.Status), CreatedAt: value.CreatedAt}
}
func toOrder(value usecase.Order) server.Order {
	return server.Order{Id: uuid.MustParse(value.ID), ReservationId: uuid.MustParse(value.ReservationID), ListingId: uuid.MustParse(value.ListingID), UserId: uuid.MustParse(value.UserID), Status: server.Created, CreatedAt: value.CreatedAt}
}
