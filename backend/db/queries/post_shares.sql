-- name: CreatePostShare :one
INSERT INTO post_shares (post_id, group_id)
VALUES ($1, $2)
RETURNING *;

-- name: DeletePostShare :exec
DELETE FROM post_shares
WHERE post_id = $1 AND group_id = $2;
