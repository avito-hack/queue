package postgresql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlgen "github.com/avito-hack/queue/services/tickets/gen/sql"
	"github.com/avito-hack/queue/services/tickets/internal/domain"
)

type LifecycleRepository struct {
	pool  *pgxpool.Pool
	newID func() uuid.UUID
	clock func() time.Time
}

func NewLifecycleRepository(pool *pgxpool.Pool) *LifecycleRepository {
	return &LifecycleRepository{pool: pool, newID: uuid.New, clock: time.Now}
}

func (r *LifecycleRepository) ExpireIssued(ctx context.Context, now time.Time, limit int) (int, error) {
	expired := 0
	err := pgx.BeginFunc(ctx, r.pool, func(transaction pgx.Tx) error {
		queries := sqlgen.New(transaction)
		rows, err := queries.SelectExpiredTickets(ctx, sqlgen.SelectExpiredTicketsParams{
			ExpiredAt: toPGTimestamptz(now),
			BatchSize: int32(limit),
		})
		if err != nil {
			return fmt.Errorf("select expired tickets: %w", err)
		}
		tickets, err := lifecycleTickets(rows)
		if err != nil {
			return fmt.Errorf("select expired tickets: %w", err)
		}
		for _, ticket := range tickets {
			rowsAffected, err := queries.CloseExpiredTicket(ctx, sqlgen.CloseExpiredTicketParams{
				FinishedAt: toPGTimestamptz(now),
				ID:         toPGUUID(ticket.ID),
			})
			if err != nil {
				return fmt.Errorf("close expired ticket: %w", err)
			}
			if rowsAffected == 0 {
				continue
			}
			if err := insertLifecycleOutbox(
				ctx,
				transaction,
				r.newID(),
				ticket,
				domain.TicketStatusClosed,
				domain.TicketCloseReasonActivationTimeout,
				domain.TicketEventClosed,
				now,
			); err != nil {
				return err
			}
			expired++
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return expired, nil
}

func (r *LifecycleRepository) RecoverStaleActivations(
	ctx context.Context,
	staleBefore time.Time,
	limit int,
) (int, error) {
	rowsAffected, err := sqlgen.New(r.pool).RecoverStaleActivations(ctx, sqlgen.RecoverStaleActivationsParams{
		UpdatedAt:   toPGTimestamptz(r.clock()),
		StaleBefore: toPGTimestamptz(staleBefore),
		BatchSize:   int32(limit),
	})
	if err != nil {
		return 0, fmt.Errorf("recover stale activation operations: %w", err)
	}

	return int(rowsAffected), nil
}

type lifecycleTicket struct {
	ID           uuid.UUID
	QueueEntryID uuid.UUID
	UserID       uuid.UUID
	ListingID    uuid.UUID
	SKUID        uuid.UUID
}

type lifecycleOutboxPayload struct {
	TicketID     uuid.UUID                `json:"ticket_id"`
	QueueEntryID uuid.UUID                `json:"queue_entry_id"`
	UserID       uuid.UUID                `json:"user_id"`
	ListingID    uuid.UUID                `json:"listing_id"`
	SKUID        uuid.UUID                `json:"sku_id"`
	Status       domain.TicketStatus      `json:"status"`
	CloseReason  domain.TicketCloseReason `json:"close_reason"`
	FinishedAt   time.Time                `json:"finished_at"`
}

func lifecycleTickets(rows []sqlgen.SelectExpiredTicketsRow) ([]lifecycleTicket, error) {
	tickets := make([]lifecycleTicket, 0, len(rows))
	for _, row := range rows {
		if !row.ID.Valid || !row.QueueEntryID.Valid || !row.UserID.Valid || !row.ListingID.Valid || !row.SkuID.Valid {
			return nil, fmt.Errorf("expired ticket has null required UUID")
		}
		tickets = append(tickets, lifecycleTicket{
			ID:           uuid.UUID(row.ID.Bytes),
			QueueEntryID: uuid.UUID(row.QueueEntryID.Bytes),
			UserID:       uuid.UUID(row.UserID.Bytes),
			ListingID:    uuid.UUID(row.ListingID.Bytes),
			SKUID:        uuid.UUID(row.SkuID.Bytes),
		})
	}

	return tickets, nil
}

func insertLifecycleOutbox(
	ctx context.Context,
	transaction pgx.Tx,
	eventID uuid.UUID,
	ticket lifecycleTicket,
	status domain.TicketStatus,
	reason domain.TicketCloseReason,
	eventType string,
	finishedAt time.Time,
) error {
	payload, err := json.Marshal(lifecycleOutboxPayload{
		TicketID:     ticket.ID,
		QueueEntryID: ticket.QueueEntryID,
		UserID:       ticket.UserID,
		ListingID:    ticket.ListingID,
		SKUID:        ticket.SKUID,
		Status:       status,
		CloseReason:  reason,
		FinishedAt:   finishedAt,
	})
	if err != nil {
		return fmt.Errorf("encode lifecycle event: %w", err)
	}
	rowsAffected, err := sqlgen.New(transaction).InsertLifecycleOutbox(ctx, sqlgen.InsertLifecycleOutboxParams{
		ID:          toPGUUID(eventID),
		AggregateID: toPGUUID(ticket.ID),
		EventType:   eventType,
		Payload:     payload,
		AvailableAt: toPGTimestamptz(finishedAt),
	})
	if err != nil {
		return fmt.Errorf("insert lifecycle event: %w", err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("insert lifecycle event: unexpected affected rows %d", rowsAffected)
	}

	return nil
}
