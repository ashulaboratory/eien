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

const (
	inviteLinkDefaultDuration = 7 * 24 * time.Hour
	inviteLinkDefaultMaxUses  = 1
)

// Handler はグループ関連のHTTPハンドラを束ねる
type Handler struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewHandler(pool *pgxpool.Pool, queries *sqlc.Queries) *Handler {
	return &Handler{pool: pool, queries: queries}
}

// ----------------------------------------------------------
// 共通: 招待トークン生成
// ----------------------------------------------------------

func generateInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// メンバーシップ確認のヘルパー
func (h *Handler) checkMembership(c echo.Context, userID, groupID uuid.UUID) (bool, error) {
	return h.queries.IsGroupMember(c.Request().Context(), sqlc.IsGroupMemberParams{
		UserID:  userID,
		GroupID: groupID,
	})
}

// ----------------------------------------------------------
// POST /api/groups
// ----------------------------------------------------------

type createGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	DisplayName string `json:"display_name"`
}

type groupResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

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
	if req.DisplayName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "display_name is required"})
	}

	ctx := c.Request().Context()
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Printf("BeginTx error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	defer tx.Rollback(ctx)

	q := h.queries.WithTx(tx)

	// グループ作成
	group, err := q.CreateGroup(ctx, sqlc.CreateGroupParams{
		Name:            req.Name,
		Description:     pgtype.Text{String: req.Description, Valid: true},
		CreatedByUserID: userID,
	})
	if err != nil {
		log.Printf("CreateGroup error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create group"})
	}

	// 作成者を admin として追加
	_, err = q.AddGroupMember(ctx, sqlc.AddGroupMemberParams{
		UserID:      userID,
		GroupID:     group.ID,
		DisplayName: req.DisplayName,
		IconUrl:     pgtype.Text{Valid: false},
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
// ----------------------------------------------------------

func (h *Handler) ListMine(c echo.Context) error {
	userID, ok := auth.UserIDFromContext(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
	}

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
			"my_display_name": r.MyDisplayName,
			"my_icon_url":     r.MyIconUrl.String,
			"my_role":         r.MyRole,
			"joined_at":       r.JoinedAt.Time.Format(time.RFC3339),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"items": items})
}

// ----------------------------------------------------------
// GET /api/groups/:id
// ----------------------------------------------------------

func (h *Handler) Get(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group id"})
	}

	isMember, err := h.checkMembership(c, userID, groupID)
	if err != nil {
		log.Printf("IsGroupMember error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	if !isMember {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
	}

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
// ----------------------------------------------------------

func (h *Handler) ListMembers(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group id"})
	}

	isMember, err := h.checkMembership(c, userID, groupID)
	if err != nil || !isMember {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
	}

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
// ----------------------------------------------------------

type createInviteResponse struct {
	Token     string `json:"token"`
	GroupID   string `json:"group_id"`
	ExpiresAt string `json:"expires_at"`
	MaxUses   int32  `json:"max_uses"`
}

func (h *Handler) CreateInvite(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group id"})
	}

	isMember, err := h.checkMembership(c, userID, groupID)
	if err != nil || !isMember {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
	}

	token, err := generateInviteToken()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	expiresAt := time.Now().Add(inviteLinkDefaultDuration)

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
// GET /api/invites/:token  (認証不要、参加前のプレビュー用)
// ----------------------------------------------------------

type invitePreviewResponse struct {
	Token         string `json:"token"`
	GroupName     string `json:"group_name"`
	ExpiresAt     string `json:"expires_at"`
	RemainingUses int32  `json:"remaining_uses"`
}

func (h *Handler) GetInvite(c echo.Context) error {
	token := c.Param("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token required"})
	}

	invite, err := h.queries.GetInviteLink(c.Request().Context(), token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusGone, map[string]string{"error": "invite link is invalid or expired"})
		}
		log.Printf("GetInviteLink error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

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
// ----------------------------------------------------------

type acceptInviteRequest struct {
	DisplayName string `json:"display_name"`
}

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

	ctx := c.Request().Context()
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Printf("BeginTx error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	defer tx.Rollback(ctx)

	q := h.queries.WithTx(tx)

	// 招待リンク検証
	invite, err := q.GetInviteLink(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusGone, map[string]string{"error": "invite link is invalid or expired"})
		}
		log.Printf("GetInviteLink error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// グループメンバーに追加
	_, err = q.AddGroupMember(ctx, sqlc.AddGroupMemberParams{
		UserID:      userID,
		GroupID:     invite.GroupID,
		DisplayName: req.DisplayName,
		IconUrl:     pgtype.Text{Valid: false},
		Role:        "member",
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return c.JSON(http.StatusConflict, map[string]string{"error": "already a member of this group"})
		}
		log.Printf("AddGroupMember error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to join group"})
	}

	// 使用回数 +1
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
