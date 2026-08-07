-- +goose Up
ALTER TABLE public.tickets
    DROP CONSTRAINT ck_tickets_state,
    DROP CONSTRAINT ck_tickets_close_reason,
    DROP CONSTRAINT ck_tickets_status;

DROP INDEX public.uq_tickets_live_user_listing;

UPDATE public.tickets
SET status = 'redeemed',
    finished_at = COALESCE(finished_at, activated_at, updated_at),
    close_reason = NULL
WHERE status = 'active';

UPDATE public.tickets
SET close_reason = NULL
WHERE status = 'redeemed' AND close_reason = 'payment_succeeded';

UPDATE public.tickets
SET close_reason = 'system_cancelled'
WHERE status = 'closed' AND close_reason IN ('payment_succeeded', 'reservation_released');

CREATE UNIQUE INDEX uq_tickets_live_user_listing
    ON public.tickets (user_id, listing_id)
    WHERE status = 'issued';

ALTER TABLE public.tickets
    ADD CONSTRAINT ck_tickets_status
        CHECK (status IN ('issued', 'redeemed', 'closed')),
    ADD CONSTRAINT ck_tickets_close_reason
        CHECK (close_reason IS NULL OR close_reason IN (
            'activation_timeout',
            'user_declined',
            'listing_closed',
            'sku_closed',
            'system_cancelled'
        )),
    ADD CONSTRAINT ck_tickets_state
        CHECK (
            (status = 'issued' AND activated_at IS NULL AND order_id IS NULL AND checkout_url IS NULL AND finished_at IS NULL AND close_reason IS NULL)
            OR
            (status = 'redeemed' AND activated_at IS NOT NULL AND order_id IS NOT NULL AND checkout_url IS NOT NULL AND finished_at IS NOT NULL AND close_reason IS NULL)
            OR
            (status = 'closed' AND finished_at IS NOT NULL AND close_reason IS NOT NULL)
        );

DROP TABLE public.inbox_events;

-- +goose Down
CREATE TABLE public.inbox_events (
    event_id uuid NOT NULL,
    event_type varchar(64) NOT NULL,
    source varchar(64) NOT NULL,
    payload jsonb NOT NULL,
    status varchar(16) NOT NULL,
    received_at timestamptz NOT NULL,
    processed_at timestamptz,
    last_error text,
    CONSTRAINT pk_inbox_events PRIMARY KEY (event_id),
    CONSTRAINT ck_inbox_events_status
        CHECK (status IN ('processing', 'processed', 'failed')),
    CONSTRAINT ck_inbox_events_processed
        CHECK (
            (status = 'processed' AND processed_at IS NOT NULL)
            OR
            (status IN ('processing', 'failed') AND processed_at IS NULL)
        )
);

ALTER TABLE public.tickets
    DROP CONSTRAINT ck_tickets_state,
    DROP CONSTRAINT ck_tickets_close_reason,
    DROP CONSTRAINT ck_tickets_status;

DROP INDEX public.uq_tickets_live_user_listing;

CREATE UNIQUE INDEX uq_tickets_live_user_listing
    ON public.tickets (user_id, listing_id)
    WHERE status IN ('issued', 'active');

ALTER TABLE public.tickets
    ADD CONSTRAINT ck_tickets_status
        CHECK (status IN ('issued', 'active', 'redeemed', 'closed')),
    ADD CONSTRAINT ck_tickets_close_reason
        CHECK (close_reason IS NULL OR close_reason IN (
            'payment_succeeded',
            'activation_timeout',
            'user_declined',
            'listing_closed',
            'sku_closed',
            'reservation_released',
            'system_cancelled'
        )),
    ADD CONSTRAINT ck_tickets_state
        CHECK (
            (status = 'issued' AND activated_at IS NULL AND order_id IS NULL AND checkout_url IS NULL AND finished_at IS NULL AND close_reason IS NULL)
            OR
            (status = 'active' AND activated_at IS NOT NULL AND order_id IS NOT NULL AND checkout_url IS NOT NULL AND finished_at IS NULL AND close_reason IS NULL)
            OR
            (status = 'redeemed' AND activated_at IS NOT NULL AND order_id IS NOT NULL AND checkout_url IS NOT NULL AND finished_at IS NOT NULL AND close_reason IS NULL)
            OR
            (status = 'closed' AND finished_at IS NOT NULL AND close_reason IS NOT NULL)
        );
