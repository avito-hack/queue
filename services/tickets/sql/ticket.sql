-- name: GetTicket :one
SELECT
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
WHERE user_id = sqlc.arg(user_id)
  AND id = sqlc.arg(id);

-- name: ListTickets :many
SELECT
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
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
  AND (sqlc.narg(listing_id)::uuid IS NULL OR listing_id = sqlc.narg(listing_id))
  AND (sqlc.narg(sku_id)::uuid IS NULL OR sku_id = sqlc.narg(sku_id))
ORDER BY issued_at DESC, id DESC;
