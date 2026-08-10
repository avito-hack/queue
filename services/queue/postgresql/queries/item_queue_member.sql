-- name: CreateItemQueueMember :one
INSERT INTO item_queue_members (
    item_id,
    user_id,
    ticket_id,
    position,
    status,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING
    id,
    item_id,
    user_id,
    ticket_id,
    position,
    status,
    created_at;

-- name: GetItemQueueMemberByUserID :one
SELECT
    id,
    item_id,
    user_id,
    ticket_id,
    position,
    status,
    created_at
FROM item_queue_members
WHERE item_id = $1
AND user_id = $2;

-- name: GetAllItemQueueMembersByItemID :many
SELECT
    id,
    item_id,
    user_id,
    ticket_id,
    position,
    status,
    created_at
FROM item_queue_members
WHERE item_id = $1
ORDER BY position NULLS LAST;

-- name: GetAllItemQueueMembersByUserID :many
SELECT
    id,
    item_id,
    user_id,
    ticket_id,
    position,
    status,
    created_at
FROM item_queue_members
WHERE user_id = $1
ORDER BY created_at;

-- name: UpdateItemQueueMember :exec
UPDATE item_queue_members
SET
    ticket_id = $3,
    position = $4,
    status = $5
WHERE item_id = $1
AND user_id = $2;

-- name: LeaveItemQueueMember :exec
UPDATE item_queue_members
SET
    status = $3,
    position = NULL
WHERE item_id = $1
AND user_id = $2;

-- name: ReactivateItemQueueMember :exec
UPDATE item_queue_members
SET
    status = 'waiting_in_line',
    position = $3,
    ticket_id = NULL
WHERE item_id = $1
AND user_id = $2;

-- name: DeleteAllItemQueueMembersByItemID :exec
DELETE FROM item_queue_members
WHERE item_id = $1;

-- name: ExistsItemQueueMember :one
SELECT EXISTS(
    SELECT 1
    FROM item_queue_members
    WHERE item_id = $1
    AND user_id = $2
);

-- name: GetItemQueueMemberPosition :one
SELECT position
FROM item_queue_members
WHERE item_id = $1
AND user_id = $2;

-- name: ShiftItemQueueMembersPositions :exec
WITH negated AS (
    UPDATE item_queue_members AS source
    SET position = -source.position
    WHERE source.item_id = $1
    AND source.position > $2
    RETURNING source.id
)
UPDATE item_queue_members AS target
SET position = -target.position - 1
WHERE target.id IN (SELECT id FROM negated);

-- name: GetUserQueueRank :one
SELECT rank
FROM (
    SELECT
        user_id,
        ROW_NUMBER() OVER (ORDER BY position) AS rank
    FROM item_queue_members
    WHERE item_id = $1
    AND status IN (
        'waiting_in_line',
        'acquired_purchase_rights',
        'placed_an_order'
    )
) ranked
WHERE user_id = $2;