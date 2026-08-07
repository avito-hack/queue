package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid user token")

type UserTokenResolver interface {
	ResolveUserID(context.Context, string) (uuid.UUID, error)
}
