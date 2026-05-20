# 投稿 (post) パッケージ

## 責務
投稿の作成・閲覧・削除と、画像アップロードを提供する。タイムライン (自分の全所属グループの投稿) も。

## エンドポイント

| Method | Path | 認証 | 認可 | 用途 |
| --- | --- | --- | --- | --- |
| POST | /api/groups/:id/posts | 必要 | メンバー | 投稿作成（multipart: body + 画像 0〜4枚）|
| GET | /api/timeline | 必要 | - | 自分の全所属グループの投稿を時系列で取得 |
| GET | /api/groups/:id/posts | 必要 | メンバー | グループ別投稿一覧 |
| GET | /api/posts/:id | 必要 | メンバー | 投稿詳細 |
| DELETE | /api/posts/:id | 必要 | 著者 | 投稿削除（soft delete） |

## ファイル構成
- `handler.go` - 5つのエンドポイント
- `storage.go` - 画像ストレージの抽象 (`ImageStore`) と `LocalImageStore` 実装
- `SPEC.md` - 本ファイル

## 画像ストレージ
- **開発**: `LocalImageStore`（`uploads/` ディレクトリに保存、`/uploads/{filename}` で配信）
- **本番**: 将来 `R2ImageStore` を実装予定（Cloudflare R2）。`ImageStore` インターフェースに沿って差し替え可能。

## アップロード制約
| 項目 | 上限 |
| --- | --- |
| 1投稿あたりの画像数 | 4枚 |
| 1画像あたりサイズ | 10MB |
| 対応形式 | JPEG / PNG / GIF / WebP |
| 本文文字数 | 1000字 |

## トランザクション
投稿作成は posts INSERT + post_images INSERT × N をトランザクションで包む。途中で失敗したら全部ロールバック。

## ページネーション
- クエリ: `?limit=20&offset=0`
- limit デフォルト 20、最大 100
- timeline / groups/:id/posts 両方で利用可能

## レスポンス例

### POST /api/groups/:id/posts
```json
{
  "id": "post-uuid",
  "group_id": "group-uuid",
  "body": "結婚しました！",
  "images": ["http://localhost:8080/uploads/xxx.jpg"],
  "created_at": "2026-05-16T11:00:00+09:00"
}
```

### GET /api/timeline
```json
{
  "items": [
    {
      "id": "post-uuid",
      "body": "...",
      "images": ["..."],
      "created_at": "2026-...",
      "group": {"id": "...", "name": "高校時代の仲間"},
      "author": {"user_id": "...", "display_name": "たかし", "icon_url": ""}
    }
  ],
  "pagination": {"limit": 20, "offset": 0}
}
```

## 設計判断
- N+1 クエリ問題: timeline では投稿数だけ画像取得クエリが走る。MVPでは許容、後で JSON_AGG で1クエリ化を検討。
- 削除は soft delete（`deleted_at` 更新）。後で物理削除バッチを検討。
- 画像はリサイズしない（現状）。アップロード時間が長くなりそうなら追加検討。

## 依存
- `auth` パッケージ (`UserIDFromContext`)
- `db/sqlc` (Queries, WithTx)
- `pgxpool.Pool`（トランザクション）
- `internal/post.ImageStore`（画像ストレージ）

## 関連SQL
- `db/queries/posts.sql` - CreatePost, GetPost, SoftDeletePost, ListGroupPosts, ListTimeline
- `db/queries/post_images.sql` - AddPostImage, ListPostImages

## 関連マイグレーション
- `db/migrations/000006_create_posts.up.sql`
- `db/migrations/000007_create_post_images.up.sql`
