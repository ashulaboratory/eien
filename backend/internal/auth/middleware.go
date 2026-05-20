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

// contextKey は context.WithValue で使う型衝突防止のための非公開型
type contextKey string

const userIDContextKey contextKey = "userID"

// Middleware は session_id Cookie を検証してリクエストに userID を付与する。
// 認証が必要なルート群でこのミドルウェアを適用する。
func Middleware(queries *sqlc.Queries) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(sessionCookieName)
			if err != nil || cookie.Value == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "authentication required",
				})
			}

			session, err := queries.GetSession(c.Request().Context(), cookie.Value)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "session expired or invalid",
					})
				}
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "internal error",
				})
			}

			// ユーザーIDをcontextに格納
			ctx := context.WithValue(c.Request().Context(), userIDContextKey, session.UserID)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// UserIDFromContext は middleware で格納された userID を取り出す。
// 認証必須のハンドラ内で使う。
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	return userID, ok
}
