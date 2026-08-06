package postgresql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type databaseTicket struct {
	id                 pgtype.UUID
	listingID          pgtype.UUID
	skuID              pgtype.UUID
	status             string
	issuedAt           time.Time
	activationDeadline time.Time
	activatedAt        pgtype.Timestamptz
	orderID            pgtype.UUID
	checkoutURL        pgtype.Text
	finishedAt         pgtype.Timestamptz
	closeReason        pgtype.Text
}

type rowsStub struct {
	tickets  []databaseTicket
	position int
	scanErr  error
	rowsErr  error
	closed   bool
}

func (s *rowsStub) Close() {
	s.closed = true
}

func (s *rowsStub) Err() error {
	return s.rowsErr
}

func (s *rowsStub) Next() bool {
	if s.position >= len(s.tickets) {
		s.closed = true
		return false
	}
	s.position++

	return true
}

func (s *rowsStub) Scan(destinations ...any) error {
	if s.scanErr != nil {
		return s.scanErr
	}

	ticket := s.tickets[s.position-1]
	*destinations[0].(*pgtype.UUID) = ticket.id
	*destinations[1].(*pgtype.UUID) = ticket.listingID
	*destinations[2].(*pgtype.UUID) = ticket.skuID
	*destinations[3].(*string) = ticket.status
	*destinations[4].(*time.Time) = ticket.issuedAt
	*destinations[5].(*time.Time) = ticket.activationDeadline
	*destinations[6].(*pgtype.Timestamptz) = ticket.activatedAt
	*destinations[7].(*pgtype.UUID) = ticket.orderID
	*destinations[8].(*pgtype.Text) = ticket.checkoutURL
	*destinations[9].(*pgtype.Timestamptz) = ticket.finishedAt
	*destinations[10].(*pgtype.Text) = ticket.closeReason

	return nil
}

type queryerStub struct {
	rows          rows
	err           error
	receivedQuery string
	receivedArgs  []any
}

func (s *queryerStub) Query(_ context.Context, query string, args ...any) (rows, error) {
	s.receivedQuery = query
	s.receivedArgs = args

	return s.rows, s.err
}

func TestTicketRepository_List_ReturnTickets(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	orderID := uuid.New()
	issuedAt := time.Date(2026, time.August, 6, 10, 0, 0, 0, time.UTC)
	activationDeadline := issuedAt.Add(15 * time.Minute)
	activatedAt := issuedAt.Add(time.Minute)
	finishedAt := issuedAt.Add(10 * time.Minute)
	resultRows := &rowsStub{tickets: []databaseTicket{{
		id:                 toPGUUID(ticketID),
		listingID:          toPGUUID(listingID),
		skuID:              toPGUUID(skuID),
		status:             string(domain.TicketStatusRedeemed),
		issuedAt:           issuedAt,
		activationDeadline: activationDeadline,
		activatedAt:        pgtype.Timestamptz{Time: activatedAt, Valid: true},
		orderID:            toPGUUID(orderID),
		checkoutURL:        pgtype.Text{String: "/checkout/1", Valid: true},
		finishedAt:         pgtype.Timestamptz{Time: finishedAt, Valid: true},
		closeReason:        pgtype.Text{String: string(domain.TicketCloseReasonPaymentSucceeded), Valid: true},
	}}}
	queryer := &queryerStub{rows: resultRows}
	repository := &TicketRepository{queryer: queryer}

	// when
	tickets, err := repository.List(context.Background(), userID, usecase.ListTicketsFilter{})

	// then
	require.NoError(t, err)
	require.Len(t, tickets, 1)
	assert.Equal(t, ticketID, tickets[0].ID)
	assert.Equal(t, listingID, tickets[0].ListingID)
	assert.Equal(t, skuID, tickets[0].SKUID)
	assert.Equal(t, domain.TicketStatusRedeemed, tickets[0].Status)
	assert.Equal(t, issuedAt, tickets[0].IssuedAt)
	assert.Equal(t, activationDeadline, tickets[0].ActivationDeadline)
	assert.Equal(t, activatedAt, *tickets[0].ActivatedAt)
	assert.Equal(t, orderID, *tickets[0].OrderID)
	assert.Equal(t, "/checkout/1", *tickets[0].CheckoutURL)
	assert.Equal(t, finishedAt, *tickets[0].FinishedAt)
	assert.Equal(t, domain.TicketCloseReasonPaymentSucceeded, *tickets[0].CloseReason)
	assert.Contains(t, queryer.receivedQuery, "WHERE user_id = $1")
	assert.Contains(t, queryer.receivedQuery, "ORDER BY issued_at DESC, id DESC")
	assert.Equal(t, []any{toPGUUID(userID)}, queryer.receivedArgs)
	assert.True(t, resultRows.closed)
}

