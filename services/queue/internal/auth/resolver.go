package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type UserTokenResolver interface {
	ResolveUserID(ctx context.Context, token string) (uuid.UUID, error)
}
