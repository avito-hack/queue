package postgresql

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	sqlgen "github.com/avito-hack/queue/services/tickets/gen/sql"
	"github.com/avito-hack/queue/services/tickets/internal/domain"
	"github.com/avito-hack/queue/services/tickets/internal/usecase"
)

type ListingEventRepository struct {
	pool  *pgxpool.Pool
	newID func() uuid.UUID
}

func NewListingEventRepository(pool *pgxpool.Pool) *ListingEventRepository {
	return &ListingEventRepository{pool: pool, newID: uuid.New}
}

func (r *ListingEventRepository) RevokeListingTickets(
	ctx context.Context,
	command usecase.RevokeListingTicketsCommand,
) (int, error) {
	revoked := 0
	err := pgx.BeginFunc(ctx, r.pool, func(transaction pgx.Tx) error {
		queries := sqlgen.New(transaction)
		inserted, err := queries.InsertInboxEvent(ctx, sqlgen.InsertInboxEventParams{
			EventID:    toPGUUID(command.Event.ID),
			EventType:  command.Event.Type,
			Source:     command.Event.Source,
			Payload:    command.Event.Payload,
			ReceivedAt: toPGTimestamptz(command.HandledAt),
		})
		if err != nil {
			return fmt.Errorf("insert listing inbox event: %w", err)
		}
		if inserted == 0 {
			return nil
		}

		active, err := queries.CountActiveListingTickets(ctx, sqlgen.CountActiveListingTicketsParams{
			ListingID: toPGUUID(command.ListingID),
			ActiveAt:  toPGTimestamptz(command.HandledAt),
		})
		if err != nil {
			return fmt.Errorf("count active listing tickets: %w", err)
		}
		closeCount := int64(0)
		switch {
		case command.CloseAll:
			closeCount = active
		case command.RevocationLimit > 0:
			closeCount = min(active, int64(command.RevocationLimit))
		}
		if closeCount < 0 {
			closeCount = 0
		}
		if closeCount > math.MaxInt32 {
			closeCount = math.MaxInt32
		}

		rows, err := queries.SelectOldestRevocableListingTickets(
			ctx,
			sqlgen.SelectOldestRevocableListingTicketsParams{
				ListingID:   toPGUUID(command.ListingID),
				ActiveAt:    toPGTimestamptz(command.HandledAt),
				TicketLimit: int32(closeCount),
			},
		)
		if err != nil {
			return fmt.Errorf("select oldest revocable listing tickets: %w", err)
		}
		tickets, err := revocableListingTickets(rows)
		if err != nil {
			return fmt.Errorf("select oldest revocable listing tickets: %w", err)
		}

		for _, ticket := range tickets {
			rowsAffected, err := queries.CloseRevokedListingTicket(ctx, sqlgen.CloseRevokedListingTicketParams{
				CloseReason: toPGText(string(command.CloseReason)),
				FinishedAt:  toPGTimestamptz(command.HandledAt),
				TicketID:    toPGUUID(ticket.ID),
			})
			if err != nil {
				return fmt.Errorf("close revoked listing ticket: %w", err)
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
				command.CloseReason,
				domain.TicketEventClosed,
				command.HandledAt,
			); err != nil {
				return err
			}
			revoked++
		}

		processed, err := queries.MarkInboxEventProcessed(ctx, sqlgen.MarkInboxEventProcessedParams{
			ProcessedAt: toPGTimestamptz(command.HandledAt),
			EventID:     toPGUUID(command.Event.ID),
		})
		if err != nil {
			return fmt.Errorf("mark listing inbox event processed: %w", err)
		}
		if processed != 1 {
			return fmt.Errorf("mark listing inbox event processed: unexpected affected rows %d", processed)
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return revoked, nil
}

func revocableListingTickets(rows []sqlgen.SelectOldestRevocableListingTicketsRow) ([]lifecycleTicket, error) {
	tickets := make([]lifecycleTicket, 0, len(rows))
	for _, row := range rows {
		if !row.ID.Valid || !row.QueueEntryID.Valid || !row.UserID.Valid || !row.ListingID.Valid || !row.SkuID.Valid {
			return nil, fmt.Errorf("revocable ticket has null required UUID")
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
