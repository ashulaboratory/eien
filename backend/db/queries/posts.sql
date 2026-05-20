-- name: CreatePost :one
INSERT INTO posts (group_id, author_user_id, body)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetPost :one
SELECT * FROM posts
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: SoftDeletePost :exec
UPDATE posts SET deleted_at = NOW()
WHERE id = $1 AND author_user_id = $2;

-- name: ListGroupPosts :many
SELECT
    p.id,
    p.group_id,
    p.author_user_id,
    p.body,
    p.created_at,
    gm.display_name AS author_display_name,
    gm.icon_url AS author_icon_url
FROM posts p
INNER JOIN group_members gm
    ON gm.user_id = p.author_user_id AND gm.group_id = p.group_id
WHERE p.group_id = $1 AND p.deleted_at IS NULL
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTimeline :many
SELECT
    p.id,
    p.group_id,
    p.author_user_id,
    p.body,
    p.created_at,
    g.name AS group_name,
    gm.display_name AS author_display_name,
    gm.icon_url AS author_icon_url
FROM posts p
INNER JOIN groups g ON g.id = p.group_id
INNER JOIN group_members gm
    ON gm.user_id = p.author_user_id AND gm.group_id = p.group_id
WHERE p.group_id IN (
    SELECT gm_timeline.group_id FROM group_members gm_timeline WHERE gm_timeline.user_id = $1
)
  AND p.deleted_at IS NULL
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;
