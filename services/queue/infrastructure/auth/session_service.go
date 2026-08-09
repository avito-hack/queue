package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type SessionService struct {
	introspectionURL string
	httpClient       *http.Client
	cacheTTL         time.Duration

	mu    sync.RWMutex
	cache map[string]*sessionCache
}

func NewSessionService(introspectionURL string, timeout, cacheTTL time.Duration) *SessionService {
	return &SessionService{
		introspectionURL: introspectionURL,
		httpClient:       &http.Client{Timeout: timeout},
		cacheTTL:         cacheTTL,
		cache:            make(map[string]*sessionCache),
	}
}

func (s *SessionService) Validate(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", errors.New("empty token")
	}

	now := time.Now().Unix()

	s.mu.RLock()

	cached, ok := s.cache[token]
	if ok && now < cached.expiresAt {
		s.mu.RUnlock()

		return cached.userID, nil
	}

	s.mu.RUnlock()

	reqBody := introspectRequest{
		Token: token,
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal introspection request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.introspectionURL,
		bytes.NewReader(raw),
	)
	if err != nil {
		return "", fmt.Errorf("create introspection request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("introspection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("introspection failed: status %d", resp.StatusCode)
	}

	var parsed introspectResponse

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode introspection response: %w", err)
	}

	if !parsed.Active {
		return "", errors.New("inactive token")
	}

	expireAt := time.Now().Add(s.cacheTTL).Unix()

	s.mu.Lock()

	s.cache[token] = &sessionCache{
		userID:    parsed.UserID,
		expiresAt: expireAt,
	}

	s.mu.Unlock()

	return parsed.UserID, nil
}