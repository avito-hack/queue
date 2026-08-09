package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/queue/internal/domain"
	"github.com/avito-hack/queue/services/queue/internal/usecase"
)

type healthCheckerStub struct{}

func (h *healthCheckerStub) Check(context.Context) error {
	return nil
}

type queueServiceStub struct {
	enqueueErr error
}

type userTokenResolverStub struct {
	userID uuid.UUID
}

func (s *userTokenResolverStub) ResolveUserID(context.Context, string) (uuid.UUID, error) {
	return s.userID, nil
}

func (s *queueServiceStub) Enqueue(context.Context, uuid.UUID, uuid.UUID) error {
	return s.enqueueErr
}

func (s *queueServiceStub) Dequeue(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (s *queueServiceStub) ClearItemQueue(context.Context, uuid.UUID) error {
	return nil
}

func (s *queueServiceStub) GetUserPosition(context.Context, uuid.UUID, uuid.UUID) (uint, error) {
	return 0, nil
}

func (s *queueServiceStub) GetItemQueueState(context.Context, uuid.UUID) (domain.ItemQueueStateInfo, error) {
	return domain.ItemQueueStateInfo{
		State:        domain.QueueTicketsAvailable,
		WaitingCount: 0,
	}, nil
}

func (s *queueServiceStub) GetUserQueues(context.Context, uuid.UUID) ([]*domain.UserQueueInfo, error) {
	return nil, nil
}

func Test_NewRouter_Enqueue_ReturnUnauthorizedWithoutToken(t *testing.T) {
	// given
	handler := NewHandler(&healthCheckerStub{}, &queueServiceStub{}, slog.Default())
	router, err := NewRouter(handler, &userTokenResolverStub{}, slog.Default())
	require.NoError(t, err)

	itemID := uuid.New()
	request := httptest.NewRequest(http.MethodPost, "/v1/queue/"+itemID.String()+"/enqueue", nil)
	recorder := httptest.NewRecorder()

	// when
	router.ServeHTTP(recorder, request)

	// then
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assertBodyCode(t, recorder, "unauthorized")
}

func Test_NewRouter_Enqueue_BlackBoxStatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		enqueueErr error
		expected   int
		errCode    string
	}{
		{
			name:       "given queue not found when enqueue then return 404",
			enqueueErr: usecase.ErrQueueNotFound,
			expected:   http.StatusNotFound,
			errCode:    "not_found",
		},
		{
			name:       "given user already in queue when enqueue then return 409",
			enqueueErr: usecase.ErrUserAlreadyInQueue,
			expected:   http.StatusConflict,
			errCode:    "conflict",
		},
		{
			name:       "given user has active ticket when enqueue then return 409",
			enqueueErr: usecase.ErrUserHasActiveTicket,
			expected:   http.StatusConflict,
			errCode:    "conflict",
		},
		{
			name:       "given queue unavailable when enqueue then return 422",
			enqueueErr: usecase.ErrQueueUnavailable,
			expected:   http.StatusUnprocessableEntity,
			errCode:    "queue_unavailable",
		},
		{
			name:       "given no error when enqueue then return 201",
			enqueueErr: nil,
			expected:   http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			userID := uuid.New()
			itemID := uuid.New()
			service := &queueServiceStub{enqueueErr: tt.enqueueErr}

			handler := NewHandler(&healthCheckerStub{}, service, slog.Default())
			router, err := NewRouter(handler, &userTokenResolverStub{userID: userID}, slog.Default())
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPost, "/v1/queue/"+itemID.String()+"/enqueue", nil)
			request.Header.Set("Authorization", "Bearer test-token")
			recorder := httptest.NewRecorder()

			// when
			router.ServeHTTP(recorder, request)

			// then
			assert.Equal(t, tt.expected, recorder.Code)
			if tt.errCode != "" {
				assertBodyCode(t, recorder, tt.errCode)
			}
		})
	}
}

func assertBodyCode(t *testing.T, recorder *httptest.ResponseRecorder, expected string) {
	t.Helper()

	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, expected, body["code"])
}
