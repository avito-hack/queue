-- name: FindActivationOperation :one
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
  AND idempotency_key = sqlc.arg(idempotency_key)
FOR UPDATE;

-- name: GetActivationOperation :one
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

-- name: LockActivationOperation :one
SELECT
    id,
    actor_id,
    ticket_id,
    request_hash,
    state,
    response_status,
    response_body
FROM public.idempotency_operations
WHERE id = sqlc.arg(id)
  AND operation = sqlc.arg(operation)
FOR UPDATE;

-- name: LockTicketForActivation :one
SELECT
    listing_id,
    sku_id,
    status,
    activation_deadline
FROM public.tickets
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
FOR UPDATE;

-- name: FindProcessingTicketActivation :one
SELECT id
FROM public.idempotency_operations
WHERE ticket_id = sqlc.arg(ticket_id)
  AND operation = sqlc.arg(operation)
  AND state = sqlc.arg(state);

-- name: InsertActivationOperation :execrows
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

-- name: ActivateTicket :execrows
UPDATE public.tickets
SET status = 'redeemed',
    activated_at = sqlc.arg(completed_at),
    order_id = sqlc.arg(order_id),
    checkout_url = sqlc.arg(checkout_url),
    finished_at = sqlc.arg(completed_at),
    updated_at = sqlc.arg(completed_at),
    version = version + 1
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
  AND status = 'issued';

-- name: CompleteActivationOperation :execrows
UPDATE public.idempotency_operations
SET state = 'completed',
    response_status = sqlc.arg(response_status),
    response_body = sqlc.arg(response_body),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND state = 'processing';

-- name: FailActivationOperation :execrows
UPDATE public.idempotency_operations
SET state = 'failed',
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND operation = 'activate_ticket'
  AND state = 'processing';

-- name: RetryActivationOperation :execrows
UPDATE public.idempotency_operations
SET state = 'processing',
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND operation = 'activate_ticket'
  AND state = 'failed';

-- name: InsertActivationOutbox :execrows
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
