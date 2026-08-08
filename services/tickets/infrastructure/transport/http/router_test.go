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
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/gen/server"
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

type ticketGetterStub struct {
	ticket           domain.Ticket
	err              error
	receivedUserID   uuid.UUID
	receivedTicketID uuid.UUID
	calls            int
}

type ticketActivatorStub struct {
	result                 usecase.ActivationResult
	err                    error
	receivedUserID         uuid.UUID
	receivedTicketID       uuid.UUID
	receivedIdempotencyKey uuid.UUID
	calls                  int
}

type ticketDeclinerStub struct {
	result                 usecase.DeclineTicketResult
	err                    error
	receivedUserID         uuid.UUID
	receivedTicketID       uuid.UUID
	receivedIdempotencyKey uuid.UUID
	calls                  int
}

type ticketIssuerStub struct {
	result                 usecase.IssueTicketResult
	err                    error
	receivedRequest        usecase.IssueTicketRequest
	receivedIdempotencyKey uuid.UUID
	calls                  int
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

func (s *ticketGetterStub) Get(_ context.Context, userID uuid.UUID, ticketID uuid.UUID) (domain.Ticket, error) {
	s.calls++
	s.receivedUserID = userID
	s.receivedTicketID = ticketID

	return s.ticket, s.err
}

func (s *ticketActivatorStub) Activate(
	_ context.Context,
	userID uuid.UUID,
	ticketID uuid.UUID,
	idempotencyKey uuid.UUID,
) (usecase.ActivationResult, error) {
	s.calls++
	s.receivedUserID = userID
	s.receivedTicketID = ticketID
	s.receivedIdempotencyKey = idempotencyKey

	return s.result, s.err
}

func (s *ticketDeclinerStub) Decline(
	_ context.Context,
	userID uuid.UUID,
	ticketID uuid.UUID,
	idempotencyKey uuid.UUID,
) (usecase.DeclineTicketResult, error) {
	s.calls++
	s.receivedUserID = userID
	s.receivedTicketID = ticketID
	s.receivedIdempotencyKey = idempotencyKey

	return s.result, s.err
}

func (s *ticketIssuerStub) Issue(
	_ context.Context,
	request usecase.IssueTicketRequest,
	idempotencyKey uuid.UUID,
) (usecase.IssueTicketResult, error) {
	s.calls++
	s.receivedRequest = request
	s.receivedIdempotencyKey = idempotencyKey

	return s.result, s.err
}

func newTestHandler(lister TicketLister, getters ...TicketGetter) *Handler {
	getter := TicketGetter(&ticketGetterStub{})
	if len(getters) > 0 {
		getter = getters[0]
	}

	return NewHandler(
		usecase.NewHealth(),
		lister,
		getter,
		&ticketActivatorStub{},
		&ticketDeclinerStub{},
		&ticketIssuerStub{},
	)
}

func newTestHandlerWithActivator(activator TicketActivator) *Handler {
	return NewHandler(
		usecase.NewHealth(),
		&ticketListerStub{},
		&ticketGetterStub{},
		activator,
		&ticketDeclinerStub{},
		&ticketIssuerStub{},
	)
}

func newTestHandlerWithDecliner(decliner TicketDecliner) *Handler {
	return NewHandler(
		usecase.NewHealth(),
		&ticketListerStub{},
		&ticketGetterStub{},
		&ticketActivatorStub{},
		decliner,
		&ticketIssuerStub{},
	)
}

func newTestHandlerWithIssuer(issuer TicketIssuer) *Handler {
	return NewHandler(
		usecase.NewHealth(),
		&ticketListerStub{},
		&ticketGetterStub{},
		&ticketActivatorStub{},
		&ticketDeclinerStub{},
		issuer,
	)
}

func Test_GetHealthz_ReturnOK(t *testing.T) {
	// given
	router, err := NewRouter(
		newTestHandler(&ticketListerStub{}),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Body.String())
}

func Test_PostInternalV1TicketIssue_ReturnTicket(t *testing.T) {
	// given
	tests := []struct {
		name       string
		created    bool
		statusCode int
	}{
		{name: "new ticket", created: true, statusCode: http.StatusCreated},
		{name: "replayed ticket", created: false, statusCode: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			queueEntryID := uuid.New()
			userID := uuid.New()
			ticketID := uuid.New()
			listingID := uuid.New()
			skuID := uuid.New()
			idempotencyKey := uuid.New()
			issuedAt := time.Date(2026, time.August, 8, 10, 0, 0, 0, time.UTC)
			activationDeadline := issuedAt.Add(15 * time.Minute)
			issuer := &ticketIssuerStub{result: usecase.IssueTicketResult{
				Ticket: domain.Ticket{
					ID:                 ticketID,
					ListingID:          listingID,
					SKUID:              skuID,
					Status:             domain.TicketStatusIssued,
					IssuedAt:           issuedAt,
					ActivationDeadline: activationDeadline,
					AvailableActions: []domain.TicketAvailableAction{
						domain.TicketAvailableActionActivate,
						domain.TicketAvailableActionDecline,
					},
				},
				QueueEntryID: queueEntryID,
				UserID:       userID,
				Created:      test.created,
			}}
			resolver := &userTokenResolverStub{userID: uuid.New()}
			router, err := NewRouter(newTestHandlerWithIssuer(issuer), resolver)
			require.NoError(t, err)
			request := httptest.NewRequest(
				http.MethodPost,
				"/internal/v1/ticket/issue",
				strings.NewReader(`{
					"queue_entry_id":"`+queueEntryID.String()+`",
					"user_id":"`+userID.String()+`",
					"listing_id":"`+listingID.String()+`",
					"sku_id":"`+skuID.String()+`"
				}`),
			)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Idempotency-Key", idempotencyKey.String())
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			require.Equal(t, test.statusCode, recorder.Code)
			require.JSONEq(t, `{
				"id":"`+ticketID.String()+`",
				"listing_id":"`+listingID.String()+`",
				"sku_id":"`+skuID.String()+`",
				"status":"issued",
				"issued_at":"2026-08-08T10:00:00Z",
				"activation_deadline":"2026-08-08T10:15:00Z",
				"activated_at":null,
				"order_id":null,
				"checkout_url":null,
				"finished_at":null,
				"finish_reason":null,
				"available_actions":["activate","decline"]
			}`, recorder.Body.String())
			require.Equal(t, usecase.IssueTicketRequest{
				QueueEntryID: queueEntryID,
				UserID:       userID,
				ListingID:    listingID,
				SKUID:        skuID,
			}, issuer.receivedRequest)
			require.Equal(t, idempotencyKey, issuer.receivedIdempotencyKey)
			require.Equal(t, 1, issuer.calls)
			require.Empty(t, resolver.receivedToken)
		})
	}
}

