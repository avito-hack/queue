package usecase

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Service) CreateUser(name string) (User, error) {
	if strings.TrimSpace(name) == "" {
		return User{}, ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user := User{ID: uuid.NewString(), Name: name, CreatedAt: time.Now().UTC()}
	s.users[user.ID] = user
	return user, nil
}

func (s *Service) GetUser(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}
