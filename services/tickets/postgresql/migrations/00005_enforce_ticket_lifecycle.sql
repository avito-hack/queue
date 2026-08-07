-- +goose Up
DROP INDEX public.uq_idempotency_operations_ticket_operation;

CREATE UNIQUE INDEX uq_idempotency_operations_ticket_operation
    ON public.idempotency_operations (ticket_id, operation)
    WHERE ticket_id IS NOT NULL AND state IN ('processing', 'failed');

CREATE INDEX idx_tickets_expiration
    ON public.tickets (activation_deadline, id)
    WHERE status = 'issued';

CREATE UNIQUE INDEX uq_tickets_live_user_listing
    ON public.tickets (user_id, listing_id)
    WHERE status = 'issued';

CREATE INDEX idx_activation_operations_recovery
    ON public.idempotency_operations (updated_at, id)
    WHERE operation = 'activate_ticket' AND state = 'processing';

CREATE INDEX idx_outbox_events_delivery
    ON public.outbox_events (available_at, id)
    WHERE status IN ('pending', 'processing');

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
    ADD CONSTRAINT ck_tickets_activation_deadline
        CHECK (activation_deadline > issued_at),
    ADD CONSTRAINT ck_tickets_version
        CHECK (version > 0),
    ADD CONSTRAINT ck_tickets_state
        CHECK (
            (status = 'issued' AND activated_at IS NULL AND order_id IS NULL AND checkout_url IS NULL AND finished_at IS NULL AND close_reason IS NULL)
            OR
            (status = 'redeemed' AND activated_at IS NOT NULL AND order_id IS NOT NULL AND checkout_url IS NOT NULL AND finished_at IS NOT NULL AND close_reason IS NULL)
            OR
            (status = 'closed' AND finished_at IS NOT NULL AND close_reason IS NOT NULL)
        );

ALTER TABLE public.idempotency_operations
    ADD CONSTRAINT ck_idempotency_operations_state
        CHECK (state IN ('processing', 'completed', 'failed')),
    ADD CONSTRAINT ck_idempotency_operations_response
        CHECK (
            (state = 'completed' AND response_status IS NOT NULL AND response_body IS NOT NULL)
            OR
            (state IN ('processing', 'failed') AND response_status IS NULL AND response_body IS NULL)
        );

ALTER TABLE public.outbox_events
    ADD CONSTRAINT ck_outbox_events_type
        CHECK (event_type IN ('ticket.closed', 'ticket.redeemed')),
    ADD CONSTRAINT ck_outbox_events_status
        CHECK (status IN ('pending', 'processing', 'published')),
    ADD CONSTRAINT ck_outbox_events_attempts
        CHECK (attempts >= 0),
    ADD CONSTRAINT ck_outbox_events_published
        CHECK (
            (status = 'published' AND published_at IS NOT NULL)
            OR
            (status IN ('pending', 'processing') AND published_at IS NULL)
        );

ALTER TABLE public.inbox_events
    ADD CONSTRAINT ck_inbox_events_status
        CHECK (status IN ('processing', 'processed', 'failed')),
    ADD CONSTRAINT ck_inbox_events_processed
        CHECK (
            (status = 'processed' AND processed_at IS NOT NULL)
            OR
            (status IN ('processing', 'failed') AND processed_at IS NULL)
        );

-- +goose Down
ALTER TABLE public.inbox_events
    DROP CONSTRAINT ck_inbox_events_processed,
    DROP CONSTRAINT ck_inbox_events_status;

ALTER TABLE public.outbox_events
    DROP CONSTRAINT ck_outbox_events_published,
    DROP CONSTRAINT ck_outbox_events_attempts,
    DROP CONSTRAINT ck_outbox_events_status,
    DROP CONSTRAINT ck_outbox_events_type;

ALTER TABLE public.idempotency_operations
    DROP CONSTRAINT ck_idempotency_operations_response,
    DROP CONSTRAINT ck_idempotency_operations_state;

ALTER TABLE public.tickets
    DROP CONSTRAINT ck_tickets_state,
    DROP CONSTRAINT ck_tickets_version,
    DROP CONSTRAINT ck_tickets_activation_deadline,
    DROP CONSTRAINT ck_tickets_close_reason,
    DROP CONSTRAINT ck_tickets_status;

DROP INDEX public.idx_outbox_events_delivery;
DROP INDEX public.idx_activation_operations_recovery;
DROP INDEX public.idx_tickets_expiration;
DROP INDEX public.uq_tickets_live_user_listing;
DROP INDEX public.uq_idempotency_operations_ticket_operation;

CREATE UNIQUE INDEX uq_idempotency_operations_ticket_operation
    ON public.idempotency_operations (ticket_id, operation)
    WHERE ticket_id IS NOT NULL;