func Test_PostInternalV1TicketIssue_UsecaseReturnsDomainError_ReturnMappedError(t *testing.T) {
	// given
	tests := []struct {
		name       string
		err        error
		statusCode int
		response   string
	}{
		{
			name:       "invalid issue",
			err:        usecase.ErrInvalidTicketIssue,
			statusCode: http.StatusBadRequest,
			response:   `{"error":"bad_request","message":"` + usecase.ErrInvalidTicketIssue.Error() + `"}`,
		},
		{
			name:       "ticket not issuable",
			err:        usecase.ErrTicketNotIssuable,
			statusCode: http.StatusConflict,
			response:   `{"error":"ticket_not_issuable","message":"` + usecase.ErrTicketNotIssuable.Error() + `"}`,
		},
		{
			name:       "idempotency conflict",
			err:        usecase.ErrIdempotencyConflict,
			statusCode: http.StatusConflict,
			response:   `{"error":"idempotency_conflict","message":"` + usecase.ErrIdempotencyConflict.Error() + `"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issuer := &ticketIssuerStub{err: test.err}
			router, err := NewRouter(
				newTestHandlerWithIssuer(issuer),
				&userTokenResolverStub{userID: uuid.New()},
			)
			require.NoError(t, err)
			request := newIssueTicketHTTPRequest(uuid.New(), uuid.New(), uuid.New(), uuid.New())
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			require.Equal(t, test.statusCode, recorder.Code)
			require.JSONEq(t, test.response, recorder.Body.String())
			require.Equal(t, 1, issuer.calls)
		})
	}
}

func Test_PostInternalV1TicketIssue_UsecaseReturnsError_ReturnInternalError(t *testing.T) {
	// given
	issuer := &ticketIssuerStub{err: errors.New("database failed")}
	router, err := NewRouter(
		newTestHandlerWithIssuer(issuer),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := newIssueTicketHTTPRequest(uuid.New(), uuid.New(), uuid.New(), uuid.New())
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.JSONEq(t, `{"error":"internal_error","message":"internal server error"}`, recorder.Body.String())
	require.Equal(t, 1, issuer.calls)
}

func Test_PostInternalV1TicketIssue_InvalidRequest_ReturnBadRequest(t *testing.T) {
	// given
	validBody := `{
		"queue_entry_id":"` + uuid.NewString() + `",
		"user_id":"` + uuid.NewString() + `",
		"listing_id":"` + uuid.NewString() + `",
		"sku_id":"` + uuid.NewString() + `"
	}`
	tests := []struct {
		name           string
		body           string
		idempotencyKey string
	}{
		{name: "missing body", idempotencyKey: uuid.NewString()},
		{name: "malformed body", body: `{`, idempotencyKey: uuid.NewString()},
		{name: "extra field", body: strings.TrimSuffix(validBody, "}") + `,"extra":true}`, idempotencyKey: uuid.NewString()},
		{name: "missing idempotency key", body: validBody},
		{name: "invalid idempotency key", body: validBody, idempotencyKey: "invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issuer := &ticketIssuerStub{}
			router, err := NewRouter(
				newTestHandlerWithIssuer(issuer),
				&userTokenResolverStub{userID: uuid.New()},
			)
			require.NoError(t, err)
			request := httptest.NewRequest(
				http.MethodPost,
				"/internal/v1/ticket/issue",
				strings.NewReader(test.body),
			)
			request.Header.Set("Content-Type", "application/json")
			if test.idempotencyKey != "" {
				request.Header.Set("Idempotency-Key", test.idempotencyKey)
			}
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Zero(t, issuer.calls)
		})
	}
}

func Test_IssueTicket_NilBody_ReturnBadRequest(t *testing.T) {
	// given
	issuer := &ticketIssuerStub{}
	handler := newTestHandlerWithIssuer(issuer)

	// when
	response, err := handler.IssueTicket(context.Background(), server.IssueTicketRequestObject{})

	// then
	require.NoError(t, err)
	require.Equal(t, server.IssueTicket400JSONResponse{
		BadRequestJSONResponse: badRequestError("request body is required"),
	}, response)
	require.Zero(t, issuer.calls)
}

func newIssueTicketHTTPRequest(queueEntryID, userID, listingID, skuID uuid.UUID) *http.Request {
	request := httptest.NewRequest(
		http.MethodPost,
		"/internal/v1/ticket/issue",
		strings.NewReader(`{
			"queue_entry_id":"`+queueEntryID.String()+`",
			"user_id":"`+userID.String()+`",
			"listing_id":"`+listingID.String()+`",
			"sku_id":"`+skuID.String()+`"
		}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", uuid.NewString())

	return request
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
			AvailableActions:   []domain.TicketAvailableAction{},
		},
	}}
	resolver := &userTokenResolverStub{userID: userID}
	router, err := NewRouter(newTestHandler(lister), resolver)
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
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{
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
			"finish_reason": null,
			"available_actions": []
		}]
	}`, recorder.Body.String())
	require.Equal(t, userID, lister.receivedUserID)
	require.Equal(t, "abc-token", resolver.receivedToken)
	require.NotNil(t, lister.receivedFilter.Status)
	require.Equal(t, domain.TicketStatusRedeemed, *lister.receivedFilter.Status)
	require.NotNil(t, lister.receivedFilter.ListingID)
	require.Equal(t, listingID, *lister.receivedFilter.ListingID)
	require.NotNil(t, lister.receivedFilter.SKUID)
	require.Equal(t, skuID, *lister.receivedFilter.SKUID)
}

func Test_GetV1TicketList_WithoutTickets_ReturnEmptyArray(t *testing.T) {
	// given
	userID := uuid.New()
	lister := &ticketListerStub{tickets: []domain.Ticket{}}
	router, err := NewRouter(
		newTestHandler(lister),
		&userTokenResolverStub{userID: userID},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"ticket":[]}`, recorder.Body.String())
}

