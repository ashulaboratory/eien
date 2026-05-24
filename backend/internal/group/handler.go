package group

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/ashulaboratory/eien/backend/db/sqlc"
	"github.com/ashulaboratory/eien/backend/internal/auth"
)

// 招待リンクのデフォルト設定
const (
	inviteLinkDefaultDuration = 7 * 24 * time.Hour // 招待リンクの有効期限(7日)
	inviteLinkDefaultMaxUses  = 1                  // 1リンクあたりの使用回数上限(デフォルト1回)
)

// Handler はグループ関連の HTTP ハンドラを束ねる
type Handler struct {
	pool    *pgxpool.Pool  // トランザクション用に接続プール直接保持
	queries *sqlc.Queries  // sqlc 生成のクエリ実行用
}

func NewHandler(pool *pgxpool.Pool, queries *sqlc.Queries) *Handler {
	return &Handler{pool: pool, queries: queries}
}

// ----------------------------------------------------------
// 共通ヘルパー
// ----------------------------------------------------------

// generateInviteToken は URL に入る予測不能な招待トークンを生成。
// 32バイト(256bit) を crypto/rand で生成し、URL 安全な base64 に変換する。
// セッションIDと同じ思想 (math/rand は予測可能なため不可)。
func generateInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// checkMembership は指定ユーザーが指定グループのメンバーかを確認するヘルパー
func (h *Handler) checkMembership(c echo.Context, userID, groupID uuid.UUID) (bool, error) {
	return h.queries.IsGroupMember(c.Request().Context(), sqlc.IsGroupMemberParams{
		UserID:  userID,
		GroupID: groupID,
	})
}

// ----------------------------------------------------------
// POST /api/groups
// グループを新規作成(作成者は admin として自動追加)
// ----------------------------------------------------------

type createGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DisplayName string `json:"display_name"` // 作成者がこのグループ内で名乗る名前
}

type groupResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// Create はグループ新規作成のハンドラ。
// 処理の流れ:
// 1. 認証ユーザーを取得
// 2. リクエストをバリデーション (名前と display_name は必須)
// 3. トランザクションで groups + group_members(role=admin) を一括 INSERT
// 4. 作成者は自動的にそのグループの admin として追加される
func (h *Handler) Create(c echo.Context) error {
	userID, ok := auth.UserIDFromContext(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
	}

	var req createGroupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "group name is required"})
	}
	// display_name はグループごとの表示名 (Eien 固有の設計)
	if req.DisplayName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "display_name is required"})
	}

	// グループ作成とメンバー追加をトランザクションで一括実行
	ctx := c.Request().Context()
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Printf("BeginTx error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	defer tx.Rollback(ctx)

	q := h.queries.WithTx(tx)

	// groups テーブルに INSERT
	group, err := q.CreateGroup(ctx, sqlc.CreateGroupParams{
		Name:            req.Name,
		Description:     pgtype.Text{String: req.Description, Valid: true},
		CreatedByUserID: userID,
	})
	if err != nil {
		log.Printf("CreateGroup error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create group"})
	}

	// group_members テーブルに作成者を admin として追加
	_, err = q.AddGroupMember(ctx, sqlc.AddGroupMemberParams{
		UserID:      userID,
		GroupID:     group.ID,
		DisplayName: req.DisplayName,
		IconUrl:     pgtype.Text{Valid: false}, // 初期値は未設定
		Role:        "admin",
	})
	if err != nil {
		log.Printf("AddGroupMember error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to add creator to group"})
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Commit error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusCreated, groupResponse{
		ID:          group.ID.String(),
		Name:        group.Name,
		Description: group.Description.String,
		CreatedAt:   group.CreatedAt.Time.Format(time.RFC3339),
	})
}

// ----------------------------------------------------------
// GET /api/groups
// 自分が所属しているグループの一覧を取得
// ----------------------------------------------------------

