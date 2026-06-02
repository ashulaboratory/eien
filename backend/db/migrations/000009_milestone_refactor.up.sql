-- =====================================================
-- マイルストーン化リファクタ
-- - posts.group_id を削除（投稿はユーザーの所有物に）
-- - post_shares 中間テーブル新規作成（多対多）
-- - 既存 posts は全削除（テストデータのみのため）
-- =====================================================

-- 1. 既存 posts データを全削除（テストデータ前提）
DELETE FROM posts;

-- 2. posts.group_id を削除する前に、関連インデックスを削除
DROP INDEX IF EXISTS idx_posts_group_id_created_at;

-- 3. posts.group_id カラム削除
ALTER TABLE posts DROP COLUMN group_id;

-- 4. タイムライン用のインデックス追加（created_at 単独）
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);

-- 5. 中間テーブル post_shares を作成
CREATE TABLE post_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    shared_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (post_id, group_id)
);

-- 6. post_shares のインデックス
CREATE INDEX idx_post_shares_post_id ON post_shares(post_id);
CREATE INDEX idx_post_shares_group_id_shared_at ON post_shares(group_id, shared_at DESC);