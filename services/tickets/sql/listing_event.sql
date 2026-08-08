-- name: InsertInboxEvent :execrows
INSERT INTO public.inbox_events (
    event_id,
    event_type,
    source,
    payload,
    status,
    received_at,
    processed_at,
    last_error
) VALUES (
    sqlc.arg(event_id),
    sqlc.arg(event_type),
    sqlc.arg(source),
    sqlc.arg(payload),
    'processing',
    sqlc.arg(received_at),
    NULL,
    NULL
)
ON CONFLICT (event_id) DO NOTHING;

-- name: CountActiveListingTickets :one
SELECT COUNT(*)
FROM public.tickets
WHERE listing_id = sqlc.arg(listing_id)
  AND status = 'issued'
  AND activation_deadline > sqlc.arg(active_at);

-- name: SelectOldestRevocableListingTickets :many
SELECT ticket.id, ticket.queue_entry_id, ticket.user_id, ticket.listing_id, ticket.sku_id
FROM public.tickets AS ticket
WHERE ticket.listing_id = sqlc.arg(listing_id)
  AND ticket.status = 'issued'
  AND ticket.activation_deadline > sqlc.arg(active_at)
  AND NOT EXISTS (
      SELECT 1
      FROM public.idempotency_operations AS operation
      WHERE operation.ticket_id = ticket.id
        AND operation.operation = 'activate_ticket'
        AND operation.state = 'processing'
  )
ORDER BY ticket.issued_at, ticket.id
FOR UPDATE
LIMIT sqlc.arg(ticket_limit);

-- name: CloseRevokedListingTicket :execrows
UPDATE public.tickets AS ticket
SET status = 'closed',
    close_reason = sqlc.arg(close_reason),
    finished_at = sqlc.arg(finished_at),
    updated_at = sqlc.arg(finished_at),
    version = version + 1
WHERE ticket.id = sqlc.arg(ticket_id)
  AND ticket.status = 'issued'
  AND ticket.activation_deadline > sqlc.arg(finished_at)
  AND NOT EXISTS (
      SELECT 1
      FROM public.idempotency_operations AS operation
      WHERE operation.ticket_id = ticket.id
        AND operation.operation = 'activate_ticket'
        AND operation.state = 'processing'
  );

-- name: MarkInboxEventProcessed :execrows
UPDATE public.inbox_events
SET status = 'processed',
    processed_at = sqlc.arg(processed_at),
    last_error = NULL
WHERE event_id = sqlc.arg(event_id)
  AND status = 'processing';
