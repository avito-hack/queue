-- name: CreateItemQueue :exec
INSERT INTO item_queues (item_id, state, created_at, updated_at)
VALUES ($1, $2, $3, $4);

-- name: GetItemQueueByID :one
SELECT item_id, state, created_at, updated_at
FROM item_queues
WHERE item_id = $1;

-- name: UpdateItemQueue :exec
UPDATE item_queues
SET state = $2, updated_at = $3
WHERE item_id = $1;

-- name: DeleteItemQueue :exec
DELETE FROM item_queues
WHERE item_id = $1;

-- name: ExistsItemQueue :one
SELECT EXISTS(
    SELECT 1
    FROM item_queues
    WHERE item_id = $1
);