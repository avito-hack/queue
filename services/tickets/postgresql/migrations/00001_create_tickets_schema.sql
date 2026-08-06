-- +goose Up
CREATE TABLE public.tickets (
    id uuid NOT NULL,
    queue_entry_id uuid NOT NULL,
    user_id uuid NOT NULL,
    listing_id uuid NOT NULL,
    sku_id uuid NOT NULL,
    status varchar(32) NOT NULL,
    activation_deadline timestamptz NOT NULL,
    activated_at timestamptz,
    order_id uuid,
    checkout_url text,
    close_reason varchar(64),
    issued_at timestamptz NOT NULL,
    finished_at timestamptz,
    updated_at timestamptz NOT NULL,
    version bigint NOT NULL,
    CONSTRAINT pk_tickets PRIMARY KEY (id)
);

CREATE TABLE public.idempotency_operations (
    id uuid NOT NULL,
    idempotency_key uuid NOT NULL,
    operation varchar(32) NOT NULL,
    actor_id uuid NOT NULL,
    ticket_id uuid,
    request_hash varchar(64) NOT NULL,
    state varchar(16) NOT NULL,
    response_status integer,
    response_body jsonb,
    expires_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT pk_idempotency_operations PRIMARY KEY (id)
);

CREATE TABLE public.outbox_events (
    id uuid NOT NULL,
    aggregate_type varchar(32) NOT NULL,
    aggregate_id uuid NOT NULL,
    event_type varchar(64) NOT NULL,
    payload jsonb NOT NULL,
    status varchar(16) NOT NULL,
    attempts integer NOT NULL,
    available_at timestamptz NOT NULL,
    published_at timestamptz,
    CONSTRAINT pk_outbox_events PRIMARY KEY (id)
);

CREATE TABLE public.inbox_events (
    event_id uuid NOT NULL,
    event_type varchar(64) NOT NULL,
    source varchar(64) NOT NULL,
    payload jsonb NOT NULL,
    status varchar(16) NOT NULL,
    received_at timestamptz NOT NULL,
    processed_at timestamptz,
    last_error text,
    CONSTRAINT pk_inbox_events PRIMARY KEY (event_id)
);

-- +goose Down
DROP TABLE public.inbox_events;
DROP TABLE public.outbox_events;
DROP TABLE public.idempotency_operations;
DROP TABLE public.tickets;
