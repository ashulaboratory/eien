package handler

import (
	"errors"
	"log"
	"net/http"
    "net/mail"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/ashulaboratory/eien/backend/db/sqlc"
)

//Authは認証関連のエンドポイントを束ねるハンドラ
type Auth struct {
	queries *sqlc.Queries
}

//NerAuthは依存を注入してAuthハンドラを作成する。(依存性注入パターン)
func NewAuth(queries *sqlc.Queries) *Auth {
	return &Auth{queries: queries}
}

// ============================================
// POST /api/auth/register
// ============================================

type registerRequest struct {
	Email    string    `json:"email"`
	Password string    `json:"password"`
	Birthday string    `json:"birthday"`
}

type registerResponse struct {
	ID string `json:"id"`
	Email string `json:"email"`
	Birthday string `json:"birthday"`
}

//Registerは新規ユーザー登録を処理する。
func (a *Auth) Register(c echo.Context) error {
	// リクエストパース
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// バリデーション
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing required fields",
		})
	}
	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Password must be at least 8 characters",
		})
	}
	birthday, err := time.Parse("2006-01-02", req.Birthday)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid birthday format",
		})
	}

	//パスワードハッシュ化
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt error: %v",err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "internal error",
		})
	}

	// DB INSERT
	user, err := a.queries.CreateUser(c.Request().Context(), sqlc.CreateUserParams{
		Email: req.Email,
		PasswordHash: string(hash),
		Birthday: pgtype.Date{Time: birthday, Valid: true},
	})
	if err != nil {
		// PostgreSQLのUNIQUE違反エラー(メール重複を判定)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "email already exists",
			})
		}
		log.Printf("CreateUser error: %v",err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to create user",
		})
	}

	return c.JSON(http.StatusCreated, registerResponse{
		ID: user.ID.String(),
		Email: user.Email,
		Birthday: user.Birthday.Time.Format("2006-01-02"),
	})
}