func (h *Handler) ListMine(c echo.Context) error {
	userID, ok := auth.UserIDFromContext(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
	}

	// JOIN: group_members ⨝ groups で「自分が入ってる全グループ + その中での自分の情報」
	rows, err := h.queries.ListUserGroups(c.Request().Context(), userID)
	if err != nil {
		log.Printf("ListUserGroups error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list groups"})
	}

	items := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]interface{}{
			"id":              r.ID.String(),
			"name":            r.Name,
			"description":     r.Description.String,
			"my_display_name": r.MyDisplayName, // このグループでの自分の表示名
			"my_icon_url":     r.MyIconUrl.String,
			"my_role":         r.MyRole,        // admin or member
			"joined_at":       r.JoinedAt.Time.Format(time.RFC3339),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"items": items})
}

// ----------------------------------------------------------
// GET /api/groups/:id
// グループ詳細を取得 (メンバーのみ)
// ----------------------------------------------------------

func (h *Handler) Get(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group id"})
	}

	// メンバーシップ確認 (権限チェック)
	isMember, err := h.checkMembership(c, userID, groupID)
	if err != nil {
		log.Printf("IsGroupMember error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	if !isMember {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
	}

	// グループ情報を取得
	group, err := h.queries.GetGroup(c.Request().Context(), groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "group not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, groupResponse{
		ID:          group.ID.String(),
		Name:        group.Name,
		Description: group.Description.String,
		CreatedAt:   group.CreatedAt.Time.Format(time.RFC3339),
	})
}

// ----------------------------------------------------------
// GET /api/groups/:id/members
// グループのメンバー一覧を取得 (メンバーのみ閲覧可)
// ----------------------------------------------------------

func (h *Handler) ListMembers(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group id"})
	}

	// メンバーシップ確認
	isMember, err := h.checkMembership(c, userID, groupID)
	if err != nil || !isMember {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
	}

	// メンバー一覧取得
	members, err := h.queries.ListGroupMembers(c.Request().Context(), groupID)
	if err != nil {
		log.Printf("ListGroupMembers error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	items := make([]map[string]interface{}, 0, len(members))
	for _, m := range members {
		items = append(items, map[string]interface{}{
			"user_id":      m.UserID.String(),
			"display_name": m.DisplayName,
			"icon_url":     m.IconUrl.String,
			"role":         m.Role,
			"joined_at":    m.JoinedAt.Time.Format(time.RFC3339),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"items": items})
}

// ----------------------------------------------------------
// POST /api/groups/:id/invites
// 招待リンクを発行 (グループメンバーが新規メンバーを招くため)
// ----------------------------------------------------------

type createInviteResponse struct {
	Token     string `json:"token"`      // 招待URLに入るトークン
	GroupID   string `json:"group_id"`
	ExpiresAt string `json:"expires_at"` // 有効期限(7日後)
	MaxUses   int32  `json:"max_uses"`   // 最大使用回数(デフォルト1)
}

// CreateInvite は招待リンクを発行するハンドラ。
// 処理の流れ:
// 1. メンバーシップ確認 (招待を発行する権限がある)
// 2. crypto/rand で予測不能な token を生成
// 3. invite_links テーブルに INSERT
// 4. クライアントは返ってきた token で URL を組み立てて配布する
//    例: https://eien.example.com/invite/<token>
func (h *Handler) CreateInvite(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group id"})
	}

	// メンバーシップ確認
	isMember, err := h.checkMembership(c, userID, groupID)
	if err != nil || !isMember {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
	}

	// 予測不能なトークン生成
	token, err := generateInviteToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	expiresAt := time.Now().Add(inviteLinkDefaultDuration)

	// invite_links テーブルに INSERT
	invite, err := h.queries.CreateInviteLink(c.Request().Context(), sqlc.CreateInviteLinkParams{
		GroupID:         groupID,
		Token:           token,
		CreatedByUserID: userID,
		ExpiresAt:       pgtype.Timestamptz{Time: expiresAt, Valid: true},
		MaxUses:         inviteLinkDefaultMaxUses,
	})
	if err != nil {
		log.Printf("CreateInviteLink error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create invite"})
	}

	return c.JSON(http.StatusCreated, createInviteResponse{
		Token:     invite.Token,
		GroupID:   invite.GroupID.String(),
		ExpiresAt: invite.ExpiresAt.Time.Format(time.RFC3339),
		MaxUses:   invite.MaxUses,
	})
}

