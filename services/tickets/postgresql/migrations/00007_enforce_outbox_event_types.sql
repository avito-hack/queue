-- +goose Up
DELETE FROM public.outbox_events
WHERE event_type NOT IN ('ticket.closed', 'ticket.redeemed');

ALTER TABLE public.outbox_events
    ADD CONSTRAINT ck_outbox_events_type
        CHECK (event_type IN ('ticket.closed', 'ticket.redeemed'));

-- +goose Down
ALTER TABLE public.outbox_events
    DROP CONSTRAINT ck_outbox_events_type;
