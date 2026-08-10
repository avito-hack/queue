-- +goose Up

SELECT 'up SQL query';

CREATE TABLE item_queues (
    item_id UUID PRIMARY KEY,
    state TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE item_queue_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id UUID NOT NULL,
    user_id UUID NOT NULL,
    ticket_id UUID,
    position INT CHECK(position IS NULL OR position > 0),
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE(item_id, user_id),
    FOREIGN KEY(item_id) REFERENCES item_queues(item_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX item_queue_members_active_position_key
ON item_queue_members(item_id, position)
WHERE status IN ('waiting_in_line', 'acquired_purchase_rights', 'placed_an_order');

CREATE INDEX idx_item_queue_members_item_position
ON item_queue_members(item_id, position);

CREATE INDEX idx_item_queue_members_user_id
ON item_queue_members(user_id);

-- +goose Down

SELECT 'down SQL query';

DROP INDEX idx_item_queue_members_user_id;
DROP INDEX idx_item_queue_members_item_position;
DROP INDEX item_queue_members_active_position_key;

DROP TABLE item_queue_members;
DROP TABLE item_queues;
