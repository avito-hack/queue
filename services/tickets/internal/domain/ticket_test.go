package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTicket_ActionsAt_ReturnAvailableActions(t *testing.T) {
	// given
	now := time.Date(2026, time.August, 6, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		status   TicketStatus
		deadline time.Time
		expected []TicketAvailableAction
	}{
		{
			name:     "issued before deadline",
			status:   TicketStatusIssued,
			deadline: now.Add(time.Minute),
			expected: []TicketAvailableAction{TicketAvailableActionActivate, TicketAvailableActionDecline},
		},
		{
			name:     "issued at deadline",
			status:   TicketStatusIssued,
			deadline: now,
			expected: []TicketAvailableAction{},
		},
		{
			name:     "issued after deadline",
			status:   TicketStatusIssued,
			deadline: now.Add(-time.Minute),
			expected: []TicketAvailableAction{},
		},
		{
			name:     "active",
			status:   TicketStatusActive,
			deadline: now.Add(-time.Minute),
			expected: []TicketAvailableAction{TicketAvailableActionCheckout},
		},
		{
			name:     "redeemed",
			status:   TicketStatusRedeemed,
			deadline: now.Add(time.Minute),
			expected: []TicketAvailableAction{},
		},
		{
			name:     "closed",
			status:   TicketStatusClosed,
			deadline: now.Add(time.Minute),
			expected: []TicketAvailableAction{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ticket := Ticket{Status: test.status, ActivationDeadline: test.deadline}

			// when
			actions := ticket.ActionsAt(now)

			// then
			assert.Equal(t, test.expected, actions)
			assert.NotNil(t, actions)
		})
	}
}

func TestTicketStatus_Valid_ReturnValidity(t *testing.T) {
	tests := []struct {
		name     string
		status   TicketStatus
		expected bool
	}{
		{name: "issued", status: TicketStatusIssued, expected: true},
		{name: "active", status: TicketStatusActive, expected: true},
		{name: "redeemed", status: TicketStatusRedeemed, expected: true},
		{name: "closed", status: TicketStatusClosed, expected: true},
		{name: "unknown", status: TicketStatus("unknown"), expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			status := test.status

			// when
			valid := status.Valid()

			// then
			assert.Equal(t, test.expected, valid)
		})
	}
}

func TestTicketCloseReason_Valid_ReturnValidity(t *testing.T) {
	tests := []struct {
		name     string
		reason   TicketCloseReason
		expected bool
	}{
		{name: "payment succeeded", reason: TicketCloseReasonPaymentSucceeded, expected: true},
		{name: "activation timeout", reason: TicketCloseReasonActivationTimeout, expected: true},
		{name: "user declined", reason: TicketCloseReasonUserDeclined, expected: true},
		{name: "listing closed", reason: TicketCloseReasonListingClosed, expected: true},
		{name: "SKU closed", reason: TicketCloseReasonSKUClosed, expected: true},
		{name: "reservation released", reason: TicketCloseReasonReservationReleased, expected: true},
		{name: "system cancelled", reason: TicketCloseReasonSystemCancelled, expected: true},
		{name: "unknown", reason: TicketCloseReason("unknown"), expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			reason := test.reason

			// when
			valid := reason.Valid()

			// then
			assert.Equal(t, test.expected, valid)
		})
	}
}
