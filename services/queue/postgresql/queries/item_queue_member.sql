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
SELECT
    position
FROM item_queue_members
WHERE item_id = $1
AND user_id = $2;


-- name: ShiftItemQueueMembersPositions :exec
UPDATE item_queue_members
SET position = position - 1
WHERE item_id = $1
AND position > $2;


-- name: GetUserQueueRank :one
SELECT COUNT(*) + 1 AS rank
FROM item_queue_members outer_q
WHERE outer_q.item_id = $1
AND outer_q.position <= (
    SELECT inner_q.position 
    FROM item_queue_members inner_q 
    WHERE inner_q.item_id = $1 
    AND inner_q.user_id = $2
);



-- name: GetRealUserQueuePosition :one
SELECT COUNT(*) + 1 AS position
FROM item_queue_members
WHERE item_queue_members.item_id = $1
AND item_queue_members.status IN (
    'waiting_in_line',
    'acquired_purchase_rights',
    'placed_an_order'
)
AND item_queue_members.position < (
    SELECT position
    FROM item_queue_members
    WHERE item_queue_members.item_id = $1
    AND item_queue_members.user_id = $2
);

-- name: GetItemQueueMemberByTicketID :one
SELECT
    id,
    item_id,
    user_id,
    ticket_id,
    position,
    status,
    created_at
FROM item_queue_members
WHERE ticket_id = $1;