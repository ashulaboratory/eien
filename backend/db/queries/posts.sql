-- =====================================================
-- マイルストーン (posts) クエリ群
-- group_id を持たない設計に書き換え (post_shares で多対多関係)
-- =====================================================

-- name: CreatePost :one
-- マイルストーンを新規作成 (シェア先は別途 post_shares に追加)
INSERT INTO posts (author_user_id, body)
VALUES ($1, $2)
RETURNING *;

-- name: GetPost :one
-- 単一マイルストーン取得 (削除済みは除外)
SELECT * FROM posts
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: SoftDeletePost :exec
-- 自分の投稿のみ論理削除可能
UPDATE posts SET deleted_at = NOW()
WHERE id = $1 AND author_user_id = $2;

-- name: ListTimeline :many
-- 統合タイムライン: 自分のマイルストーン + 所属グループにシェアされた他人のマイルストーン
-- 表示名は「最初にシェアされたグループでの著者名」、
-- 0シェアの場合は「著者が最初に参加したグループでの著者名」をフォールバック
SELECT
    p.id,
    p.author_user_id,
    p.body,
    p.created_at,
    COALESCE(
        (
            SELECT gm.display_name
            FROM post_shares ps
            JOIN group_members gm
                ON gm.user_id = p.author_user_id AND gm.group_id = ps.group_id
            WHERE ps.post_id = p.id
            ORDER BY ps.shared_at ASC
            LIMIT 1
        ),
        (
            SELECT gm.display_name
            FROM group_members gm
            WHERE gm.user_id = p.author_user_id
            ORDER BY gm.joined_at ASC
            LIMIT 1
        )
    ) AS author_display_name,
    (SELECT COUNT(*) FROM post_shares WHERE post_id = p.id) AS share_count
FROM posts p
WHERE p.deleted_at IS NULL
  AND (
    p.author_user_id = $1
    OR p.id IN (
      SELECT ps2.post_id FROM post_shares ps2
      JOIN group_members gm2 ON gm2.group_id = ps2.group_id
      WHERE gm2.user_id = $1
    )
  )
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListGroupPosts :many
-- 特定グループにシェアされたマイルストーン (フィルタチップ「○○グループ」選択時)
-- 表示名はそのグループでの display_name を使う
SELECT
    p.id,
    p.author_user_id,
    p.body,
    p.created_at,
    gm.display_name AS author_display_name
FROM posts p
INNER JOIN post_shares ps ON ps.post_id = p.id AND ps.group_id = $1
INNER JOIN group_members gm
    ON gm.user_id = p.author_user_id AND gm.group_id = ps.group_id
WHERE p.deleted_at IS NULL
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListMyMilestones :many
-- 自分のマイルストーン (シェア有無問わず、フィルタチップ「マイ記録」選択時)
SELECT
    p.id,
    p.author_user_id,
    p.body,
    p.created_at,
    (
        SELECT gm.display_name
        FROM group_members gm
        WHERE gm.user_id = p.author_user_id
        ORDER BY gm.joined_at ASC
        LIMIT 1
    ) AS author_display_name,
    (SELECT COUNT(*) FROM post_shares WHERE post_id = p.id) AS share_count
FROM posts p
WHERE p.deleted_at IS NULL AND p.author_user_id = $1
ORDER BY p.created_at DESC
LIMIT $2 OFFSET $3;