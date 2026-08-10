-- +goose Up
CREATE UNIQUE INDEX uq_tickets_queue_entry_id
    ON public.tickets (queue_entry_id);

-- +goose Down
DROP INDEX public.uq_tickets_queue_entry_id;
