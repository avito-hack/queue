package avitoadapter

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	generated "github.com/avito-hack/queue/services/tickets/gen/clients/avitoadapter"
	identityauth "github.com/avito-hack/queue/services/tickets/internal/auth"
)

type UserTokenResolver struct {
	client generated.ClientWithResponsesInterface
}

func NewUserTokenResolver(baseURL string, client *http.Client) (*UserTokenResolver, error) {
	generatedClient, err := newGeneratedClient(baseURL, client)
	if err != nil {
		return nil, err
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
