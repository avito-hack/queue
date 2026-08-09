package auth

import "errors"

type contextKey string

const (
	UserIDKey              contextKey = "user_id"
	AuthorizationHeaderKey contextKey = "authorization_header"
)

var ErrInvalidToken = errors.New("invalid token")

type sessionCache struct {
	userID    string
	expiresAt int64
}

type introspectRequest struct {
	Token string `json:"token"`
}

type introspectResponse struct {
	Active  bool   `json:"active"`
	UserID  string `json:"user_id"`
	Expires string `json:"expires_at"`
}
