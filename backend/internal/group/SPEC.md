# グループ (group) パッケージ

## 責務
グループの作成・閲覧、メンバー管理、招待リンク発行・受諾を提供する。

## エンドポイント

| Method | Path | 認証 | 認可 | 用途 |
| --- | --- | --- | --- | --- |
| POST | /api/groups | 必要 | - | 新規グループ作成（作成者は admin で自動追加） |
| GET | /api/groups | 必要 | - | 自分が所属するグループ一覧 |
| GET | /api/groups/:id | 必要 | メンバー | グループ詳細 |
| GET | /api/groups/:id/members | 必要 | メンバー | メンバー一覧 |
| POST | /api/groups/:id/invites | 必要 | メンバー | 招待リンク発行（7日有効、1回限り） |
| GET | /api/invites/:token | **不要** | - | 招待リンクのプレビュー（未登録者でも見られる） |
| POST | /api/invites/:token/accept | 必要 | - | 招待リンクを使ってグループに参加 |

## ファイル構成
- `handler.go` - 全エンドポイント実装 + 招待トークン生成 + メンバーシップ確認ヘルパー
- `SPEC.md` - 本ファイル

## トランザクション使用箇所
データ整合性のためDBトランザクションで包む処理：

1. **グループ作成 (Create)**
   - groups INSERT → group_members INSERT (作成者を admin で追加)
   - 途中で失敗したら両方ロールバック

2. **招待受諾 (AcceptInvite)**
   - 招待リンク検証 → group_members INSERT → invite_links UPDATE (current_uses++)
   - 途中で失敗したら参加もカウントもなかったことに

## 招待リンクのデフォルト
- 有効期限: 7日
- 最大使用回数: 1回
- トークン: CSPRNG 32バイト = base64 URL-safe 文字列

将来的にAPIで `expires_at` / `max_uses` を指定可能にする予定。

## 認可ルール
- グループ詳細・メンバー一覧・招待発行: そのグループのメンバーのみ (`IsGroupMember` で確認)
- 招待リンクプレビュー: 認証不要（リンクを開く人が未登録の可能性があるため）
- 招待受諾: ログイン済みユーザーが対象

## レスポンス例

### GET /api/groups （自分の所属グループ一覧）
```json
{
  "items": [
    {
      "id": "uuid",
      "name": "高校時代の仲間",
      "description": "...",
      "my_display_name": "たかし",
      "my_icon_url": "",
      "my_role": "admin",
      "joined_at": "2026-05-16T11:00:00+09:00"
    }
  ]
}
```

### POST /api/groups/:id/invites （招待発行）
```json
{
  "token": "Y65IsjBndh1_ddnZd44XSvfVKHzOMmEU55MPI1fMMEY=",
  "group_id": "uuid",
  "expires_at": "2026-05-23T11:00:00+09:00",
  "max_uses": 1
}
```

→ フロントは `https://eien.fly.dev/invite/{token}` の形でURL組み立て。

## 依存
- `auth` パッケージ (`Middleware`, `UserIDFromContext`)
- `db/sqlc` (Queries, WithTx)
- `pgxpool.Pool` (トランザクション開始用)

## 関連SQL
- `db/queries/groups.sql` - CreateGroup, GetGroup, ListUserGroups
- `db/queries/group_members.sql` - AddGroupMember, ListGroupMembers, GetGroupMember, IsGroupMember, UpdateGroupMemberProfile
- `db/queries/invite_links.sql` - CreateInviteLink, GetInviteLink, IncrementInviteLinkUses, DeleteExpiredInviteLinks
