-- name: CreateGroup :one
INSERT INTO groups (name, description, created_by_user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetGroup :one
SELECT * FROM groups
WHERE id = $1
LIMIT 1;

-- name: ListUserGroups :many
SELECT
    g.id,
    g.name,
    g.description,
    g.created_by_user_id,
    g.created_at,
    g.updated_at,
    gm.display_name AS my_display_name,
    gm.icon_url AS my_icon_url,
    gm.role AS my_role,
    gm.joined_at
FROM groups g
INNER JOIN group_members gm ON gm.group_id = g.id
WHERE gm.user_id = $1
ORDER BY gm.joined_at DESC;
