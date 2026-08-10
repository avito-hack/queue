-- name: ClaimOutboxEvents :many
WITH candidates AS (
    SELECT id
    FROM public.outbox_events
    WHERE (outbox_events.status = 'pending' AND outbox_events.available_at <= sqlc.arg(claimed_at))
       OR (outbox_events.status = 'processing' AND outbox_events.available_at <= sqlc.arg(claimed_at))
    ORDER BY outbox_events.available_at, outbox_events.id
    FOR UPDATE SKIP LOCKED
    LIMIT sqlc.arg(batch_size)
)
UPDATE public.outbox_events AS event
SET status = 'processing',
    attempts = attempts + 1,
    available_at = sqlc.arg(lease_deadline)
FROM candidates
WHERE event.id = candidates.id
RETURNING event.id, event.event_type, event.payload, event.attempts;

-- name: PublishOutboxEvent :execrows
UPDATE public.outbox_events
SET status = 'published',
    published_at = sqlc.arg(published_at)
WHERE id = sqlc.arg(id)
  AND status = 'processing';

-- name: RetryOutboxEvent :execrows
UPDATE public.outbox_events
SET status = 'pending',
    available_at = sqlc.arg(available_at)
WHERE id = sqlc.arg(id)
  AND status = 'processing';
