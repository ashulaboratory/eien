-- name: AddGroupMember :one
INSERT INTO group_members (user_id, group_id, display_name, icon_url, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListGroupMembers :many
SELECT * FROM group_members
WHERE group_id = $1
ORDER BY joined_at ASC;

-- name: GetGroupMember :one
SELECT * FROM group_members
WHERE user_id = $1 AND group_id = $2
LIMIT 1;

-- name: IsGroupMember :one
SELECT EXISTS(
    SELECT 1 FROM group_members
    WHERE user_id = $1 AND group_id = $2
) AS is_member;

-- name: UpdateGroupMemberProfile :one
UPDATE group_members
SET display_name = $3, icon_url = $4
WHERE user_id = $1 AND group_id = $2
RETURNING *;
