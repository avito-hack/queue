package avitoadapter

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	generated "github.com/avito-hack/queue/services/tickets/gen/clients/avitoadapter"
	identityauth "github.com/avito-hack/queue/services/tickets/internal/auth"
)

type UserTokenResolver struct {
	client generated.ClientWithResponsesInterface
}

func NewUserTokenResolver(baseURL string, client *http.Client) (*UserTokenResolver, error) {
	if client == nil {
		return nil, fmt.Errorf("http client is nil")
	}

	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse avito adapter URL: %w", err)
	}
	if parsedBaseURL.Scheme != "http" && parsedBaseURL.Scheme != "https" {
		return nil, fmt.Errorf("avito adapter URL must use HTTP or HTTPS")
	}
	if parsedBaseURL.Host == "" {
		return nil, fmt.Errorf("avito adapter URL host is required")
	}
	if parsedBaseURL.RawQuery != "" || parsedBaseURL.Fragment != "" {
		return nil, fmt.Errorf("avito adapter URL must not contain query or fragment")
	}

	clientCopy := *client
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	generatedClient, err := generated.NewClientWithResponses(
		parsedBaseURL.String(),
		generated.WithHTTPClient(&clientCopy),
	)
	if err != nil {
		return nil, fmt.Errorf("create generated avito adapter client: %w", err)
	}

	return &UserTokenResolver{client: generatedClient}, nil
}

func (r *UserTokenResolver) ResolveUserID(ctx context.Context, token string) (uuid.UUID, error) {
	response, err := r.client.ValidateUserWithResponse(ctx, generated.ValidateUserJSONRequestBody{Token: token})
	if err != nil {
		return uuid.Nil, fmt.Errorf("validate user request: %w", err)
	}

	switch response.StatusCode() {
	case http.StatusOK:
		if response.JSON200 == nil || response.JSON200.UserId == uuid.Nil {
			return uuid.Nil, fmt.Errorf("decode validate user response: empty user id")
		}

		return response.JSON200.UserId, nil
	case http.StatusUnauthorized:
		return uuid.Nil, identityauth.ErrInvalidToken
	default:
		return uuid.Nil, fmt.Errorf("validate user: unexpected status %d", response.StatusCode())
	}
}
