package domain

import (
	"time"

	"github.com/google/uuid"
)

type TicketStatus string

const (
	TicketStatusIssued   TicketStatus = "issued"
	TicketStatusActive   TicketStatus = "active"
	TicketStatusRedeemed TicketStatus = "redeemed"
	TicketStatusClosed   TicketStatus = "closed"
)

func (s TicketStatus) Valid() bool {
	switch s {
	case TicketStatusIssued, TicketStatusActive, TicketStatusRedeemed, TicketStatusClosed:
		return true
	default:
		return false
	}
}

type TicketCloseReason string

const (
	TicketCloseReasonPaymentSucceeded    TicketCloseReason = "payment_succeeded"
	TicketCloseReasonActivationTimeout   TicketCloseReason = "activation_timeout"
	TicketCloseReasonUserDeclined        TicketCloseReason = "user_declined"
	TicketCloseReasonListingClosed       TicketCloseReason = "listing_closed"
	TicketCloseReasonSKUClosed           TicketCloseReason = "sku_closed"
	TicketCloseReasonReservationReleased TicketCloseReason = "reservation_released"
	TicketCloseReasonSystemCancelled     TicketCloseReason = "system_cancelled"
)

func (r TicketCloseReason) Valid() bool {
	switch r {
	case TicketCloseReasonPaymentSucceeded,
		TicketCloseReasonActivationTimeout,
		TicketCloseReasonUserDeclined,
		TicketCloseReasonListingClosed,
		TicketCloseReasonSKUClosed,
		TicketCloseReasonReservationReleased,
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
	TicketAvailableActionCheckout TicketAvailableAction = "checkout"
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

	switch t.Status {
	case TicketStatusIssued:
		if now.Before(t.ActivationDeadline) {
			actions = append(actions, TicketAvailableActionActivate, TicketAvailableActionDecline)
		}
	case TicketStatusActive:
		actions = append(actions, TicketAvailableActionCheckout)
	}

	return actions
}
