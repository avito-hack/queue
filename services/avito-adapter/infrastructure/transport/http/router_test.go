package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