func Test_GetV1TicketList_WithoutToken_ReturnUnauthorized(t *testing.T) {
	// given
	lister := &ticketListerStub{}
	router, err := NewRouter(
		newTestHandler(lister),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.JSONEq(t, `{"error":"unauthorized","message":"bearer token is required"}`, recorder.Body.String())
	require.Zero(t, lister.calls)
}

func Test_GetV1TicketList_WithInvalidToken_ReturnUnauthorized(t *testing.T) {
	// given
	lister := &ticketListerStub{}
	resolver := &userTokenResolverStub{err: identityauth.ErrInvalidToken}
	router, err := NewRouter(newTestHandler(lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.JSONEq(t, `{"error":"unauthorized","message":"bearer token is invalid"}`, recorder.Body.String())
	require.Zero(t, lister.calls)
}

func Test_GetV1TicketList_AuthServiceReturnsError_ReturnInternalError(t *testing.T) {
	// given
	lister := &ticketListerStub{}
	resolver := &userTokenResolverStub{err: errors.New("adapter unavailable")}
	router, err := NewRouter(newTestHandler(lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.JSONEq(t, `{"error":"internal_error","message":"internal server error"}`, recorder.Body.String())
	require.Zero(t, lister.calls)
}

func Test_GetV1TicketList_WithCustomTokenResolver_ReturnTicketsForResolvedUser(t *testing.T) {
	// given
	userID := uuid.New()
	lister := &ticketListerStub{tickets: []domain.Ticket{}}
	resolver := &userTokenResolverStub{userID: userID}
	router, err := NewRouter(newTestHandler(lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer   opaque-token ")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "opaque-token", resolver.receivedToken)
	require.Equal(t, userID, lister.receivedUserID)
}

func Test_GetV1TicketList_WithAvitoAdapterResolver_ReturnTicketsForResolvedUser(t *testing.T) {
	// given
	userID := uuid.New()
	lister := &ticketListerStub{tickets: []domain.Ticket{}}
	client := &http.Client{Transport: authRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body struct {
			Token string `json:"token"`
		}
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		require.Equal(t, "abc-token", body.Token)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"user_id":"` + userID.String() + `"}`)),
		}, nil
	})}
	resolver, err := avitoadapter.NewUserTokenResolver("http://avito-adapter:8080", client)
	require.NoError(t, err)
	router, err := NewRouter(newTestHandler(lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, userID, lister.receivedUserID)
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
	router, err := NewRouter(newTestHandler(lister), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.JSONEq(t, `{"error":"unauthorized","message":"bearer token is invalid"}`, recorder.Body.String())
	require.Zero(t, lister.calls)
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
				newTestHandler(lister),
				&userTokenResolverStub{userID: uuid.New()},
			)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list?"+test.query, nil)
			request.Header.Set("Authorization", "Bearer "+uuid.NewString())
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Zero(t, lister.calls)
		})
	}
}

func Test_GetV1TicketList_UsecaseReturnsError_ReturnInternalError(t *testing.T) {
	// given
	lister := &ticketListerStub{err: errors.New("database failed")}
	router, err := NewRouter(
		newTestHandler(lister),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer "+uuid.NewString())
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.JSONEq(t, `{"error":"internal_error","message":"internal server error"}`, recorder.Body.String())
}

func Test_GetV1Ticket_ReturnTicket(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	issuedAt := time.Date(2026, time.August, 6, 10, 0, 0, 0, time.UTC)
	activationDeadline := issuedAt.Add(15 * time.Minute)
	getter := &ticketGetterStub{ticket: domain.Ticket{
		ID:                 ticketID,
		ListingID:          listingID,
		SKUID:              skuID,
		Status:             domain.TicketStatusIssued,
		IssuedAt:           issuedAt,
		ActivationDeadline: activationDeadline,
		AvailableActions: []domain.TicketAvailableAction{
			domain.TicketAvailableActionActivate,
			domain.TicketAvailableActionDecline,
		},
	}}
	router, err := NewRouter(
		newTestHandler(&ticketListerStub{}, getter),
		&userTokenResolverStub{userID: userID},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/"+ticketID.String(), nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{
		"id": "`+ticketID.String()+`",
		"listing_id": "`+listingID.String()+`",
		"sku_id": "`+skuID.String()+`",
		"status": "issued",
		"issued_at": "2026-08-06T10:00:00Z",
		"activation_deadline": "2026-08-06T10:15:00Z",
		"activated_at": null,
		"order_id": null,
		"checkout_url": null,
		"finished_at": null,
		"finish_reason": null,
		"available_actions": ["activate", "decline"]
	}`, recorder.Body.String())
	require.Equal(t, userID, getter.receivedUserID)
	require.Equal(t, ticketID, getter.receivedTicketID)
	require.Equal(t, 1, getter.calls)
}

func Test_GetV1Ticket_TicketNotFound_ReturnNotFound(t *testing.T) {
	// given
	ticketID := uuid.New()
	getter := &ticketGetterStub{err: usecase.ErrTicketNotFound}
	router, err := NewRouter(
		newTestHandler(&ticketListerStub{}, getter),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/"+ticketID.String(), nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.JSONEq(t, `{"error":"ticket_not_found","message":"ticket not found"}`, recorder.Body.String())
	require.Equal(t, ticketID, getter.receivedTicketID)
}

func Test_GetV1Ticket_UsecaseReturnsError_ReturnInternalError(t *testing.T) {
	// given
	getter := &ticketGetterStub{err: errors.New("database failed")}
	router, err := NewRouter(
		newTestHandler(&ticketListerStub{}, getter),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/"+uuid.NewString(), nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.JSONEq(t, `{"error":"internal_error","message":"internal server error"}`, recorder.Body.String())
}

func Test_GetV1Ticket_InvalidTicketID_ReturnBadRequest(t *testing.T) {
	// given
	getter := &ticketGetterStub{}
	router, err := NewRouter(
		newTestHandler(&ticketListerStub{}, getter),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/not-a-uuid", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Zero(t, getter.calls)
}

func Test_GetV1Ticket_WithoutToken_ReturnUnauthorized(t *testing.T) {
	// given
	getter := &ticketGetterStub{}
	router, err := NewRouter(
		newTestHandler(&ticketListerStub{}, getter),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/"+uuid.NewString(), nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.JSONEq(t, `{"error":"unauthorized","message":"bearer token is required"}`, recorder.Body.String())
	require.Zero(t, getter.calls)
}

func Test_PostV1TicketActivate_ReturnRedeemedTicket(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	idempotencyKey := uuid.New()
	orderID := uuid.New()
	activator := &ticketActivatorStub{result: usecase.ActivationResult{
		TicketID:    ticketID,
		Status:      domain.TicketStatusRedeemed,
		OrderID:     orderID,
		CheckoutURL: "/checkout?ticket=" + ticketID.String(),
	}}
	router, err := NewRouter(
		newTestHandlerWithActivator(activator),
		&userTokenResolverStub{userID: userID},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/v1/ticket/"+ticketID.String()+"/activate", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	request.Header.Set("Idempotency-Key", idempotencyKey.String())
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{
		"ticket_id":"`+ticketID.String()+`",
		"status":"redeemed",
		"order_id":"`+orderID.String()+`",
		"checkout_url":"/checkout?ticket=`+ticketID.String()+`"
	}`, recorder.Body.String())
	require.Equal(t, userID, activator.receivedUserID)
	require.Equal(t, ticketID, activator.receivedTicketID)
	require.Equal(t, idempotencyKey, activator.receivedIdempotencyKey)
	require.Equal(t, 1, activator.calls)
}

func Test_PostV1TicketActivate_UsecaseReturnsDomainError_ReturnMappedError(t *testing.T) {
	// given
	tests := []struct {
		name       string
		err        error
		statusCode int
		response   string
	}{
		{
			name:       "invalid activation",
			err:        usecase.ErrInvalidActivation,
			statusCode: http.StatusBadRequest,
			response:   `{"error":"bad_request","message":"invalid ticket activation"}`,
		},
		{
			name:       "ticket not found",
			err:        usecase.ErrTicketNotFound,
			statusCode: http.StatusNotFound,
			response:   `{"error":"ticket_not_found","message":"ticket not found"}`,
		},
		{
			name:       "ticket not activatable",
			err:        usecase.ErrTicketNotActivatable,
			statusCode: http.StatusConflict,
			response:   `{"error":"ticket_not_activatable","message":"ticket is not activatable"}`,
		},
		{
			name:       "idempotency conflict",
			err:        usecase.ErrIdempotencyConflict,
			statusCode: http.StatusConflict,
			response:   `{"error":"idempotency_conflict","message":"idempotency conflict"}`,
		},
		{
			name:       "activation in progress",
			err:        usecase.ErrActivationInProgress,
			statusCode: http.StatusConflict,
			response:   `{"error":"activation_in_progress","message":"ticket activation is in progress"}`,
		},
		{
			name:       "activation expired",
			err:        usecase.ErrTicketActivationExpired,
			statusCode: http.StatusGone,
			response:   `{"error":"ticket_activation_expired","message":"ticket activation expired"}`,
		},
		{
			name:       "order unavailable",
			err:        usecase.ErrOrderUnavailable,
			statusCode: http.StatusServiceUnavailable,
			response:   `{"error":"checkout_unavailable","message":"checkout is temporarily unavailable"}`,
		},
		{
			name:       "order rejected",
			err:        usecase.ErrOrderRejected,
			statusCode: http.StatusConflict,
			response:   `{"error":"checkout_rejected","message":"order creation rejected"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ticketID := uuid.New()
			activator := &ticketActivatorStub{err: test.err}
			router, err := NewRouter(
				newTestHandlerWithActivator(activator),
				&userTokenResolverStub{userID: uuid.New()},
			)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/v1/ticket/"+ticketID.String()+"/activate", nil)
			request.Header.Set("Authorization", "Bearer abc-token")
			request.Header.Set("Idempotency-Key", uuid.NewString())
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			require.Equal(t, test.statusCode, recorder.Code)
			require.JSONEq(t, test.response, recorder.Body.String())
			require.Equal(t, 1, activator.calls)
			require.Equal(t, ticketID, activator.receivedTicketID)
		})
	}
}

func Test_PostV1TicketActivate_UsecaseReturnsError_ReturnInternalError(t *testing.T) {
	// given
	activator := &ticketActivatorStub{err: errors.New("database failed")}
	router, err := NewRouter(
		newTestHandlerWithActivator(activator),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/v1/ticket/"+uuid.NewString()+"/activate", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	request.Header.Set("Idempotency-Key", uuid.NewString())
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.JSONEq(t, `{"error":"internal_error","message":"internal server error"}`, recorder.Body.String())
}

func Test_PostV1TicketActivate_InvalidRequest_ReturnBadRequest(t *testing.T) {
	// given
	tests := []struct {
		name           string
		ticketID       string
		idempotencyKey string
	}{
		{name: "invalid ticket id", ticketID: "invalid", idempotencyKey: uuid.NewString()},
		{name: "missing idempotency key", ticketID: uuid.NewString()},
		{name: "invalid idempotency key", ticketID: uuid.NewString(), idempotencyKey: "invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			activator := &ticketActivatorStub{}
			router, err := NewRouter(
				newTestHandlerWithActivator(activator),
				&userTokenResolverStub{userID: uuid.New()},
			)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/v1/ticket/"+test.ticketID+"/activate", nil)
			request.Header.Set("Authorization", "Bearer abc-token")
			if test.idempotencyKey != "" {
				request.Header.Set("Idempotency-Key", test.idempotencyKey)
			}
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Zero(t, activator.calls)
		})
	}
}

func Test_PostV1TicketActivate_WithoutToken_ReturnUnauthorized(t *testing.T) {
	// given
	activator := &ticketActivatorStub{}
	router, err := NewRouter(
		newTestHandlerWithActivator(activator),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/v1/ticket/"+uuid.NewString()+"/activate", nil)
	request.Header.Set("Idempotency-Key", uuid.NewString())
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.JSONEq(t, `{"error":"unauthorized","message":"bearer token is required"}`, recorder.Body.String())
	require.Zero(t, activator.calls)
}

func Test_PostV1TicketDecline_ReturnClosedTicket(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	idempotencyKey := uuid.New()
	decliner := &ticketDeclinerStub{result: usecase.DeclineTicketResult{
		TicketID: ticketID,
		Status:   domain.TicketStatusClosed,
	}}
	resolver := &userTokenResolverStub{userID: userID}
	router, err := NewRouter(newTestHandlerWithDecliner(decliner), resolver)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/v1/ticket/"+ticketID.String()+"/decline", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	request.Header.Set("Idempotency-Key", idempotencyKey.String())
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{
		"ticket_id":"`+ticketID.String()+`",
		"status":"closed"
	}`, recorder.Body.String())
	require.Equal(t, userID, decliner.receivedUserID)
	require.Equal(t, ticketID, decliner.receivedTicketID)
	require.Equal(t, idempotencyKey, decliner.receivedIdempotencyKey)
	require.Equal(t, "abc-token", resolver.receivedToken)
	require.Equal(t, 1, decliner.calls)
}

func Test_PostV1TicketDecline_UsecaseReturnsDomainError_ReturnMappedError(t *testing.T) {
	// given
	tests := []struct {
		name       string
		err        error
		statusCode int
		response   string
	}{
		{
			name:       "invalid decline",
			err:        usecase.ErrInvalidDecline,
			statusCode: http.StatusBadRequest,
			response:   `{"error":"bad_request","message":"invalid ticket decline"}`,
		},
		{
			name:       "ticket not found",
			err:        usecase.ErrTicketNotFound,
			statusCode: http.StatusNotFound,
			response:   `{"error":"ticket_not_found","message":"ticket not found"}`,
		},
		{
			name:       "ticket not declinable",
			err:        usecase.ErrTicketNotDeclinable,
			statusCode: http.StatusConflict,
			response:   `{"error":"ticket_not_declinable","message":"ticket is not declinable"}`,
		},
		{
			name:       "idempotency conflict",
			err:        usecase.ErrIdempotencyConflict,
			statusCode: http.StatusConflict,
			response:   `{"error":"idempotency_conflict","message":"idempotency conflict"}`,
		},
		{
			name:       "activation in progress",
			err:        usecase.ErrActivationInProgress,
			statusCode: http.StatusConflict,
			response:   `{"error":"activation_in_progress","message":"ticket activation is in progress"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ticketID := uuid.New()
			decliner := &ticketDeclinerStub{err: test.err}
			router, err := NewRouter(
				newTestHandlerWithDecliner(decliner),
				&userTokenResolverStub{userID: uuid.New()},
			)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/v1/ticket/"+ticketID.String()+"/decline", nil)
			request.Header.Set("Authorization", "Bearer abc-token")
			request.Header.Set("Idempotency-Key", uuid.NewString())
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			require.Equal(t, test.statusCode, recorder.Code)
			require.JSONEq(t, test.response, recorder.Body.String())
			require.Equal(t, 1, decliner.calls)
			require.Equal(t, ticketID, decliner.receivedTicketID)
		})
	}
}

func Test_PostV1TicketDecline_UsecaseReturnsError_ReturnInternalError(t *testing.T) {
	// given
	decliner := &ticketDeclinerStub{err: errors.New("database failed")}
	router, err := NewRouter(
		newTestHandlerWithDecliner(decliner),
		&userTokenResolverStub{userID: uuid.New()},
	)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/v1/ticket/"+uuid.NewString()+"/decline", nil)
	request.Header.Set("Authorization", "Bearer abc-token")
	request.Header.Set("Idempotency-Key", uuid.NewString())
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.JSONEq(t, `{"error":"internal_error","message":"internal server error"}`, recorder.Body.String())
	require.Equal(t, 1, decliner.calls)
}

func Test_PostV1TicketDecline_InvalidRequest_ReturnBadRequest(t *testing.T) {
	// given
	tests := []struct {
		name            string
		ticketID        string
		idempotencyKeys []string
	}{
		{name: "invalid ticket id", ticketID: "invalid", idempotencyKeys: []string{uuid.NewString()}},
		{name: "missing idempotency key", ticketID: uuid.NewString()},
		{name: "invalid idempotency key", ticketID: uuid.NewString(), idempotencyKeys: []string{"invalid"}},
		{
			name:            "multiple idempotency keys",
			ticketID:        uuid.NewString(),
			idempotencyKeys: []string{uuid.NewString(), uuid.NewString()},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decliner := &ticketDeclinerStub{}
			router, err := NewRouter(
				newTestHandlerWithDecliner(decliner),
				&userTokenResolverStub{userID: uuid.New()},
			)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/v1/ticket/"+test.ticketID+"/decline", nil)
			request.Header.Set("Authorization", "Bearer abc-token")
			for _, idempotencyKey := range test.idempotencyKeys {
				request.Header.Add("Idempotency-Key", idempotencyKey)
			}
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Zero(t, decliner.calls)
		})
	}
}

func Test_PostV1TicketDecline_AuthenticationFails_ReturnError(t *testing.T) {
	// given
	tests := []struct {
		name       string
		token      string
		resolver   *userTokenResolverStub
		statusCode int
		response   string
	}{
		{
			name:       "missing token",
			resolver:   &userTokenResolverStub{userID: uuid.New()},
			statusCode: http.StatusUnauthorized,
			response:   `{"error":"unauthorized","message":"bearer token is required"}`,
		},
		{
			name:       "invalid token",
			token:      "invalid-token",
			resolver:   &userTokenResolverStub{err: identityauth.ErrInvalidToken},
			statusCode: http.StatusUnauthorized,
			response:   `{"error":"unauthorized","message":"bearer token is invalid"}`,
		},
		{
			name:       "identity service unavailable",
			token:      "abc-token",
			resolver:   &userTokenResolverStub{err: errors.New("adapter unavailable")},
			statusCode: http.StatusInternalServerError,
			response:   `{"error":"internal_error","message":"internal server error"}`,
		},
		{
			name:       "empty user id",
			token:      "abc-token",
			resolver:   &userTokenResolverStub{},
			statusCode: http.StatusInternalServerError,
			response:   `{"error":"internal_error","message":"internal server error"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decliner := &ticketDeclinerStub{}
			router, err := NewRouter(newTestHandlerWithDecliner(decliner), test.resolver)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/v1/ticket/"+uuid.NewString()+"/decline", nil)
			request.Header.Set("Idempotency-Key", uuid.NewString())
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			require.Equal(t, test.statusCode, recorder.Code)
			require.JSONEq(t, test.response, recorder.Body.String())
			require.Zero(t, decliner.calls)
		})
	}
}
