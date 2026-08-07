package usecase

import "errors"

var (
	ErrInvalidRequest       = errors.New("invalid request")
	ErrQueueNotFound        = errors.New("queue not found")
	ErrQueueAlreadyExists   = errors.New("queue already exists")
	ErrQueueUnavailable     = errors.New("queue unavailable")
	ErrUserAlreadyInQueue   = errors.New("user already in queue")
	ErrUserHasActiveTicket  = errors.New("user already has an active ticket")
	ErrUserNotInQueue       = errors.New("user not found in queue")
	ErrUserCannotLeaveQueue = errors.New("user cannot leave queue")
	ErrPositionNotFound     = errors.New("user position not found")
	ErrQueueEmpty           = errors.New("queue is empty")
	ErrInvalidQueueState    = errors.New("invalid queue state")
	ErrInvalidMemberStatus  = errors.New("invalid member status")
	ErrInternal             = errors.New("internal error")
)
