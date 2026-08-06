-- +goose Up
CREATE INDEX idx_tickets_user_issued_at_id
    ON public.tickets (user_id, issued_at DESC, id DESC);

-- +goose Down
DROP INDEX public.idx_tickets_user_issued_at_id;
