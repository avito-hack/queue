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
    position INT NOT NULL CHECK (position > 0),
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),

    UNIQUE(item_id, user_id),

    FOREIGN KEY (item_id) REFERENCES item_queues(item_id) ON DELETE CASCADE
);

CREATE INDEX idx_item_queue_members_item_position
ON item_queue_members(item_id, position);

CREATE INDEX idx_item_queue_members_user_id
ON item_queue_members(user_id);

CREATE UNIQUE INDEX idx_item_queue_members_unique_position
ON item_queue_members(item_id, position);

CREATE UNIQUE INDEX idx_item_queue_member_position
ON item_queue_members(item_id, position);

-- +goose Down

SELECT 'down SQL query';

DROP INDEX idx_item_queue_member_position;
DROP INDEX idx_item_queue_members_unique_position;
DROP INDEX idx_item_queue_members_item_position;
DROP INDEX idx_item_queue_members_user_id;

DROP TABLE item_queue_members;
DROP TABLE item_queues;