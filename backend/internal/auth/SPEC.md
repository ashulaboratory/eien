# 認証 (auth) パッケージ

## 責務
ユーザー認証 (register / login / logout) と、認証必須ルート用のミドルウェアを提供する。

## エンドポイント

| Method | Path | 認証 | 用途 |
| --- | --- | --- | --- |
| POST | /api/auth/register | 不要 | 新規ユーザー登録（成功時に自動でログイン状態にする） |
| POST | /api/auth/login | 不要 | ログイン → Set-Cookie で session_id 発行 |
| POST | /api/auth/logout | 不要 | サーバ側セッション削除 + Cookie 失効 |
| GET | /api/auth/me | **必要** | 現在ログイン中のユーザー情報を返す |

## ファイル構成
- `handler.go` - 4つのエンドポイント実装 + セッション発行ヘルパー
- `middleware.go` - 認証ミドルウェアと `UserIDFromContext` ヘルパー
- `SPEC.md` - 本ファイル

## 認証方式
- **セッションCookie 方式** (HttpOnly, SameSite=Lax)
- セッションIDは CSPRNG 32バイト = 256bit
- セッションは `sessions` テーブルに保存（サーバ側で取り消し可能）
- セッション有効期限: 30日（固定期限、更新型ではない）

## セキュリティ要点
- パスワードは bcrypt (DefaultCost=10) でハッシュ化
- ログイン失敗時はメール存在の有無を明かさない（ユーザー列挙対策）
- Cookie `Secure` フラグは本番(HTTPS)で true、開発で false ※ハンドラのTODOコメント箇所で切替
- セッションIDが漏れても他人になりすませない（HttpOnly でJS読取不可、SameSite=Lax でCSRF軽減）

## 依存
- `db/sqlc` - DB操作（Queries）
- `bcrypt`, `pgx`, `pgconn`, `pgtype`, `google/uuid`, `echo`

## 認証必須ルートでの使い方
```go
authenticated := e.Group("/api", auth.Middleware(queries))
authenticated.GET("/auth/me", authHandler.Me)
// 他の認証必須エンドポイントもここに追加
```

ハンドラ内でユーザーIDを取り出す：
```go
userID, ok := auth.UserIDFromContext(c.Request().Context())
if !ok {
    return c.JSON(http.StatusUnauthorized, ...)
}
```

## 関連SQL
- `db/queries/users.sql` - CreateUser, GetUserByEmail, GetUserByID
- `db/queries/sessions.sql` - CreateSession, GetSession, DeleteSession, DeleteExpiredSessions
