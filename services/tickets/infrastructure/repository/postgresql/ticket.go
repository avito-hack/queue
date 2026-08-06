package postgresql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

const listTicketsQuery = `SELECT
    id,
    listing_id,
    sku_id,
    status,
    issued_at,
    activation_deadline,
    activated_at,
    order_id,
    checkout_url,
    finished_at,
    close_reason
FROM public.tickets
WHERE user_id = $1`

type rows interface {
	Close()
	Err() error
	Next() bool
	Scan(...any) error
}

type queryer interface {
	Query(context.Context, string, ...any) (rows, error)
}

type poolQueryer struct {
	pool *pgxpool.Pool
}

func (q poolQueryer) Query(ctx context.Context, query string, args ...any) (rows, error) {
	return q.pool.Query(ctx, query, args...)
}

type TicketRepository struct {
	queryer queryer
}

func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{queryer: poolQueryer{pool: pool}}
}

func (r *TicketRepository) List(ctx context.Context, userID uuid.UUID, filter usecase.ListTicketsFilter) ([]domain.Ticket, error) {
	query, args := buildListTicketsQuery(userID, filter)
	result, err := r.queryer.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tickets: %w", err)
	}
	defer result.Close()

	tickets := make([]domain.Ticket, 0)
	for result.Next() {
		ticket, err := scanTicket(result)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("read tickets: %w", err)
	}

	return tickets, nil
}

func buildListTicketsQuery(userID uuid.UUID, filter usecase.ListTicketsFilter) (string, []any) {
	var query strings.Builder
	query.WriteString(listTicketsQuery)
	args := []any{toPGUUID(userID)}

	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		query.WriteString(" AND status = $")
		query.WriteString(strconv.Itoa(len(args)))
	}
	if filter.ListingID != nil {
		args = append(args, toPGUUID(*filter.ListingID))
		query.WriteString(" AND listing_id = $")
		query.WriteString(strconv.Itoa(len(args)))
	}
	if filter.SKUID != nil {
		args = append(args, toPGUUID(*filter.SKUID))
		query.WriteString(" AND sku_id = $")
		query.WriteString(strconv.Itoa(len(args)))
	}

	query.WriteString(" ORDER BY issued_at DESC, id DESC")

	return query.String(), args
}

func scanTicket(result rows) (domain.Ticket, error) {
	var ticket domain.Ticket
	var id pgtype.UUID
	var listingID pgtype.UUID
	var skuID pgtype.UUID
	var status string
	var activatedAt pgtype.Timestamptz
	var orderID pgtype.UUID
	var checkoutURL pgtype.Text
	var finishedAt pgtype.Timestamptz
	var closeReason pgtype.Text

	err := result.Scan(
		&id,
		&listingID,
		&skuID,
		&status,
		&ticket.IssuedAt,
		&ticket.ActivationDeadline,
		&activatedAt,
		&orderID,
		&checkoutURL,
		&finishedAt,
		&closeReason,
	)
	if err != nil {
		return domain.Ticket{}, fmt.Errorf("scan ticket: %w", err)
	}

	if !id.Valid || !listingID.Valid || !skuID.Valid {
		return domain.Ticket{}, fmt.Errorf("scan ticket: required UUID is null")
	}
	ticket.ID = uuid.UUID(id.Bytes)
	ticket.ListingID = uuid.UUID(listingID.Bytes)
	ticket.SKUID = uuid.UUID(skuID.Bytes)
	ticket.Status = domain.TicketStatus(status)
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
