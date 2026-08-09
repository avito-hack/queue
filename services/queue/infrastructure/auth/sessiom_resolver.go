package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type SessionResolver struct {
	session *SessionService
}

func NewSessionResolver(s *SessionService) *SessionResolver {
	return &SessionResolver{session: s}
}

func (r *SessionResolver) ResolveUserID(ctx context.Context, token string) (uuid.UUID, error) {
	userIDStr, err := r.session.Validate(ctx, token)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	id, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: parse user id: %v", ErrInvalidToken, err)
	}
	return id, nil
}
