package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/infrastructure/client/avitoadapter"
	identityauth "github.com/avito-hack/queue/services/tickets/internal/auth"
	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type ticketListerStub struct {
	tickets        []domain.Ticket
	err            error
	receivedUserID uuid.UUID
	receivedFilter usecase.ListTicketsFilter
	calls          int
}

type userTokenResolverStub struct {
	userID        uuid.UUID
	err           error
	receivedToken string
}

type authRoundTripFunc func(*http.Request) (*http.Response, error)

func (f authRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func (s *userTokenResolverStub) ResolveUserID(_ context.Context, token string) (uuid.UUID, error) {
	s.receivedToken = token

	return s.userID, s.err
}

func (s *ticketListerStub) List(_ context.Context, userID uuid.UUID, filter usecase.ListTicketsFilter) ([]domain.Ticket, error) {
	s.calls++
	s.receivedUserID = userID
	s.receivedFilter = filter

	return s.tickets, s.err
}

func Test_GetHealthz_ReturnOK(t *testing.T) {
	// given
	router, err := NewRouter(
		NewHandler(usecase.NewHealth(), &ticketListerStub{}),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, recorder.Body.String())
}

func Test_GetV1TicketList_ReturnTickets(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	orderID := uuid.New()
	issuedAt := time.Date(2026, time.August, 6, 10, 0, 0, 0, time.UTC)
	activationDeadline := issuedAt.Add(15 * time.Minute)
	activatedAt := issuedAt.Add(time.Minute)
	finishedAt := issuedAt.Add(10 * time.Minute)
	checkoutURL := "/checkout/1"
	closeReason := domain.TicketCloseReasonPaymentSucceeded
	lister := &ticketListerStub{tickets: []domain.Ticket{
		{
			ID:                 ticketID,
			ListingID:          listingID,
			SKUID:              skuID,
			Status:             domain.TicketStatusRedeemed,
			IssuedAt:           issuedAt,
			ActivationDeadline: activationDeadline,
			ActivatedAt:        &activatedAt,
			OrderID:            &orderID,
			CheckoutURL:        &checkoutURL,
			FinishedAt:         &finishedAt,
			CloseReason:        &closeReason,
			AvailableActions:   []domain.TicketAvailableAction{},
		},
	}}
	resolver := &userTokenResolverStub{userID: userID}
	router, err := NewRouter(NewHandler(usecase.NewHealth(), lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(
		http.MethodGet,
		"/v1/ticket/list?status=redeemed&listing_id="+listingID.String()+"&sku_id="+skuID.String(),
		nil,
	)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{
		"ticket": [{
			"id": "`+ticketID.String()+`",
			"listing_id": "`+listingID.String()+`",
			"sku_id": "`+skuID.String()+`",
			"status": "redeemed",
			"issued_at": "2026-08-06T10:00:00Z",
			"activation_deadline": "2026-08-06T10:15:00Z",
			"activated_at": "2026-08-06T10:01:00Z",
			"order_id": "`+orderID.String()+`",
			"checkout_url": "/checkout/1",
			"finished_at": "2026-08-06T10:10:00Z",
			"finish_reason": "payment_succeeded",
			"available_actions": []
		}]
	}`, recorder.Body.String())
	assert.Equal(t, userID, lister.receivedUserID)
	assert.Equal(t, "abc-token", resolver.receivedToken)
	require.NotNil(t, lister.receivedFilter.Status)
	assert.Equal(t, domain.TicketStatusRedeemed, *lister.receivedFilter.Status)
	require.NotNil(t, lister.receivedFilter.ListingID)
	assert.Equal(t, listingID, *lister.receivedFilter.ListingID)
	require.NotNil(t, lister.receivedFilter.SKUID)
	assert.Equal(t, skuID, *lister.receivedFilter.SKUID)
}

func Test_GetV1TicketList_WithoutTickets_ReturnEmptyArray(t *testing.T) {
	// given
	userID := uuid.New()
	lister := &ticketListerStub{tickets: []domain.Ticket{}}
	router, err := NewRouter(
		NewHandler(usecase.NewHealth(), lister),
		&userTokenResolverStub{userID: userID},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"ticket":[]}`, recorder.Body.String())
}

func Test_GetV1TicketList_WithoutToken_ReturnUnauthorized(t *testing.T) {
	// given
	lister := &ticketListerStub{}
	router, err := NewRouter(
		NewHandler(usecase.NewHealth(), lister),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.JSONEq(t, `{"error":"unauthorized","message":"bearer token is required"}`, recorder.Body.String())
	assert.Zero(t, lister.calls)
}

func Test_GetV1TicketList_WithInvalidToken_ReturnUnauthorized(t *testing.T) {
	// given
	lister := &ticketListerStub{}
	resolver := &userTokenResolverStub{err: identityauth.ErrInvalidToken}
	router, err := NewRouter(NewHandler(usecase.NewHealth(), lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.JSONEq(t, `{"error":"unauthorized","message":"bearer token is invalid"}`, recorder.Body.String())
	assert.Zero(t, lister.calls)
}

func Test_GetV1TicketList_AuthServiceReturnsError_ReturnInternalError(t *testing.T) {
	// given
	lister := &ticketListerStub{}
	resolver := &userTokenResolverStub{err: errors.New("adapter unavailable")}
	router, err := NewRouter(NewHandler(usecase.NewHealth(), lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"error":"internal_error","message":"internal server error"}`, recorder.Body.String())
	assert.Zero(t, lister.calls)
}

func Test_GetV1TicketList_WithCustomTokenResolver_ReturnTicketsForResolvedUser(t *testing.T) {
	// given
	userID := uuid.New()
	lister := &ticketListerStub{tickets: []domain.Ticket{}}
	resolver := &userTokenResolverStub{userID: userID}
	router, err := NewRouter(NewHandler(usecase.NewHealth(), lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer   opaque-token ")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "opaque-token", resolver.receivedToken)
	assert.Equal(t, userID, lister.receivedUserID)
}

func Test_GetV1TicketList_WithAvitoAdapterResolver_ReturnTicketsForResolvedUser(t *testing.T) {
	// given
	userID := uuid.New()
	lister := &ticketListerStub{tickets: []domain.Ticket{}}
	client := &http.Client{Transport: authRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body struct {
			Token string `json:"token"`
		}
		assert.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		assert.Equal(t, "abc-token", body.Token)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"user_id":"` + userID.String() + `"}`)),
		}, nil
	})}
	resolver, err := avitoadapter.NewUserTokenResolver("http://avito-adapter:8080", client)
	require.NoError(t, err)
	router, err := NewRouter(NewHandler(usecase.NewHealth(), lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, userID, lister.receivedUserID)
}

func Test_GetV1TicketList_AvitoAdapterRejectsToken_ReturnUnauthorized(t *testing.T) {
	// given
	lister := &ticketListerStub{}
	client := &http.Client{Transport: authRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}
	resolver, err := avitoadapter.NewUserTokenResolver("http://avito-adapter:8080", client)
	require.NoError(t, err)
	router, err := NewRouter(NewHandler(usecase.NewHealth(), lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.JSONEq(t, `{"error":"unauthorized","message":"bearer token is invalid"}`, recorder.Body.String())
	assert.Zero(t, lister.calls)
}

func Test_GetV1TicketList_WithInvalidFilter_ReturnBadRequest(t *testing.T) {
	// given
	tests := []struct {
		name  string
		query string
	}{
		{name: "status", query: "status=unknown"},
		{name: "auth unavailable phrase", query: "status=user%20identity%20service%20unavailable"},
		{name: "invalid bearer phrase", query: "status=bearer%20token%20is%20invalid"},
		{name: "listing id", query: "listing_id=invalid"},
		{name: "sku id", query: "sku_id=invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lister := &ticketListerStub{}
			router, err := NewRouter(
				NewHandler(usecase.NewHealth(), lister),
				&userTokenResolverStub{userID: uuid.New()},
			)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list?"+test.query, nil)
			request.Header.Set("Authorization", "Bearer "+uuid.NewString())
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Zero(t, lister.calls)
		})
	}
}

func Test_GetV1TicketList_UsecaseReturnsError_ReturnInternalError(t *testing.T) {
	// given
	lister := &ticketListerStub{err: errors.New("database failed")}
	router, err := NewRouter(
		NewHandler(usecase.NewHealth(), lister),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer "+uuid.NewString())
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"error":"internal_error","message":"internal server error"}`, recorder.Body.String())
}
