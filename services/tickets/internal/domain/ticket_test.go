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
	// given
	tests := []struct {
		status   TicketStatus
		expected bool
	}{
		{status: TicketStatusIssued, expected: true},
		{status: TicketStatusActive, expected: true},
		{status: TicketStatusRedeemed, expected: true},
		{status: TicketStatusClosed, expected: true},
		{status: TicketStatus("unknown"), expected: false},
	}

	for _, test := range tests {
		// when / then
		assert.Equal(t, test.expected, test.status.Valid())
	}
}

func TestTicketCloseReason_Valid_ReturnValidity(t *testing.T) {
	// given
	tests := []struct {
		reason   TicketCloseReason
		expected bool
	}{
		{reason: TicketCloseReasonPaymentSucceeded, expected: true},
		{reason: TicketCloseReasonActivationTimeout, expected: true},
		{reason: TicketCloseReasonUserDeclined, expected: true},
		{reason: TicketCloseReasonListingClosed, expected: true},
		{reason: TicketCloseReasonSKUClosed, expected: true},
		{reason: TicketCloseReasonReservationReleased, expected: true},
		{reason: TicketCloseReasonSystemCancelled, expected: true},
		{reason: TicketCloseReason("unknown"), expected: false},
	}

	for _, test := range tests {
		// when / then
		assert.Equal(t, test.expected, test.reason.Valid())
	}
}
