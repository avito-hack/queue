package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

func Test_GetHealthz_ReturnOK(t *testing.T) {
	// given
	router, err := NewRouter(NewHandler(usecase.NewHealth()))
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, recorder.Body.String())
}

func Test_GetV1TicketList_ReturnNotImplemented(t *testing.T) {
	// given
	router, err := NewRouter(NewHandler(usecase.NewHealth()))
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	request.Header.Set("Authorization", "Bearer stub-token")
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"error":"not_implemented","message":"tickets service is not implemented"}`, recorder.Body.String())
}

func Test_GetV1TicketList_WithoutToken_ReturnUnauthorized(t *testing.T) {
	// given
	router, err := NewRouter(NewHandler(usecase.NewHealth()))
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodGet, "/v1/ticket/list", nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.JSONEq(t, `{"error":"unauthorized","message":"bearer token is required"}`, recorder.Body.String())
}
