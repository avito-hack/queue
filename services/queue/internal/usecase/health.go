package usecase

import "context"

type Health struct{}

func NewHealth() *Health {
	return &Health{}
}

func (h *Health) Check(context.Context) error {
	return nil
}
