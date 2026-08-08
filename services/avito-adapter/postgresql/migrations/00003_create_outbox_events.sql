-- +goose Up
CREATE TABLE public.outbox_events (
    id uuid PRIMARY KEY,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL,
    available_at timestamptz NOT NULL,
    claimed_at timestamptz,
    lease_until timestamptz,
    published_at timestamptz,
    attempts integer NOT NULL DEFAULT 0,
    CONSTRAINT ck_outbox_events_type CHECK (event_type IN ('listing.quantity.changed', 'listing.status.changed')),
    CONSTRAINT ck_outbox_events_attempts CHECK (attempts >= 0)
);

CREATE INDEX idx_outbox_events_available_at
    ON public.outbox_events (available_at, created_at)
    WHERE published_at IS NULL;

-- +goose Down
DROP INDEX public.idx_outbox_events_available_at;
DROP TABLE public.outbox_events;
