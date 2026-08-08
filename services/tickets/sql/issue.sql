-- name: FindIssueOperation :one
SELECT
    id,
    actor_id,
    request_hash,
    state,
    response_status,
    response_body
FROM public.idempotency_operations
WHERE actor_id = sqlc.arg(actor_id)
  AND operation = sqlc.arg(operation)
  AND idempotency_key = sqlc.arg(idempotency_key);

-- name: InsertIssueOperation :execrows
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
    NULL,
    sqlc.arg(request_hash),
    sqlc.arg(state),
    NULL,
    NULL,
    sqlc.arg(expires_at),
    sqlc.arg(updated_at)
)
ON CONFLICT (actor_id, operation, idempotency_key) DO NOTHING;

-- name: InsertIssuedTicket :execrows
WITH listing_lock AS (
    SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(listing_id)::uuid::text, 0))
)
INSERT INTO public.tickets (
    id,
    queue_entry_id,
    user_id,
    listing_id,
    sku_id,
    status,
    activation_deadline,
    activated_at,
    order_id,
    checkout_url,
    close_reason,
    issued_at,
    finished_at,
    updated_at,
    version
) SELECT
    sqlc.arg(id),
    sqlc.arg(queue_entry_id),
    sqlc.arg(user_id),
    sqlc.arg(listing_id)::uuid,
    sqlc.arg(sku_id),
    'issued',
    sqlc.arg(activation_deadline),
    NULL,
    NULL,
    NULL,
    NULL,
    sqlc.arg(issued_at),
    NULL,
    sqlc.arg(issued_at),
    1
FROM listing_lock
ON CONFLICT (queue_entry_id) DO NOTHING;

-- name: FindTicketByQueueEntry :one
SELECT
    id,
    queue_entry_id,
    user_id,
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
WHERE queue_entry_id = sqlc.arg(queue_entry_id);

-- name: CompleteIssueOperation :execrows
UPDATE public.idempotency_operations
SET state = 'completed',
    response_status = sqlc.arg(response_status),
    response_body = sqlc.arg(response_body),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND state = 'processing';