func TestTicketRepository_List_WithFilters_ReturnScopedQuery(t *testing.T) {
	// given
	userID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	status := domain.TicketStatusActive
	queryer := &queryerStub{rows: &rowsStub{}}
	repository := &TicketRepository{queryer: queryer}
	filter := usecase.ListTicketsFilter{Status: &status, ListingID: &listingID, SKUID: &skuID}

	// when
	tickets, err := repository.List(context.Background(), userID, filter)

	// then
	require.NoError(t, err)
	assert.Empty(t, tickets)
	assert.NotNil(t, tickets)
	assert.Contains(t, queryer.receivedQuery, "WHERE user_id = $1 AND status = $2 AND listing_id = $3 AND sku_id = $4")
	assert.Equal(t, []any{
		toPGUUID(userID),
		string(domain.TicketStatusActive),
		toPGUUID(listingID),
		toPGUUID(skuID),
	}, queryer.receivedArgs)
}

func TestTicketRepository_List_QueryReturnsError_ReturnWrappedError(t *testing.T) {
	// given
	queryError := errors.New("query failed")
	repository := &TicketRepository{queryer: &queryerStub{err: queryError}}

	// when
	_, err := repository.List(context.Background(), uuid.New(), usecase.ListTicketsFilter{})

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, queryError)
	assert.Equal(t, "query tickets: query failed", err.Error())
}

func TestTicketRepository_List_ScanReturnsError_ReturnWrappedError(t *testing.T) {
	// given
	scanError := errors.New("scan failed")
	resultRows := &rowsStub{tickets: []databaseTicket{{}}, scanErr: scanError}
	repository := &TicketRepository{queryer: &queryerStub{rows: resultRows}}

	// when
	_, err := repository.List(context.Background(), uuid.New(), usecase.ListTicketsFilter{})

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, scanError)
	assert.Equal(t, "scan ticket: scan failed", err.Error())
	assert.True(t, resultRows.closed)
}

func TestTicketRepository_List_RowsReturnError_ReturnWrappedError(t *testing.T) {
	// given
	rowsError := errors.New("rows failed")
	resultRows := &rowsStub{rowsErr: rowsError}
	repository := &TicketRepository{queryer: &queryerStub{rows: resultRows}}

	// when
	_, err := repository.List(context.Background(), uuid.New(), usecase.ListTicketsFilter{})

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, rowsError)
	assert.Equal(t, "read tickets: rows failed", err.Error())
}

func TestTicketRepository_List_UnknownStatus_ReturnError(t *testing.T) {
	// given
	resultRows := &rowsStub{tickets: []databaseTicket{{
		id:        toPGUUID(uuid.New()),
		listingID: toPGUUID(uuid.New()),
		skuID:     toPGUUID(uuid.New()),
		status:    "unknown",
	}}}
	repository := &TicketRepository{queryer: &queryerStub{rows: resultRows}}

	// when
	_, err := repository.List(context.Background(), uuid.New(), usecase.ListTicketsFilter{})

	// then
	require.EqualError(t, err, `scan ticket: unknown status "unknown"`)
}

func TestTicketRepository_List_UnknownCloseReason_ReturnError(t *testing.T) {
	// given
	resultRows := &rowsStub{tickets: []databaseTicket{{
		id:          toPGUUID(uuid.New()),
		listingID:   toPGUUID(uuid.New()),
		skuID:       toPGUUID(uuid.New()),
		status:      string(domain.TicketStatusClosed),
		closeReason: pgtype.Text{String: "unknown", Valid: true},
	}}}
	repository := &TicketRepository{queryer: &queryerStub{rows: resultRows}}

	// when
	_, err := repository.List(context.Background(), uuid.New(), usecase.ListTicketsFilter{})

	// then
	require.EqualError(t, err, `scan ticket: unknown close reason "unknown"`)
}
