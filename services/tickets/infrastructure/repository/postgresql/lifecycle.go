package postgresql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/avito-hack/queue/services/tickets/internal/domain"
)

const selectExpiredTicketsQuery = `SELECT id, queue_entry_id, user_id, listing_id, sku_id
FROM public.tickets
WHERE status = 'issued' AND activation_deadline <= $1
ORDER BY activation_deadline, id
FOR UPDATE SKIP LOCKED
LIMIT $2`

const closeExpiredTicketQuery = `UPDATE public.tickets
SET status = 'closed',
    close_reason = 'activation_timeout',
    finished_at = $2,
    updated_at = $2,
    version = version + 1
WHERE id = $1 AND status = 'issued' AND activation_deadline <= $2`

const recoverStaleActivationsQuery = `WITH candidates AS (
    SELECT id
    FROM public.idempotency_operations
    WHERE operation = 'activate_ticket' AND state = 'processing' AND updated_at <= $1
    ORDER BY updated_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT $2
)
UPDATE public.idempotency_operations AS operation
SET state = 'failed', updated_at = $3
FROM candidates
WHERE operation.id = candidates.id AND operation.state = 'processing'`

const insertLifecycleOutboxQuery = `INSERT INTO public.outbox_events (
    id,
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    status,
    attempts,
    available_at,
    published_at
) VALUES ($1, 'ticket', $2, $3, $4, 'pending', 0, $5, NULL)`

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
		tickets, err := selectLifecycleTickets(ctx, transaction, selectExpiredTicketsQuery, now, limit)
		if err != nil {
			return fmt.Errorf("select expired tickets: %w", err)
		}
		for _, ticket := range tickets {
			commandTag, err := transaction.Exec(ctx, closeExpiredTicketQuery, toPGUUID(ticket.ID), now)
			if err != nil {
				return fmt.Errorf("close expired ticket: %w", err)
			}
			if commandTag.RowsAffected() == 0 {
				continue
			}
			if err := r.insertLifecycleOutbox(
				ctx,
				transaction,
				ticket,
				domain.TicketStatusClosed,
				domain.TicketCloseReasonActivationTimeout,
				"ticket.closed",
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
	commandTag, err := r.pool.Exec(ctx, recoverStaleActivationsQuery, staleBefore, limit, r.clock())
	if err != nil {
		return 0, fmt.Errorf("recover stale activation operations: %w", err)
	}

	return int(commandTag.RowsAffected()), nil
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

func selectLifecycleTickets(
	ctx context.Context,
	transaction pgx.Tx,
	query string,
	args ...any,
) ([]lifecycleTicket, error) {
	rows, err := transaction.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := make([]lifecycleTicket, 0)
	for rows.Next() {
		var ticket lifecycleTicket
		if err := rows.Scan(&ticket.ID, &ticket.QueueEntryID, &ticket.UserID, &ticket.ListingID, &ticket.SKUID); err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tickets, nil
}

func (r *LifecycleRepository) insertLifecycleOutbox(
	ctx context.Context,
	transaction pgx.Tx,
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
	commandTag, err := transaction.Exec(
		ctx,
		insertLifecycleOutboxQuery,
		toPGUUID(r.newID()),
		toPGUUID(ticket.ID),
		eventType,
		payload,
		finishedAt,
	)
	if err != nil {
		return fmt.Errorf("insert lifecycle event: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("insert lifecycle event: unexpected affected rows %d", commandTag.RowsAffected())
	}

	return nil
}
