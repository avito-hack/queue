package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/avito-adapter/internal/usecase"
)

func Test_GetHealth_ReturnOK(t *testing.T) {
	// given
	router, err := NewRouter(NewHandler(usecase.NewHealth(), usecase.NewService()))
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"status":"ok"}`, recorder.Body.String())
}

func Test_ValidateUserToken_ReturnUserID(t *testing.T) {
	// given
	service := usecase.NewService()
	user, err := service.CreateUser(context.Background(), "buyer", "token-1")
	require.NoError(t, err)
	router, err := NewRouter(NewHandler(usecase.NewHealth(), service))
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/v1/users/validate", bytes.NewBufferString(`{"token":"Bearer token-1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"user_id":"`+user.ID+`"}`, recorder.Body.String())
}

func Test_ValidateUserToken_ReturnUnauthorizedWhenTokenMissing(t *testing.T) {
	// given
	router, err := NewRouter(NewHandler(usecase.NewHealth(), usecase.NewService()))
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/v1/users/validate", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func Test_CreateOrder_ReturnCreated(t *testing.T) {
	// given
	service := usecase.NewService()
	seller, err := service.CreateUser(context.Background(), "seller", "seller-token")
	require.NoError(t, err)
	buyer, err := service.CreateUser(context.Background(), "buyer", "buyer-token")
	require.NoError(t, err)
	listing, err := service.CreateListing(context.Background(), seller.ID, "item", 100, 1, true)
	require.NoError(t, err)
	ticketID := uuid.NewString()
	skuID := uuid.NewString()
	body, err := json.Marshal(map[string]string{"ticketId": ticketID, "listingId": listing.ID, "skuId": skuID, "userId": buyer.ID})
	require.NoError(t, err)
	router, err := NewRouter(NewHandler(usecase.NewHealth(), service))
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/v1/orders/create", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", uuid.NewString())
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.JSONEq(t, `{"ticketId":"`+ticketID+`","listingId":"`+listing.ID+`","skuId":"`+skuID+`","userId":"`+buyer.ID+`"}`, selectJSONFields(t, recorder.Body.Bytes(), "ticketId", "listingId", "skuId", "userId"))
}

func selectJSONFields(t *testing.T, body []byte, fields ...string) string {
	t.Helper()
	var source map[string]any
	require.NoError(t, json.Unmarshal(body, &source))
	selected := make(map[string]any, len(fields))
	for _, field := range fields {
		selected[field] = source[field]
	}
	result, err := json.Marshal(selected)
	require.NoError(t, err)
	return string(result)
}
