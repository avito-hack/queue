package domain

import "errors"

var (
	ErrQueueNotFound  = errors.New("item queue not found")
	ErrAlreadyExists  = errors.New("item queue already exists")
	ErrMemberNotFound = errors.New("item queue member not found")
	ErrMemberAlreadyActive = errors.New("queue member already active")
)