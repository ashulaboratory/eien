-- name: AddPostImage :one
INSERT INTO post_images (post_id, image_url, sort_order)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListPostImages :many
SELECT * FROM post_images
WHERE post_id = $1
ORDER BY sort_order ASC;

-- name: CanAccessPostImage :one
-- 指定ファイル名の画像にリクエストユーザーがアクセスできるか判定する。
-- 許可条件:
--   1. 投稿の著者本人である ( p.author_user_id = user_id )
--   2. その投稿がシェアされているいずれかのグループのメンバーである
-- 画像ファイル名 ($1) を image_url の末尾とマッチさせて post を逆引き。
SELECT EXISTS (
    SELECT 1
    FROM post_images pi
    JOIN posts p ON p.id = pi.post_id
    WHERE pi.image_url LIKE '%/' || $1
      AND p.deleted_at IS NULL
      AND (
        p.author_user_id = $2
        OR EXISTS (
            SELECT 1
            FROM post_shares ps
            JOIN group_members gm ON gm.group_id = ps.group_id
            WHERE ps.post_id = p.id AND gm.user_id = $2
        )
      )
) AS allowed;
