-- name: AddPostImage :one
INSERT INTO post_images (post_id, image_url, sort_order)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListPostImages :many
SELECT * FROM post_images
WHERE post_id = $1
ORDER BY sort_order ASC;
