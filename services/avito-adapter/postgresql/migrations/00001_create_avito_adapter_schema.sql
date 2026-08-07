-- +goose Up
CREATE TABLE public.users (
    id uuid NOT NULL,
    name text NOT NULL,
    token text NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT uq_users_token UNIQUE (token),
    CONSTRAINT ck_users_name CHECK (length(trim(name)) > 0),
    CONSTRAINT ck_users_token CHECK (length(trim(token)) > 0)
);

CREATE TABLE public.listings (
    id uuid NOT NULL,
    seller_id uuid NOT NULL,
    title text NOT NULL,
    price bigint NOT NULL,
    quantity integer NOT NULL,
    queue_enabled boolean NOT NULL DEFAULT false,
    status varchar(16) NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT pk_listings PRIMARY KEY (id),
    CONSTRAINT fk_listings_seller_id FOREIGN KEY (seller_id) REFERENCES public.users (id),
    CONSTRAINT ck_listings_title CHECK (length(trim(title)) > 0),
    CONSTRAINT ck_listings_price CHECK (price >= 0),
    CONSTRAINT ck_listings_quantity CHECK (quantity >= 0),
    CONSTRAINT ck_listings_status CHECK (status IN ('active', 'paused', 'removed'))
);

CREATE TABLE public.orders (
    id uuid NOT NULL,
    ticket_id uuid NOT NULL,
    listing_id uuid NOT NULL,
    sku_id uuid NOT NULL,
    user_id uuid NOT NULL,
    idempotency_key varchar(255) NOT NULL,
    checkout_url text NOT NULL,
    status varchar(16) NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT pk_orders PRIMARY KEY (id),
    CONSTRAINT fk_orders_listing_id FOREIGN KEY (listing_id) REFERENCES public.listings (id),
    CONSTRAINT fk_orders_user_id FOREIGN KEY (user_id) REFERENCES public.users (id),
    CONSTRAINT uq_orders_ticket_id UNIQUE (ticket_id),
    CONSTRAINT uq_orders_idempotency_key UNIQUE (idempotency_key),
    CONSTRAINT ck_orders_idempotency_key CHECK (length(trim(idempotency_key)) > 0),
    CONSTRAINT ck_orders_checkout_url CHECK (length(trim(checkout_url)) > 0),
    CONSTRAINT ck_orders_checkout_url_format CHECK (checkout_url ~ '^https?://[^/@]+(/|$)' OR checkout_url ~ '^/([^/]|$)'),
    CONSTRAINT ck_orders_status CHECK (status = 'created')
);

CREATE INDEX idx_listings_status_created_at_id
    ON public.listings (status, created_at DESC, id DESC);

CREATE INDEX idx_listings_seller_created_at_id
    ON public.listings (seller_id, created_at DESC, id DESC);

CREATE INDEX idx_orders_user_created_at_id
    ON public.orders (user_id, created_at DESC, id DESC);

CREATE INDEX idx_orders_sku_id
    ON public.orders (sku_id);

CREATE INDEX idx_orders_listing_id
    ON public.orders (listing_id);

-- +goose Down
DROP INDEX public.idx_orders_sku_id;
DROP INDEX public.idx_orders_listing_id;
DROP INDEX public.idx_orders_user_created_at_id;
DROP INDEX public.idx_listings_seller_created_at_id;
DROP INDEX public.idx_listings_status_created_at_id;
DROP TABLE public.orders;
DROP TABLE public.listings;
DROP TABLE public.users;
