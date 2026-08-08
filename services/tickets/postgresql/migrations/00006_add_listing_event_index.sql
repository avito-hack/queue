-- +goose Up
CREATE INDEX idx_tickets_listing_issued_at_id
    ON public.tickets (listing_id, issued_at, id)
    WHERE status = 'issued';

-- +goose Down
DROP INDEX public.idx_tickets_listing_issued_at_id;
