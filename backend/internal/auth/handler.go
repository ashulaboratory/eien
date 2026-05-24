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

// 認証セッションに関する定数
const (
	sessionCookieName = "session_id"        // Cookie に保存するセッションIDの名前
	sessionDuration   = 30 * 24 * time.Hour // セッションの有効期限(30日)
)

// Handler は認証関連のHTTPハンドラを束ねる構造体
type Handler struct {
	queries *sqlc.Queries // sqlc が生成した DB クエリ実行用の構造体
}

// NewHandler は Handler のコンストラクタ(main.go から呼ばれる)
func NewHandler(queries *sqlc.Queries) *Handler {
	return &Handler{queries: queries}
}

// ----------------------------------------------------------
// 共通: セッション発行
// ----------------------------------------------------------

// generateSessionID は暗号学的に予測不能なセッションIDを生成する。
// crypto/rand で 32バイト(256bit)のランダム値を作り、URL安全な base64 に変換。
// math/rand は擬似乱数で予測可能なため、秘密値の生成には使えない。
func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// issueSession は新しいセッションを発行してブラウザに Cookie をセットする。
// 1. ランダムなセッションIDを生成
// 2. sessions テーブルに INSERT
// 3. HttpOnly + SameSite=Lax の Cookie として返す
func (h *Handler) issueSession(c echo.Context, userID uuid.UUID) error {
	// セッションIDを生成
	sessionID, err := generateSessionID()
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(sessionDuration)

	// DB の sessions テーブルにレコードを挿入
	_, err = h.queries.CreateSession(c.Request().Context(), sqlc.CreateSessionParams{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return err
	}

	// セッションIDを Cookie としてブラウザにセット
	// HttpOnly: JavaScript からアクセス不可 (XSS でセッション盗難を防ぐ)
	// SameSite=Lax: 別サイト経由の POST で Cookie を送らない (CSRF 対策)
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

// registerRequest はリクエストボディの JSON をデコードする型
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Birthday string `json:"birthday"`
}

// userResponse はクライアントに返すユーザー情報の JSON 型
type userResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Birthday string `json:"birthday,omitempty"`
}

// Register は新規ユーザー登録のハンドラ。
// 1. リクエストボディをバインド
// 2. 入力をバリデーション (メール形式・パスワード長・誕生日形式)
// 3. パスワードを bcrypt でハッシュ化
// 4. users テーブルに INSERT (メール重複は 23505 で検知)
// 5. セッションを発行して自動ログイン
func (h *Handler) Register(c echo.Context) error {
	// リクエストボディを registerRequest 構造体にデコード
	// & で構造体のアドレスを渡し、c.Bind が中身を書き換える
	var req registerRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// メールアドレスの形式チェック
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid email format"})
	}

	// パスワード長チェック(8文字以上)
	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
	}

	// 誕生日を YYYY-MM-DD 形式でパース ("2006-01-02" は Go のリファレンス時刻)
	birthday, err := time.Parse("2006-01-02", req.Birthday)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid birthday format (expected YYYY-MM-DD)"})
	}

	// パスワードを bcrypt でハッシュ化
	// ソルト生成・パラメータ保存・timing attack 対策は全部ライブラリ内で処理される
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// users テーブルに新規レコードを INSERT
	user, err := h.queries.CreateUser(c.Request().Context(), sqlc.CreateUserParams{
		Email:        req.Email,
		PasswordHash: string(hash),
		Birthday:     pgtype.Date{Time: birthday, Valid: true},
	})
	// メールアドレス重複は PostgreSQL の UNIQUE 制約違反 (エラーコード 23505) で検知
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c.JSON(http.StatusConflict, map[string]string{"error": "email already registered"})
		}
		log.Printf("CreateUser error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
	}

	// 登録成功時、セッションを発行して自動ログイン状態にする
	if err := h.issueSession(c, user.ID); err != nil {
		log.Printf("issueSession error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	// 201 Created でユーザー情報を返す
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

// Login はログインのハンドラ。
// 1. メール / パスワードを受け取る
// 2. メールでユーザーを引く
// 3. bcrypt でパスワードを照合 (timing attack 対策はライブラリ内で行う)
// 4. セッションを発行して Cookie をセット
func (h *Handler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email and password required"})
	}

	// メールアドレスでユーザーを検索
	user, err := h.queries.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		// メール存在の有無を明かさない (ユーザー列挙対策)
		// → 存在しないメールでも "invalid email or password" を返す
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		}
		log.Printf("GetUserByEmail error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// bcrypt でパスワードハッシュを照合
	// 内部で定数時間比較が行われ、timing attack を防ぐ
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
	}

	// セッション発行 → Cookie セット
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

// Logout はログアウト処理。
// 1. Cookie からセッションIDを取得
// 2. DB の sessions テーブルから該当セッションを DELETE (強制無効化)
// 3. ブラウザの Cookie を空にして MaxAge=-1 で即座に削除指示
func (h *Handler) Logout(c echo.Context) error {
	// Cookie からセッションIDを取得
	cookie, err := c.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		// DB から該当セッションを DELETE (失敗しても無視: _ で受け流す)
		_ = h.queries.DeleteSession(c.Request().Context(), cookie.Value)
	}

	// ブラウザ側の Cookie も削除
	c.SetCookie(&http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1, // 即座に削除を指示
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.NoContent(http.StatusNoContent)
}

// ----------------------------------------------------------
// GET /api/auth/me (要認証)
// ----------------------------------------------------------

// Me は現在ログイン中のユーザー情報を返すハンドラ。
// 認証ミドルウェアが context に user_id を設定済みの前提で動く。
func (h *Handler) Me(c echo.Context) error {
	// context から認証ミドルウェアが設定した user_id を取り出す
	userID, ok := UserIDFromContext(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
	}

	// user_id でユーザーを検索
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
