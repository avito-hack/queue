package postgresql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlgen "github.com/avito-hack/queue/services/tickets/gen/sql"
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
	tickets []databaseTicket
	scanErr error
	rowsErr error
	closed  bool
}

type rowStub struct {
	ticket  databaseTicket
	scanErr error
}

type queryerStub struct {
	rows               *rowsStub
	row                *rowStub
	err                error
	receivedGetParams  sqlgen.GetTicketParams
	receivedListParams sqlgen.ListTicketsParams
}

func (s *queryerStub) GetTicket(_ context.Context, params sqlgen.GetTicketParams) (sqlgen.GetTicketRow, error) {
	s.receivedGetParams = params
	if s.err != nil {
		return sqlgen.GetTicketRow{}, s.err
	}
	if s.row.scanErr != nil {
		return sqlgen.GetTicketRow{}, s.row.scanErr
	}

	return getTicketRow(s.row.ticket), nil
}

func (s *queryerStub) ListTickets(_ context.Context, params sqlgen.ListTicketsParams) ([]sqlgen.ListTicketsRow, error) {
	s.receivedListParams = params
	if s.err != nil {
		return nil, s.err
	}
	if s.rows == nil {
		return []sqlgen.ListTicketsRow{}, nil
	}
	s.rows.closed = true
	if s.rows.scanErr != nil {
		return nil, s.rows.scanErr
	}
	if s.rows.rowsErr != nil {
		return nil, s.rows.rowsErr
	}

	rows := make([]sqlgen.ListTicketsRow, 0, len(s.rows.tickets))
	for _, ticket := range s.rows.tickets {
		rows = append(rows, listTicketRow(ticket))
	}

	return rows, nil
}

func getTicketRow(ticket databaseTicket) sqlgen.GetTicketRow {
	return sqlgen.GetTicketRow{
		ID:                 ticket.id,
		ListingID:          ticket.listingID,
		SkuID:              ticket.skuID,
		Status:             ticket.status,
		IssuedAt:           toPGTimestamptz(ticket.issuedAt),
		ActivationDeadline: toPGTimestamptz(ticket.activationDeadline),
		ActivatedAt:        ticket.activatedAt,
		OrderID:            ticket.orderID,
		CheckoutUrl:        ticket.checkoutURL,
		FinishedAt:         ticket.finishedAt,
		CloseReason:        ticket.closeReason,
	}
}

func listTicketRow(ticket databaseTicket) sqlgen.ListTicketsRow {
	return sqlgen.ListTicketsRow{
		ID:                 ticket.id,
		ListingID:          ticket.listingID,
		SkuID:              ticket.skuID,
		Status:             ticket.status,
		IssuedAt:           toPGTimestamptz(ticket.issuedAt),
		ActivationDeadline: toPGTimestamptz(ticket.activationDeadline),
		ActivatedAt:        ticket.activatedAt,
		OrderID:            ticket.orderID,
		CheckoutUrl:        ticket.checkoutURL,
		FinishedAt:         ticket.finishedAt,
		CloseReason:        ticket.closeReason,
	}
}

func TestTicketRepository_Get_ReturnTicket(t *testing.T) {
	// given
	userID := uuid.New()
	ticketID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	issuedAt := time.Date(2026, time.August, 6, 10, 0, 0, 0, time.UTC)
	activationDeadline := issuedAt.Add(15 * time.Minute)
	queryer := &queryerStub{row: &rowStub{ticket: databaseTicket{
		id:                 toPGUUID(ticketID),
		listingID:          toPGUUID(listingID),
		skuID:              toPGUUID(skuID),
		status:             string(domain.TicketStatusIssued),
		issuedAt:           issuedAt,
		activationDeadline: activationDeadline,
	}}}
	repository := &TicketRepository{queryer: queryer}

	// when
	ticket, err := repository.Get(context.Background(), userID, ticketID)

	// then
	require.NoError(t, err)
	assert.Equal(t, ticketID, ticket.ID)
	assert.Equal(t, listingID, ticket.ListingID)
	assert.Equal(t, skuID, ticket.SKUID)
	assert.Equal(t, domain.TicketStatusIssued, ticket.Status)
	assert.Equal(t, issuedAt, ticket.IssuedAt)
	assert.Equal(t, activationDeadline, ticket.ActivationDeadline)
	assert.Equal(t, sqlgen.GetTicketParams{
		UserID: toPGUUID(userID),
		ID:     toPGUUID(ticketID),
	}, queryer.receivedGetParams)
}

func TestTicketRepository_Get_TicketDoesNotExist_ReturnTicketNotFound(t *testing.T) {
	// given
	repository := &TicketRepository{queryer: &queryerStub{row: &rowStub{scanErr: pgx.ErrNoRows}}}

	// when
	_, err := repository.Get(context.Background(), uuid.New(), uuid.New())

	// then
	assert.ErrorIs(t, err, usecase.ErrTicketNotFound)
}

func TestTicketRepository_Get_ScanReturnsError_ReturnWrappedError(t *testing.T) {
	// given
	scanError := errors.New("scan failed")
	repository := &TicketRepository{queryer: &queryerStub{row: &rowStub{scanErr: scanError}}}

	// when
	_, err := repository.Get(context.Background(), uuid.New(), uuid.New())

	// then
	require.Error(t, err)
	assert.ErrorIs(t, err, scanError)
	assert.Equal(t, "scan failed", err.Error())
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
	assert.Nil(t, tickets[0].CloseReason)
	assert.Equal(t, sqlgen.ListTicketsParams{UserID: toPGUUID(userID)}, queryer.receivedListParams)
	assert.True(t, resultRows.closed)
}

func TestTicketRepository_List_WithFilters_ReturnScopedQuery(t *testing.T) {
	// given
	userID := uuid.New()
	listingID := uuid.New()
	skuID := uuid.New()
	status := domain.TicketStatusRedeemed
	queryer := &queryerStub{rows: &rowsStub{}}
	repository := &TicketRepository{queryer: queryer}
	filter := usecase.ListTicketsFilter{Status: &status, ListingID: &listingID, SKUID: &skuID}

	// when
	tickets, err := repository.List(context.Background(), userID, filter)

	// then
	require.NoError(t, err)
	assert.Empty(t, tickets)
	assert.NotNil(t, tickets)
	assert.Equal(t, sqlgen.ListTicketsParams{
		UserID:    toPGUUID(userID),
		Status:    toPGText(string(domain.TicketStatusRedeemed)),
		ListingID: toPGUUID(listingID),
		SkuID:     toPGUUID(skuID),
	}, queryer.receivedListParams)
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
	assert.Equal(t, "query tickets: scan failed", err.Error())
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
	assert.Equal(t, "query tickets: rows failed", err.Error())
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
