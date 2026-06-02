// post handler はマイルストーン (投稿) のHTTPハンドラ。
// 多対多モデル: 1 マイルストーンが N グループに公開可能 (post_shares 中間テーブル)。
// 公開先0個も許可 = 自分専用の記録として保持できる。
package post

import (
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/ashulaboratory/eien/backend/db/sqlc"
	"github.com/ashulaboratory/eien/backend/internal/auth"
)

const (
	maxBodyLength = 1000
	maxImageCount = 4
	maxImageSize  = 10 * 1024 * 1024
)

type Handler struct {
	pool      *pgxpool.Pool
	queries   *sqlc.Queries
	images    ImageStore
	uploadDir string // 認証付き画像配信時に c.File で参照する物理パス
}

func NewHandler(pool *pgxpool.Pool, queries *sqlc.Queries, images ImageStore, uploadDir string) *Handler {
	return &Handler{pool: pool, queries: queries, images: images, uploadDir: uploadDir}
}

// ============================================================
// ヘルパー
// ============================================================

func isValidImageContentType(ct string) bool {
	switch ct {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	}
	return false
}

func parseLimitOffset(c echo.Context, defaultLimit int) (int32, int32) {
	limit := defaultLimit
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	return int32(limit), int32(offset)
}

// asString は sqlc が COALESCE 等で interface{} で返してくる値を安全に string にキャストする。
// 失敗時は空文字列 (UI 側で「不明」等を表示する想定)。
func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func (h *Handler) loadImageURLs(c echo.Context, postID uuid.UUID) []string {
	images, err := h.queries.ListPostImages(c.Request().Context(), postID)
	if err != nil {
		log.Printf("ListPostImages error: %v", err)
		return []string{}
	}
	urls := make([]string, 0, len(images))
	for _, img := range images {
		urls = append(urls, img.ImageUrl)
	}
	return urls
}

// parseGroupIDs はカンマ区切りの group_ids form 値を UUID 配列にパースする。
// 例: "uuid1,uuid2,uuid3"  →  []uuid.UUID{...}
func parseGroupIDs(c echo.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	raw := c.FormValue("group_ids")
	if raw == "" {
		return ids, nil // 0個もOK (自分専用マイルストーン)
	}
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		gid, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		ids = append(ids, gid)
	}
	return ids, nil
}

// ============================================================
// POST /api/posts
// multipart form: body, images[] (任意), group_ids (カンマ区切り, 任意)
// ============================================================

func (h *Handler) Create(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())
	ctx := c.Request().Context()

	// 本文バリデーション
	body := c.FormValue("body")
	if body == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "body is required"})
	}
	if len(body) > maxBodyLength {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "body too long (max 1000)"})
	}

	// 公開先グループのパース + メンバーシップ検証
	groupIDs, err := parseGroupIDs(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group_id"})
	}
	for _, gid := range groupIDs {
		isMember, err := h.queries.IsGroupMember(ctx, sqlc.IsGroupMemberParams{
			UserID:  userID,
			GroupID: gid,
		})
		if err != nil {
			log.Printf("IsGroupMember error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
		if !isMember {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "you are not a member of one of the selected groups",
			})
		}
	}

	// 画像処理 (R2移行までローカル保存)
	var imageURLs []string
	if form, _ := c.MultipartForm(); form != nil {
		files := form.File["images"]
		if len(files) > maxImageCount {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "too many images (max 4)"})
		}
		for _, fh := range files {
			if fh.Size > maxImageSize {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "image too large (max 10MB)"})
			}
			ct := fh.Header.Get("Content-Type")
			if !isValidImageContentType(ct) {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "unsupported image type (allowed: jpeg, png, gif, webp)",
				})
			}
			src, err := fh.Open()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read image"})
			}
			url, err := h.images.Save(ctx, src, ct)
			src.Close()
			if err != nil {
				log.Printf("image save error: %v", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store image"})
			}
			imageURLs = append(imageURLs, url)
		}
	}

	// DB トランザクション: posts + post_images + post_shares を1セットで
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Printf("BeginTx error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	defer tx.Rollback(ctx)
	q := h.queries.WithTx(tx)

	post, err := q.CreatePost(ctx, sqlc.CreatePostParams{
		AuthorUserID: userID,
		Body:         body,
	})
	if err != nil {
		log.Printf("CreatePost error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create post"})
	}

	for i, url := range imageURLs {
		if _, err := q.AddPostImage(ctx, sqlc.AddPostImageParams{
			PostID:    post.ID,
			ImageUrl:  url,
			SortOrder: int32(i),
		}); err != nil {
			log.Printf("AddPostImage error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to attach image"})
		}
	}

	for _, gid := range groupIDs {
		if _, err := q.CreatePostShare(ctx, sqlc.CreatePostShareParams{
			PostID:  post.ID,
			GroupID: gid,
		}); err != nil {
			log.Printf("CreatePostShare error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to share"})
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Commit error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":          post.ID.String(),
		"body":        post.Body,
		"images":      imageURLs,
		"share_count": len(groupIDs),
		"created_at":  post.CreatedAt.Time.Format(time.RFC3339),
	})
}

// ============================================================
// GET /api/timeline
// Query params で 3 つのモードを切り替える:
//   - group_id=xxx : 特定グループにシェアされたマイルストーン
//   - filter=mine  : 自分のマイルストーンのみ
//   - 上記なし     : 統合タイムライン (自分のもの + 所属グループへのシェア)
// ============================================================

