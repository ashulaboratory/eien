-- 逆方向: post_shares 削除 + posts.group_id 復活
-- 注意: 既存データは失われる（テスト用途のみ）

DROP TABLE IF EXISTS post_shares;

DROP INDEX IF EXISTS idx_posts_created_at;

ALTER TABLE posts ADD COLUMN group_id UUID REFERENCES groups(id);

CREATE INDEX idx_posts_group_id_created_at ON posts(group_id, created_at DESC);