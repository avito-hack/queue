package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type SessionService struct {
	introspectionURL string
	httpClient       *http.Client
	cacheTTL         time.Duration
	logger           *slog.Logger

	mu    sync.RWMutex
	cache map[string]*sessionCache
}

func NewSessionService(introspectionURL string, timeout, cacheTTL time.Duration, logger *slog.Logger) *SessionService {
	return &SessionService{
		introspectionURL: introspectionURL,
		httpClient:       &http.Client{Timeout: timeout},
		cacheTTL:         cacheTTL,
		logger:           logger,
		cache:            make(map[string]*sessionCache),
	}
}

func (s *SessionService) Validate(ctx context.Context, token string) (string, error) {
	if token == "" {
		err := errors.New("empty token")

		s.logger.Warn(
			"token validation failed: empty token",
			"error",
			err,
		)

		return "", err
	}

	now := time.Now().Unix()

	s.mu.RLock()

	cached, ok := s.cache[token]

	if ok && now < cached.expiresAt {
		s.mu.RUnlock()

		s.logger.Info(
			"token validation cache hit",
			"user_id",
			cached.userID,
		)

		return cached.userID, nil
	}

	s.mu.RUnlock()

	s.logger.Info(
		"token validation started",
		"introspection_url",
		s.introspectionURL,
	)

	reqBody := introspectRequest{
		Token: token,
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		s.logger.Error(
			"failed to marshal introspection request",
			"error",
			err,
		)

		return "", fmt.Errorf("marshal introspection request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.introspectionURL,
		bytes.NewReader(raw),
	)

	if err != nil {
		s.logger.Error(
			"failed to create introspection request",
			"error",
			err,
		)

		return "", fmt.Errorf("create introspection request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)

	if err != nil {
		s.logger.Error(
			"introspection request failed",
			"error",
			err,
			"url",
			s.introspectionURL,
		)

		return "", fmt.Errorf("introspection request failed: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected introspection status: %d", resp.StatusCode)

		s.logger.Warn(
			"introspection failed",
			"status",
			resp.StatusCode,
		)

		return "", err
	}

	var parsed introspectResponse

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		s.logger.Error(
			"failed to decode introspection response",
			"error",
			err,
		)

		return "", fmt.Errorf("decode introspection response: %w", err)
	}
	if parsed.UserID == "" {
		err := errors.New("empty user id")

		s.logger.Warn(
			"token validation returned empty user id",
		)

		return "", err
	}

	expireAt := time.Now().Add(s.cacheTTL).Unix()

	s.mu.Lock()

	s.cache[token] = &sessionCache{
		userID:    parsed.UserID,
		expiresAt: expireAt,
	}

	s.mu.Unlock()

	s.logger.Info(
		"token validated successfully",
		"user_id",
		parsed.UserID,
		"cache_ttl",
		s.cacheTTL,
	)

	return parsed.UserID, nil
}