func (h *Handler) Timeline(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())
	ctx := c.Request().Context()
	limit, offset := parseLimitOffset(c, 20)

	// 1) group_id フィルタ
	if gidStr := c.QueryParam("group_id"); gidStr != "" {
		groupID, err := uuid.Parse(gidStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group_id"})
		}
		isMember, err := h.queries.IsGroupMember(ctx, sqlc.IsGroupMemberParams{
			UserID:  userID,
			GroupID: groupID,
		})
		if err != nil || !isMember {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
		}
		return h.respondGroupTimeline(c, groupID, limit, offset)
	}

	// 2) mine フィルタ
	if c.QueryParam("filter") == "mine" {
		return h.respondMyMilestones(c, userID, limit, offset)
	}

	// 3) 統合タイムライン (デフォルト)
	rows, err := h.queries.ListTimeline(ctx, sqlc.ListTimelineParams{
		AuthorUserID: userID,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		log.Printf("ListTimeline error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch timeline"})
	}

	items := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]interface{}{
			"id":          r.ID.String(),
			"body":        r.Body,
			"images":      h.loadImageURLs(c, r.ID),
			"created_at":  r.CreatedAt.Time.Format(time.RFC3339),
			"share_count": r.ShareCount,
			"author": map[string]string{
				"user_id":      r.AuthorUserID.String(),
				"display_name": asString(r.AuthorDisplayName), // interface{} → string
			},
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"items":      items,
		"pagination": map[string]int32{"limit": limit, "offset": offset},
	})
}

func (h *Handler) respondGroupTimeline(c echo.Context, groupID uuid.UUID, limit, offset int32) error {
	rows, err := h.queries.ListGroupPosts(c.Request().Context(), sqlc.ListGroupPostsParams{
		GroupID: groupID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		log.Printf("ListGroupPosts error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch"})
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]interface{}{
			"id":         r.ID.String(),
			"body":       r.Body,
			"images":     h.loadImageURLs(c, r.ID),
			"created_at": r.CreatedAt.Time.Format(time.RFC3339),
			"author": map[string]string{
				"user_id":      r.AuthorUserID.String(),
				"display_name": r.AuthorDisplayName,
			},
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"items":      items,
		"pagination": map[string]int32{"limit": limit, "offset": offset},
	})
}

func (h *Handler) respondMyMilestones(c echo.Context, userID uuid.UUID, limit, offset int32) error {
	rows, err := h.queries.ListMyMilestones(c.Request().Context(), sqlc.ListMyMilestonesParams{
		AuthorUserID: userID,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		log.Printf("ListMyMilestones error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch"})
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		items = append(items, map[string]interface{}{
			"id":          r.ID.String(),
			"body":        r.Body,
			"images":     h.loadImageURLs(c, r.ID),
			"created_at":  r.CreatedAt.Time.Format(time.RFC3339),
			"share_count": r.ShareCount,
			"author": map[string]string{
				"user_id":      r.AuthorUserID.String(),
				"display_name": r.AuthorDisplayName,
			},
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"items":      items,
		"pagination": map[string]int32{"limit": limit, "offset": offset},
	})
}

// ============================================================
// DELETE /api/posts/:id
// 自分のマイルストーンを論理削除 (post_shares は CASCADE で自動削除)
// ============================================================

func (h *Handler) Delete(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}
	if err := h.queries.SoftDeletePost(c.Request().Context(), sqlc.SoftDeletePostParams{
		ID:           postID,
		AuthorUserID: userID,
	}); err != nil {
		log.Printf("SoftDeletePost error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	return c.NoContent(http.StatusNoContent)
}

// ============================================================
// DELETE /api/posts/:id/shares/:groupId
// 特定グループへの公開だけを解除 (他グループへの公開は維持、本体は残る)
// 著者本人 or そのグループのメンバー が実行可能
// ============================================================

func (h *Handler) Unshare(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())
	ctx := c.Request().Context()

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid post id"})
	}
	groupID, err := uuid.Parse(c.Param("groupId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group id"})
	}

	post, err := h.queries.GetPost(ctx, postID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "post not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal"})
	}
	if post.AuthorUserID != userID {
		// 著者本人でなければそのグループのメンバーである必要がある
		isMember, _ := h.queries.IsGroupMember(ctx, sqlc.IsGroupMemberParams{
			UserID:  userID,
			GroupID: groupID,
		})
		if !isMember {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not allowed"})
		}
	}

	if err := h.queries.DeletePostShare(ctx, sqlc.DeletePostShareParams{
		PostID:  postID,
		GroupID: groupID,
	}); err != nil {
		log.Printf("DeletePostShare error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal"})
	}
	return c.NoContent(http.StatusNoContent)
}

// ============================================================
// GET /api/uploads/:filename
// 認証必須 + 認可チェック付きの画像配信。
// 認可: 投稿者本人 OR シェア先グループのメンバー のみダウンロード可。
// ============================================================

// isSafeFilename はパストラバーサル攻撃を弾くためのバリデーション。
// 許可: UUID + 拡張子の形式 (例: "abc-def-...-xyz.jpg")
// 禁止: "/", "\", "..", 空文字列
func isSafeFilename(name string) bool {
	if name == "" {
		return false
	}
	if strings.ContainsAny(name, `/\`) {
		return false
	}
	if strings.Contains(name, "..") {
		return false
	}
	return true
}

func (h *Handler) ServeImage(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())
	filename := c.Param("filename")

	// パストラバーサル対策
	if !isSafeFilename(filename) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid filename"})
	}

	// アクセス権チェック
	allowed, err := h.queries.CanAccessPostImage(c.Request().Context(), sqlc.CanAccessPostImageParams{
		Filename: filename,
		UserID:   userID,
	})
	if err != nil {
		log.Printf("CanAccessPostImage error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal"})
	}
	if !allowed {
		// 存在の有無を漏らさないため 404 を返す (権限なしも未存在も同じ応答)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	// ファイル配信
	fullPath := filepath.Join(h.uploadDir, filename)
	return c.File(fullPath)
}