package postgresql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlgen "github.com/avito-hack/queue/services/tickets/gen/sql"
	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type queryer interface {
	GetTicket(context.Context, sqlgen.GetTicketParams) (sqlgen.GetTicketRow, error)
	ListTickets(context.Context, sqlgen.ListTicketsParams) ([]sqlgen.ListTicketsRow, error)
}

type TicketRepository struct {
	queryer queryer
}

func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{queryer: sqlgen.New(pool)}
}

func (r *TicketRepository) Get(ctx context.Context, userID, ticketID uuid.UUID) (domain.Ticket, error) {
	row, err := r.queryer.GetTicket(ctx, sqlgen.GetTicketParams{
		UserID: toPGUUID(userID),
		ID:     toPGUUID(ticketID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Ticket{}, usecase.ErrTicketNotFound
	}
	if err != nil {
		return domain.Ticket{}, err
	}

	return ticketFromGetRow(row)
}

func (r *TicketRepository) List(ctx context.Context, userID uuid.UUID, filter usecase.ListTicketsFilter) ([]domain.Ticket, error) {
	params := sqlgen.ListTicketsParams{UserID: toPGUUID(userID)}
	if filter.Status != nil {
		params.Status = toPGText(string(*filter.Status))
	}
	if filter.ListingID != nil {
		params.ListingID = toPGUUID(*filter.ListingID)
	}
	if filter.SKUID != nil {
		params.SkuID = toPGUUID(*filter.SKUID)
	}

	rows, err := r.queryer.ListTickets(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("query tickets: %w", err)
	}

	tickets := make([]domain.Ticket, 0, len(rows))
	for _, row := range rows {
		ticket, err := ticketFromListRow(row)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

func ticketFromGetRow(row sqlgen.GetTicketRow) (domain.Ticket, error) {
	return ticketFromFields(
		row.ID,
		row.ListingID,
		row.SkuID,
		row.Status,
		row.IssuedAt,
		row.ActivationDeadline,
		row.ActivatedAt,
		row.OrderID,
		row.CheckoutUrl,
		row.FinishedAt,
		row.CloseReason,
	)
}

func ticketFromListRow(row sqlgen.ListTicketsRow) (domain.Ticket, error) {
	return ticketFromFields(
		row.ID,
		row.ListingID,
		row.SkuID,
		row.Status,
		row.IssuedAt,
		row.ActivationDeadline,
		row.ActivatedAt,
		row.OrderID,
		row.CheckoutUrl,
		row.FinishedAt,
		row.CloseReason,
	)
}

func ticketFromFields(
	id pgtype.UUID,
	listingID pgtype.UUID,
	skuID pgtype.UUID,
	status string,
	issuedAt pgtype.Timestamptz,
	activationDeadline pgtype.Timestamptz,
	activatedAt pgtype.Timestamptz,
	orderID pgtype.UUID,
	checkoutURL pgtype.Text,
	finishedAt pgtype.Timestamptz,
	closeReason pgtype.Text,
) (domain.Ticket, error) {
	if !id.Valid || !listingID.Valid || !skuID.Valid {
		return domain.Ticket{}, fmt.Errorf("scan ticket: required UUID is null")
	}
	if !issuedAt.Valid || !activationDeadline.Valid {
		return domain.Ticket{}, fmt.Errorf("scan ticket: required timestamp is null")
	}
	ticket := domain.Ticket{
		ID:                 uuid.UUID(id.Bytes),
		ListingID:          uuid.UUID(listingID.Bytes),
		SKUID:              uuid.UUID(skuID.Bytes),
		Status:             domain.TicketStatus(status),
		IssuedAt:           issuedAt.Time,
		ActivationDeadline: activationDeadline.Time,
	}
	if !ticket.Status.Valid() {
		return domain.Ticket{}, fmt.Errorf("scan ticket: unknown status %q", status)
	}

	if activatedAt.Valid {
		value := activatedAt.Time
		ticket.ActivatedAt = &value
	}
	if orderID.Valid {
		value := uuid.UUID(orderID.Bytes)
		ticket.OrderID = &value
	}
	if checkoutURL.Valid {
		value := checkoutURL.String
		ticket.CheckoutURL = &value
	}
	if finishedAt.Valid {
		value := finishedAt.Time
		ticket.FinishedAt = &value
	}
	if closeReason.Valid {
		value := domain.TicketCloseReason(closeReason.String)
		if !value.Valid() {
			return domain.Ticket{}, fmt.Errorf("scan ticket: unknown close reason %q", closeReason.String)
		}
		ticket.CloseReason = &value
	}

	return ticket, nil
}

func toPGUUID(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

func toPGTimestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func toPGText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: true}
}

func toPGInt4(value int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(value), Valid: true}
}
