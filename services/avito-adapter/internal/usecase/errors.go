package usecase

import "errors"

var (
	ErrNotFound          = errors.New("resource not found")
	ErrInvalid           = errors.New("invalid request")
	ErrUnavailable       = errors.New("listing is unavailable")
	ErrInsufficientStock = errors.New("insufficient available quantity")
	ErrConflict          = errors.New("operation conflicts with current state")
)
