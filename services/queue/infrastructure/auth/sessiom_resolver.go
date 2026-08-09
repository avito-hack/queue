package auth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type SessionResolver struct {
	session *SessionService
	logger  *slog.Logger
}

func NewSessionResolver(s *SessionService, logger *slog.Logger) *SessionResolver {
	return &SessionResolver{
		session: s,
		logger:  logger,
	}
}

func (r *SessionResolver) ResolveUserID(ctx context.Context, token string) (uuid.UUID, error) {
	userIDStr, err := r.session.Validate(ctx, token)

	if err != nil {
		r.logger.Warn(
			"session validation failed",
			"error",
			err,
		)

		return uuid.Nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	id, err := uuid.Parse(userIDStr)

	if err != nil {
		r.logger.Error(
			"failed to parse user id from session",
			"error",
			err,
			"user_id",
			userIDStr,
		)

		return uuid.Nil, fmt.Errorf("%w: parse user id: %v", ErrInvalidToken, err)
	}

	r.logger.Info(
		"user identity resolved",
		"user_id",
		id,
	)

	return id, nil
}