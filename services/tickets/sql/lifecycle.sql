-- name: SelectExpiredTickets :many
SELECT id, queue_entry_id, user_id, listing_id, sku_id
FROM public.tickets AS ticket
WHERE ticket.status = 'issued'
  AND ticket.activation_deadline <= sqlc.arg(expired_at)
  AND NOT EXISTS (
      SELECT 1
      FROM public.idempotency_operations AS operation
      WHERE operation.ticket_id = ticket.id
        AND operation.operation = 'activate_ticket'
        AND operation.state = 'processing'
  )
ORDER BY ticket.activation_deadline, ticket.id
FOR UPDATE SKIP LOCKED
LIMIT sqlc.arg(batch_size);

-- name: CloseExpiredTicket :execrows
UPDATE public.tickets AS ticket
SET status = 'closed',
    close_reason = 'activation_timeout',
    finished_at = sqlc.arg(finished_at),
    updated_at = sqlc.arg(finished_at),
    version = version + 1
WHERE ticket.id = sqlc.arg(id)
  AND ticket.status = 'issued'
  AND ticket.activation_deadline <= sqlc.arg(finished_at)
  AND NOT EXISTS (
      SELECT 1
      FROM public.idempotency_operations AS operation
      WHERE operation.ticket_id = ticket.id
        AND operation.operation = 'activate_ticket'
        AND operation.state = 'processing'
  );

-- name: RecoverStaleActivations :execrows
WITH candidates AS (
    SELECT id
    FROM public.idempotency_operations
    WHERE idempotency_operations.operation = 'activate_ticket'
      AND idempotency_operations.state = 'processing'
      AND idempotency_operations.updated_at <= sqlc.arg(stale_before)
    ORDER BY idempotency_operations.updated_at, idempotency_operations.id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
)
UPDATE public.idempotency_operations AS operation
SET state = 'failed',
    updated_at = sqlc.arg(updated_at)
FROM candidates
WHERE operation.id = candidates.id
  AND operation.state = 'processing';

-- name: InsertLifecycleOutbox :execrows
INSERT INTO public.outbox_events (
    id,
    aggregate_type,
    aggregate_id,
    event_type,
    payload,
    status,
    attempts,
    available_at,
    published_at
) VALUES (
    sqlc.arg(id),
    'ticket',
    sqlc.arg(aggregate_id),
    sqlc.arg(event_type),
    sqlc.arg(payload),
    'pending',
    0,
    sqlc.arg(available_at),
    NULL
);
