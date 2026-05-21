package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/ashulaboratory/eien/backend/db/sqlc"
)

const (
	sessionCookieName = "session_id"
	sessionDuration   = 30 * 24 * time.Hour
)

// Handler は認証関連のHTTPハンドラを束ねる
type Handler struct {
	queries *sqlc.Queries
}

func NewHandler(queries *sqlc.Queries) *Handler {
	return &Handler{queries: queries}
}

// ----------------------------------------------------------
// 共通: セッション発行
// ----------------------------------------------------------

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (h *Handler) issueSession(c echo.Context, userID uuid.UUID) error {
	sessionID, err := generateSessionID()
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(sessionDuration)

	_, err = h.queries.CreateSession(c.Request().Context(), sqlc.CreateSessionParams{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return err
	}

	c.SetCookie(&http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // TODO: 本番(HTTPS)では true に切替
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
	return nil
}

// ----------------------------------------------------------
// POST /api/auth/register
// ----------------------------------------------------------

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Birthday string `json:"birthday"`
}

type userResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Birthday string `json:"birthday,omitempty"`
}

//ハンドラ関数 hにハンドラのポインタを渡す。
func (h *Handler) Register(c echo.Context) error {
	var req registerRequest
	//リクエストボディをバインドする。受け取った構造体を書き換える必要が合うので&をつける。
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
	}
	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
	}
	birthday, err := time.Parse("2006-01-02", req.Birthday)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid birthday format (expected YYYY-MM-DD)"})
	}
	//パスワードをハッシュ化する。
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	//ユーザーを作成する。
	user, err := h.queries.CreateUser(c.Request().Context(), sqlc.CreateUserParams{
		Email:        req.Email,
		PasswordHash: string(hash),
		Birthday:     pgtype.Date{Time: birthday, Valid: true},
	})
	//エラーが発生した場合はエラーを返す。(メールアドレスが重複している場合はエラーを返す。)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c.JSON(http.StatusConflict, map[string]string{"error": "email already registered"})
		}
		log.Printf("CreateUser error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
	}

	// 登録と同時にログイン状態にする
	if err := h.issueSession(c, user.ID); err != nil {
		log.Printf("issueSession error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	//ユーザーを返す。(成功レスポンス)
	return c.JSON(http.StatusCreated, userResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Birthday: user.Birthday.Time.Format("2006-01-02"),
	})
}

// ----------------------------------------------------------
// POST /api/auth/login
// ----------------------------------------------------------

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password required"})
	}

	user, err := h.queries.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// メール存在の有無を明かさない (ユーザー列挙対策)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		}
		log.Printf("GetUserByEmail error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
	}

	if err := h.issueSession(c, user.ID); err != nil {
		log.Printf("issueSession error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	return c.JSON(http.StatusOK, userResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Birthday: user.Birthday.Time.Format("2006-01-02"),
	})
}

// ----------------------------------------------------------
// POST /api/auth/logout
// ----------------------------------------------------------

func (h *Handler) Logout(c echo.Context) error {
	cookie, err := c.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = h.queries.DeleteSession(c.Request().Context(), cookie.Value)
	}

	c.SetCookie(&http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.NoContent(http.StatusNoContent)
}

// ----------------------------------------------------------
// GET /api/auth/me (要認証)
// ----------------------------------------------------------

func (h *Handler) Me(c echo.Context) error {
	userID, ok := UserIDFromContext(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
	}

	user, err := h.queries.GetUserByID(c.Request().Context(), userID)
	if err != nil {
		log.Printf("GetUserByID error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch user"})
	}

	return c.JSON(http.StatusOK, userResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Birthday: user.Birthday.Time.Format("2006-01-02"),
	})
}
