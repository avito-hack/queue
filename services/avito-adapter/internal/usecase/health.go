package usecase

import (
	"context"
	"fmt"
)

type HealthDependency interface {
	Ping(context.Context) error
}

type Health struct {
	dependency HealthDependency
}

func NewHealth(dependencies ...HealthDependency) *Health {
	var dependency HealthDependency
	if len(dependencies) > 0 {
		dependency = dependencies[0]
	}
	return &Health{dependency: dependency}
}

func (h *Health) Check(ctx context.Context) error {
	if h.dependency == nil {
		return nil
	}
	if err := h.dependency.Ping(ctx); err != nil {
		return fmt.Errorf("ping dependency: %w", err)
	}
	return nil
}
