-- name: CreateItemQueueMember :exec
INSERT INTO item_queue_members (item_id, user_id, position, status, created_at)
VALUES ($1, $2, $3, $4, $5);


-- name: GetItemQueueMemberByUserID :one
SELECT item_id, user_id, position, status, created_at
FROM item_queue_members
WHERE item_id = $1 AND user_id = $2;


-- name: GetAllItemQueueMembersByItemID :many
SELECT item_id, user_id, position, status, created_at
FROM item_queue_members
WHERE item_id = $1
ORDER BY position;


-- name: GetAllItemQueueMembersByUserID :many
SELECT item_id, user_id, position, status, created_at
FROM item_queue_members
WHERE user_id = $1
ORDER BY created_at;


-- name: UpdateItemQueueMember :exec
UPDATE item_queue_members
SET position = $3, status = $4
WHERE item_id = $1 AND user_id = $2;


-- name: DeleteItemQueueMember :exec
DELETE FROM item_queue_members
WHERE item_id = $1 AND user_id = $2;


-- name: DeleteAllItemQueueMembersByItemID :exec
DELETE FROM item_queue_members
WHERE item_id = $1;


-- name: ExistsItemQueueMember :one
SELECT EXISTS(
    SELECT 1
    FROM item_queue_members
    WHERE item_id = $1 AND user_id = $2
);


-- name: CountItemQueueMembers :one
SELECT COUNT(*)
FROM item_queue_members
WHERE item_id = $1;

-- name: GetItemQueueMemberPosition :one
SELECT position
FROM item_queue_members
WHERE item_id = $1
AND user_id = $2;


-- name: ShiftItemQueueMembersPositions :exec
UPDATE item_queue_members
SET position = position - 1
WHERE item_id = $1
AND position > $2;