// ----------------------------------------------------------
// GET /api/invites/:token  (認証不要)
// 招待リンクのプレビュー情報を取得 (参加前に「どのグループへの招待か」を確認)
// ----------------------------------------------------------

type invitePreviewResponse struct {
	Token         string `json:"token"`
	GroupName     string `json:"group_name"`
	ExpiresAt     string `json:"expires_at"`
	RemainingUses int32  `json:"remaining_uses"` // max_uses - current_uses
}

// GetInvite は招待リンクのプレビューを返す (認証不要)。
// 招待された人がリンクを開いた時、参加前に「どのグループの招待か」を見られる。
func (h *Handler) GetInvite(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
	}

	// invite_links から token で検索 (SQL 内で expires_at と current_uses < max_uses もチェック)
	invite, err := h.queries.GetInviteLink(c.Request().Context(), token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 期限切れ or 使用上限到達 or 存在しない
			return c.JSON(http.StatusGone, map[string]string{"error": "invite link is invalid or expired"})
		}
		log.Printf("GetInviteLink error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// グループ情報を取得 (グループ名表示用)
	group, err := h.queries.GetGroup(c.Request().Context(), invite.GroupID)
	if err != nil {
		log.Printf("GetGroup error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, invitePreviewResponse{
		Token:         invite.Token,
		GroupName:     group.Name,
		ExpiresAt:     invite.ExpiresAt.Time.Format(time.RFC3339),
		RemainingUses: invite.MaxUses - invite.CurrentUses,
	})
}

// ----------------------------------------------------------
// POST /api/invites/:token/accept
// 招待リンクを使ってグループに参加 (認証必須)
// ----------------------------------------------------------

type acceptInviteRequest struct {
	DisplayName string `json:"display_name"` // このグループでの自分の表示名
}

// AcceptInvite は招待リンクを使ってグループに参加するハンドラ。
// 処理の流れ:
// 1. token と display_name を受け取る
// 2. トランザクションを開始
// 3. invite_links を引いて、まだ有効か確認
// 4. group_members に新規メンバーとして追加 (UNIQUE 制約で重複参加を防ぐ)
// 5. invite_links.current_uses を +1
// 6. コミット
//
// 同時に複数人が使った時の race condition は、IncrementInviteLinkUses 内で
// 条件付き UPDATE (WHERE current_uses < max_uses) で DB の行ロックを使って防ぐ。
func (h *Handler) AcceptInvite(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
	}

	var req acceptInviteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if req.DisplayName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "display_name is required"})
	}

	// メンバー追加 + 使用回数加算をトランザクションで一括処理
	ctx := c.Request().Context()
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Printf("BeginTx error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	defer tx.Rollback(ctx)

	q := h.queries.WithTx(tx)

	// 招待リンクを取得 (有効性を確認)
	invite, err := q.GetInviteLink(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusGone, map[string]string{"error": "invite link is invalid or expired"})
		}
		log.Printf("GetInviteLink error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// group_members に追加
	// UNIQUE (user_id, group_id) 制約で「同じユーザーが2回参加」を DB レベルで防ぐ
	_, err = q.AddGroupMember(ctx, sqlc.AddGroupMemberParams{
		UserID:      userID,
		GroupID:     invite.GroupID,
		DisplayName: req.DisplayName,
		IconUrl:     pgtype.Text{Valid: false},
		Role:        "member",
	})
	if err != nil {
		// 既に参加済み (UNIQUE 制約違反 = 23505) なら 409 Conflict
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c.JSON(http.StatusConflict, map[string]string{"error": "already a member of this group"})
		}
		log.Printf("AddGroupMember error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to join group"})
	}

	// 使用回数を +1 (条件付き UPDATE で race condition を防ぐ)
	if err := q.IncrementInviteLinkUses(ctx, token); err != nil {
		log.Printf("IncrementInviteLinkUses error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Commit error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"group_id": invite.GroupID.String(),
		"message":  "joined group successfully",
	})
}
