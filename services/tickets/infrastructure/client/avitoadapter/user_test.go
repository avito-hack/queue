package avitoadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	generated "github.com/avito-hack/queue/services/tickets/gen/clients/avitoadapter"
	identityauth "github.com/avito-hack/queue/services/tickets/internal/auth"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestUserTokenResolver_ResolveUserID_ReturnUserID(t *testing.T) {
	// given
	expectedUserID := uuid.New()
	responseBody, err := json.Marshal(generated.ValidateUserTokenResult{UserId: expectedUserID})
	require.NoError(t, err)
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, request.Method)
		require.Equal(t, "http://avito-adapter:8080/v1/users/validate", request.URL.String())
		require.Equal(t, "application/json", request.Header.Get("Content-Type"))
		require.Empty(t, request.Header.Get("Authorization"))

		var body generated.ValidateUserTokenRequest
		require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
		require.NotNil(t, body.Token)
		require.Equal(t, "abc-token", *body.Token)

		return newHTTPResponse(http.StatusOK, string(responseBody)), nil
	})}
	resolver, err := NewUserTokenResolver("http://avito-adapter:8080", client)
	require.NoError(t, err)

	// when
	userID, err := resolver.ResolveUserID(context.Background(), "abc-token")

	// then
	require.NoError(t, err)
	require.Equal(t, expectedUserID, userID)
}

func TestUserTokenResolver_ResolveUserID_AdapterRejectsToken_ReturnInvalidToken(t *testing.T) {
	// given
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return newHTTPResponse(http.StatusUnauthorized, `{"message":"invalid token"}`), nil
	})}
	resolver, err := NewUserTokenResolver("http://avito-adapter:8080", client)
	require.NoError(t, err)

	// when
	_, err = resolver.ResolveUserID(context.Background(), "invalid-token")

	// then
	require.ErrorIs(t, err, identityauth.ErrInvalidToken)
}

func TestUserTokenResolver_ResolveUserID_AdapterReturnsUnexpectedStatus_ReturnError(t *testing.T) {
	// given
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "bad request", statusCode: http.StatusBadRequest},
		{name: "forbidden", statusCode: http.StatusForbidden},
		{name: "not found", statusCode: http.StatusNotFound},
		{name: "unprocessable entity", statusCode: http.StatusUnprocessableEntity},
		{name: "internal error", statusCode: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return newHTTPResponse(test.statusCode, `{"message":"unexpected status"}`), nil
			})}
			resolver, err := NewUserTokenResolver("http://avito-adapter:8080", client)
			require.NoError(t, err)

			// when
			_, err = resolver.ResolveUserID(context.Background(), "abc-token")

			// then
			require.EqualError(t, err, fmt.Sprintf("validate user: unexpected status %d", test.statusCode))
			require.NotErrorIs(t, err, identityauth.ErrInvalidToken)
		})
	}
}

func TestUserTokenResolver_ResolveUserID_AdapterReturnsInvalidBody_ReturnError(t *testing.T) {
	// given
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid JSON", body: "{"},
		{name: "empty user id", body: `{"user_id":"00000000-0000-0000-0000-000000000000"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return newHTTPResponse(http.StatusOK, test.body), nil
			})}
			resolver, err := NewUserTokenResolver("http://avito-adapter:8080", client)
			require.NoError(t, err)

			// when
			_, err = resolver.ResolveUserID(context.Background(), "abc-token")

			// then
			require.Error(t, err)
			require.NotErrorIs(t, err, identityauth.ErrInvalidToken)
		})
	}
}

func TestUserTokenResolver_ResolveUserID_RequestReturnsError_ReturnError(t *testing.T) {
	// given
	requestError := errors.New("request failed")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, requestError
	})}
	resolver, err := NewUserTokenResolver("http://avito-adapter:8080", client)
	require.NoError(t, err)

	// when
	_, err = resolver.ResolveUserID(context.Background(), "abc-token")

	// then
	require.Error(t, err)
	require.ErrorIs(t, err, requestError)
}

func TestUserTokenResolver_ResolveUserID_AdapterRedirects_ReturnErrorWithoutFollowingRedirect(t *testing.T) {
	// given
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		response := newHTTPResponse(http.StatusTemporaryRedirect, "")
		response.Header.Set("Location", "http://another-service/token")

		return response, nil
	})}
	resolver, err := NewUserTokenResolver("http://avito-adapter:8080", client)
	require.NoError(t, err)

	// when
	_, err = resolver.ResolveUserID(context.Background(), "abc-token")

	// then
	require.EqualError(t, err, "validate user: unexpected status 307")
	require.Equal(t, 1, calls)
}

func TestNewUserTokenResolver_InvalidConfiguration_ReturnError(t *testing.T) {
	// given
	tests := []struct {
		name    string
		baseURL string
		client  *http.Client
	}{
		{name: "nil client", baseURL: "http://avito-adapter:8080"},
		{name: "missing scheme", baseURL: "avito-adapter:8080", client: http.DefaultClient},
		{name: "missing host", baseURL: "http://", client: http.DefaultClient},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// when
			_, err := NewUserTokenResolver(test.baseURL, test.client)

			// then
			require.Error(t, err)
		})
	}
}

func newHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
