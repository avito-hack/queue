package domain

import (
	"time"

	"github.com/google/uuid"
)

type TicketStatus string

const (
	TicketStatusIssued   TicketStatus = "issued"
	TicketStatusRedeemed TicketStatus = "redeemed"
	TicketStatusClosed   TicketStatus = "closed"
)

func (s TicketStatus) Valid() bool {
	switch s {
	case TicketStatusIssued, TicketStatusRedeemed, TicketStatusClosed:
		return true
	default:
		return false
	}
}

type TicketCloseReason string

const (
	TicketCloseReasonActivationTimeout TicketCloseReason = "activation_timeout"
	TicketCloseReasonUserDeclined      TicketCloseReason = "user_declined"
	TicketCloseReasonListingClosed     TicketCloseReason = "listing_closed"
	TicketCloseReasonSKUClosed         TicketCloseReason = "sku_closed"
	TicketCloseReasonSystemCancelled   TicketCloseReason = "system_cancelled"
)

func (r TicketCloseReason) Valid() bool {
	switch r {
	case TicketCloseReasonActivationTimeout,
		TicketCloseReasonUserDeclined,
		TicketCloseReasonListingClosed,
		TicketCloseReasonSKUClosed,
		TicketCloseReasonSystemCancelled:
		return true
	default:
		return false
	}
}

type TicketAvailableAction string

const (
	TicketAvailableActionActivate TicketAvailableAction = "activate"
	TicketAvailableActionDecline  TicketAvailableAction = "decline"
)

type Ticket struct {
	ID                 uuid.UUID
	ListingID          uuid.UUID
	SKUID              uuid.UUID
	Status             TicketStatus
	IssuedAt           time.Time
	ActivationDeadline time.Time
	ActivatedAt        *time.Time
	OrderID            *uuid.UUID
	CheckoutURL        *string
	FinishedAt         *time.Time
	CloseReason        *TicketCloseReason
	AvailableActions   []TicketAvailableAction
}

func (t Ticket) ActionsAt(now time.Time) []TicketAvailableAction {
	actions := make([]TicketAvailableAction, 0, 2)

	if t.Status == TicketStatusIssued && now.Before(t.ActivationDeadline) {
		actions = append(actions, TicketAvailableActionActivate, TicketAvailableActionDecline)
	}

	return actions
}
