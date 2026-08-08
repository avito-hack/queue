-- name: FindDeclineOperation :one
SELECT
    id,
    actor_id,
    ticket_id,
    request_hash,
    state,
    response_status,
    response_body
FROM public.idempotency_operations
WHERE actor_id = sqlc.arg(actor_id)
  AND operation = sqlc.arg(operation)
  AND idempotency_key = sqlc.arg(idempotency_key);

-- name: LockTicketForDecline :one
SELECT
    queue_entry_id,
    listing_id,
    sku_id,
    status,
    activation_deadline
FROM public.tickets
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
FOR UPDATE;

-- name: HasProcessingActivation :one
SELECT EXISTS (
    SELECT 1
    FROM public.idempotency_operations
    WHERE ticket_id = sqlc.arg(ticket_id)
      AND operation = sqlc.arg(operation)
      AND state = sqlc.arg(state)
);

-- name: InsertDeclineOperation :execrows
INSERT INTO public.idempotency_operations (
    id,
    idempotency_key,
    operation,
    actor_id,
    ticket_id,
    request_hash,
    state,
    response_status,
    response_body,
    expires_at,
    updated_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(idempotency_key),
    sqlc.arg(operation),
    sqlc.arg(actor_id),
    sqlc.arg(ticket_id),
    sqlc.arg(request_hash),
    sqlc.arg(state),
    NULL,
    NULL,
    sqlc.arg(expires_at),
    sqlc.arg(updated_at)
)
ON CONFLICT (actor_id, operation, idempotency_key) DO NOTHING;

-- name: DeclineTicket :execrows
UPDATE public.tickets
SET status = 'closed',
    close_reason = 'user_declined',
    finished_at = sqlc.arg(finished_at),
    updated_at = sqlc.arg(finished_at),
    version = version + 1
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
  AND status = 'issued'
  AND activation_deadline > sqlc.arg(finished_at);

-- name: CompleteDeclineOperation :execrows
UPDATE public.idempotency_operations
SET state = 'completed',
    response_status = sqlc.arg(response_status),
    response_body = sqlc.arg(response_body),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND state = 'processing';

-- name: InsertDeclineOutbox :execrows
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
    sqlc.arg(aggregate_type),
    sqlc.arg(aggregate_id),
    sqlc.arg(event_type),
    sqlc.arg(payload),
    sqlc.arg(status),
    0,
    sqlc.arg(available_at),
    NULL
);
