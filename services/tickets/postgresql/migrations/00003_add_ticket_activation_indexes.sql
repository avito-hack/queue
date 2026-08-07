-- +goose Up
CREATE UNIQUE INDEX uq_idempotency_operations_actor_operation_key
    ON public.idempotency_operations (actor_id, operation, idempotency_key);

CREATE UNIQUE INDEX uq_idempotency_operations_ticket_operation
    ON public.idempotency_operations (ticket_id, operation)
    WHERE ticket_id IS NOT NULL;

-- +goose Down
DROP INDEX public.uq_idempotency_operations_ticket_operation;
DROP INDEX public.uq_idempotency_operations_actor_operation_key;
