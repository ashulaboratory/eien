# 投稿 (post) パッケージ — マイルストーンモデル

## 責務
ユーザー所有の **マイルストーン** の作成・閲覧・削除と、複数グループへの公開 (post_shares 多対多)、画像アップロード、認証付き画像配信を提供する。

## 設計の中心思想
- 1 マイルストーンは「ユーザーの記録」であり、グループには属さない (posts は author_user_id だけ持つ)。
- マイルストーンは 0〜N グループに公開できる (`post_shares` 中間テーブル)。
- 公開先 0 個 = 自分専用の記録としても残せる。
- グループ詳細での「投稿一覧」も、本実態は「そのグループにシェアされたマイルストーン」を引いている。

## エンドポイント

| Method | Path | 認証 | 認可 | 用途 |
| --- | --- | --- | --- | --- |
| POST | /api/posts | 必要 | 投稿者 | マイルストーン作成 (multipart: body + images + group_ids カンマ区切り) |
| GET | /api/timeline | 必要 | - | 統合タイムライン (デフォルト) / `?filter=mine` / `?group_id=xxx` で切替 |
| DELETE | /api/posts/:id | 必要 | 著者 | マイルストーンの論理削除 (post_shares は CASCADE で消える) |
| DELETE | /api/posts/:id/shares/:groupId | 必要 | 著者 or 該当グループのメンバー | 特定グループへの公開だけ解除 |
| GET | /api/uploads/:filename | 必要 | 著者 or シェア先グループのメンバー | 認証付き画像配信 (不許可は 404) |

## ファイル構成
- `handler.go` — 上記5エンドポイントと共通ヘルパー (asString / parseGroupIDs / isSafeFilename)
- `storage.go` — 画像ストレージ抽象 `ImageStore` + 開発用 `LocalImageStore`
- `SPEC.md` — 本ファイル

## 画像ストレージ
- **開発**: `LocalImageStore` (`uploads/` に保存、URL は `/api/uploads/{filename}`)
- **本番**: 将来 `R2ImageStore` (Cloudflare R2 + 署名付き URL) に差し替え予定
- **配信**: 認証クッキー必須 + DB で「投稿者本人 OR シェア先グループのメンバー」を判定。不許可は 404 (存在の有無を漏らさない)

## アップロード制約
| 項目 | 上限 |
| --- | --- |
| 1投稿あたりの画像数 | 4枚 |
| 1画像あたりサイズ | 10MB |
| 対応形式 | JPEG / PNG / GIF / WebP |
| 本文文字数 | 1000字 |

## トランザクション
投稿作成は `posts` INSERT + `post_images` INSERT × N + `post_shares` INSERT × M を1トランザクションで実行。

## ページネーション
- クエリ: `?limit=20&offset=0`
- limit デフォルト 20、最大 100
- timeline (3 モードすべて) で共通

## レスポンス例

### POST /api/posts
```json
{
  "id": "post-uuid",
  "body": "高校卒業して10年!",
  "images": ["http://localhost:8080/api/uploads/xxx.jpg"],
  "share_count": 2,
  "created_at": "2026-06-02T11:00:00+09:00"
}
```

### GET /api/timeline (統合)
```json
{
  "items": [
    {
      "id": "post-uuid",
      "body": "...",
      "images": ["..."],
      "created_at": "2026-...",
      "share_count": 2,
      "author": {"user_id": "...", "display_name": "たかし"}
    }
  ],
  "pagination": {"limit": 20, "offset": 0}
}
```

display_name の解決ルール (統合タイムライン):
1. 最初にシェアされたグループでの著者名
2. 0 シェアなら、著者が最初に参加したグループでの著者名 (フォールバック)

`group_id=xxx` で絞った場合は「そのグループでの著者名」が常に使われる。

## 設計判断
- N+1 クエリ: timeline では各投稿の画像取得クエリが走る。MVPでは許容、後で JSON_AGG で集約予定。
- 削除は soft delete (`deleted_at`)。画像ファイルは残るが認可で見えなくなる。
- 画像はリサイズしない (MVP)。

## 依存
- `auth` パッケージ (`UserIDFromContext`)
- `db/sqlc` (Queries, WithTx)
- `pgxpool.Pool` (トランザクション)
- `internal/post.ImageStore` (画像ストレージ)

## 関連 SQL
- `db/queries/posts.sql` — CreatePost / GetPost / SoftDeletePost / ListTimeline / ListGroupPosts / ListMyMilestones
- `db/queries/post_images.sql` — AddPostImage / ListPostImages / CanAccessPostImage
- `db/queries/post_shares.sql` — CreatePostShare / DeletePostShare

## 関連マイグレーション
- `db/migrations/000006_create_posts.up.sql`
- `db/migrations/000007_create_post_images.up.sql`
- `db/migrations/000009_milestone_refactor.up.sql` — posts.group_id を削除し post_shares を新設
