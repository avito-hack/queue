-- name: GetTicketOrderID :one
SELECT order_id
FROM public.tickets
WHERE id = sqlc.arg(id);

-- name: SetTicketObsoleteStatus :exec
UPDATE public.tickets
SET status = 'active'
WHERE id = sqlc.arg(id);

-- name: CloseTicketWithUnknownReason :exec
UPDATE public.tickets
SET status = 'closed',
    finished_at = updated_at,
    close_reason = 'reservation_released'
WHERE id = sqlc.arg(id);

-- name: InboxTableExists :one
SELECT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = 'public'
      AND table_name = 'inbox_events'
);

-- name: InsertOutboxEventWithType :exec
INSERT INTO public.outbox_events (
    id,
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    status,
    attempts,
    available_at
) VALUES (
    sqlc.arg(id),
    'ticket',
    sqlc.arg(aggregate_id),
    sqlc.arg(event_type),
    '{}',
    'pending',
    0,
    sqlc.arg(available_at)
);

-- name: TruncateIntegrationTables :exec
TRUNCATE public.inbox_events, public.outbox_events, public.idempotency_operations, public.tickets CASCADE;

-- name: CountTickets :one
SELECT COUNT(*)
FROM public.tickets;

-- name: CountOutboxEvents :one
SELECT COUNT(*)
FROM public.outbox_events;

-- name: CountIdempotencyOperationsByOperation :one
SELECT COUNT(*)
FROM public.idempotency_operations
WHERE operation = sqlc.arg(operation);

-- name: CountOutboxEventsByType :one
SELECT COUNT(*)
FROM public.outbox_events
WHERE event_type = sqlc.arg(event_type);

-- name: CountOutboxEventsByStatus :one
SELECT COUNT(*)
FROM public.outbox_events
WHERE status = sqlc.arg(status);

-- name: CountInboxEvents :one
SELECT COUNT(*)
FROM public.inbox_events;

-- name: ListListingTicketStates :many
SELECT id, status, close_reason
FROM public.tickets
WHERE listing_id = sqlc.arg(listing_id)
ORDER BY issued_at, id;
