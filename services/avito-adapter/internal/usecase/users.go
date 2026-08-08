package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Service) CreateUser(name, token string) (User, error) {
	token = bearerToken(token)
	if strings.TrimSpace(name) == "" || token == "" {
		return User{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, user := range s.users {
		if user.Token == token {
			return User{}, ErrConflict
		}
	}
	user := User{ID: uuid.NewString(), Name: name, Token: token, CreatedAt: time.Now().UTC()}
	s.users[user.ID] = user
	return user, nil
}

func (s *Service) ValidateUserToken(ctx context.Context, token string) (User, error) {
	token = bearerToken(token)
	if token == "" {
		return User{}, ErrUnauthorized
	}
	if s.reader != nil {
		return s.reader.GetUserByToken(ctx, token)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, user := range s.users {
		if user.Token == token {
			return user, nil
		}
	}
	return User{}, ErrUnauthorized
}

func bearerToken(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 7 && strings.EqualFold(value[:7], "Bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return value
}

func (s *Service) GetUser(ctx context.Context, id string) (User, error) {
	if s.reader != nil {
		return s.reader.GetUser(ctx, id)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}
