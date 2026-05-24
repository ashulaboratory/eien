package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/ashulaboratory/eien/backend/db/sqlc"
)

// contextKey は context.WithValue で使う型衝突防止のための非公開型。
// 文字列のままだとキーが他のパッケージと衝突する可能性があるため、独自型を定義する。
type contextKey string

// userIDContextKey は context に格納するときのキー
const userIDContextKey contextKey = "userID"

// Middleware は認証チェック用の Echo ミドルウェアを返す関数。
// 認証が必要なルート群 (e.Group("/api", Middleware(queries))) で適用する。
//
// 処理の流れ:
// 1. リクエストの Cookie からセッションIDを取り出す
// 2. sessions テーブルを引いて、有効なセッションか確認 (expires_at もチェック)
// 3. セッションが有効なら、user_id を context に詰めて次のハンドラへ
// 4. 無効なら 401 Unauthorized を返す
func Middleware(queries *sqlc.Queries) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Cookie からセッションIDを取得
			cookie, err := c.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" {
				// Cookie がない / 空 → 認証エラー
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "authentication required",
				})
			}

			// DB の sessions テーブルを引く
			// (GetSession の SQL 内で expires_at > NOW() 条件で期限切れを弾く)
			session, err := queries.GetSession(c.Request().Context(), cookie.Value)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					// セッションが存在しない or 期限切れ
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "session expired or invalid",
					})
				}
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "internal error",
				})
			}

			// 認証成功 → user_id を context に詰めて、次のハンドラへリクエストを渡す
			// 各ハンドラは UserIDFromContext() で user_id を取り出せる
			ctx := context.WithValue(c.Request().Context(), userIDContextKey, session.UserID)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// UserIDFromContext は middleware が context に格納した user_id を取り出す。
// 認証必須のハンドラ内で使う (例: Me, CreatePost など)。
// ok=false の場合は context に user_id が入っていない (= 認証されていない)。
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	return userID, ok
}
