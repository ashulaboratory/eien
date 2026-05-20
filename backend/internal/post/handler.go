package post

import (
	"errors"
	"log"
	"net/http"
	"strconv"
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
	maxImageSize  = 10 * 1024 * 1024 // 10MB
)

// Handler は投稿関連のHTTPハンドラ
type Handler struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	images  ImageStore
}

func NewHandler(pool *pgxpool.Pool, queries *sqlc.Queries, images ImageStore) *Handler {
	return &Handler{pool: pool, queries: queries, images: images}
}

// ----------------------------------------------------------
// 共通ヘルパー
// ----------------------------------------------------------

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

// ----------------------------------------------------------
// POST /api/groups/:id/posts
// multipart form: body (text) + images (file[], optional)
// ----------------------------------------------------------

func (h *Handler) Create(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group id"})
	}

	// メンバーシップ確認
	isMember, err := h.queries.IsGroupMember(c.Request().Context(), sqlc.IsGroupMemberParams{
		UserID:  userID,
		GroupID: groupID,
	})
	if err != nil || !isMember {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
	}

	// 本文
	body := c.FormValue("body")
	if body == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "body is required"})
	}
	if len(body) > maxBodyLength {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "body too long (max 1000 chars)"})
	}

	// 画像アップロード処理 (ある場合のみ)
	ctx := c.Request().Context()
	var imageURLs []string

	form, _ := c.MultipartForm()
	if form != nil {
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
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "unsupported image type (allowed: jpeg, png, gif, webp)"})
			}
			src, err := fh.Open()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read uploaded file"})
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

	// DB INSERT (post + post_images をトランザクションで)
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Printf("BeginTx error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	defer tx.Rollback(ctx)

	q := h.queries.WithTx(tx)

	post, err := q.CreatePost(ctx, sqlc.CreatePostParams{
		GroupID:      groupID,
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

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Commit error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":         post.ID.String(),
		"group_id":   post.GroupID.String(),
		"body":       post.Body,
		"images":     imageURLs,
		"created_at": post.CreatedAt.Time.Format(time.RFC3339),
	})
}

// ----------------------------------------------------------
// GET /api/timeline
// ----------------------------------------------------------

func (h *Handler) Timeline(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	limit, offset := parseLimitOffset(c, 20)

	rows, err := h.queries.ListTimeline(c.Request().Context(), sqlc.ListTimelineParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		log.Printf("ListTimeline error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch timeline"})
	}

	items := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		images := h.loadImageURLs(c, r.ID)
		items = append(items, map[string]interface{}{
			"id":         r.ID.String(),
			"body":       r.Body,
			"images":     images,
			"created_at": r.CreatedAt.Time.Format(time.RFC3339),
			"group": map[string]string{
				"id":   r.GroupID.String(),
				"name": r.GroupName,
			},
			"author": map[string]interface{}{
				"user_id":      r.AuthorUserID.String(),
				"display_name": r.AuthorDisplayName,
				"icon_url":     r.AuthorIconUrl.String,
			},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
		"pagination": map[string]int32{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// ----------------------------------------------------------
// GET /api/groups/:id/posts
// ----------------------------------------------------------

func (h *Handler) ListByGroup(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group id"})
	}

	isMember, err := h.queries.IsGroupMember(c.Request().Context(), sqlc.IsGroupMemberParams{
		UserID:  userID,
		GroupID: groupID,
	})
	if err != nil || !isMember {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
	}

	limit, offset := parseLimitOffset(c, 20)

	rows, err := h.queries.ListGroupPosts(c.Request().Context(), sqlc.ListGroupPostsParams{
		GroupID: groupID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		log.Printf("ListGroupPosts error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch posts"})
	}

	items := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		images := h.loadImageURLs(c, r.ID)
		items = append(items, map[string]interface{}{
			"id":         r.ID.String(),
			"body":       r.Body,
			"images":     images,
			"created_at": r.CreatedAt.Time.Format(time.RFC3339),
			"author": map[string]interface{}{
				"user_id":      r.AuthorUserID.String(),
				"display_name": r.AuthorDisplayName,
				"icon_url":     r.AuthorIconUrl.String,
			},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
		"pagination": map[string]int32{
			"limit":  limit,
			"offset": offset,
		},
	})
}

// ----------------------------------------------------------
// GET /api/posts/:id
// ----------------------------------------------------------

func (h *Handler) Get(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid post id"})
	}

	post, err := h.queries.GetPost(c.Request().Context(), postID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "post not found"})
		}
		log.Printf("GetPost error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// 投稿先グループのメンバーかチェック
	isMember, err := h.queries.IsGroupMember(c.Request().Context(), sqlc.IsGroupMemberParams{
		UserID:  userID,
		GroupID: post.GroupID,
	})
	if err != nil || !isMember {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
	}

	images := h.loadImageURLs(c, post.ID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":         post.ID.String(),
		"group_id":   post.GroupID.String(),
		"body":       post.Body,
		"images":     images,
		"created_at": post.CreatedAt.Time.Format(time.RFC3339),
	})
}

// ----------------------------------------------------------
// DELETE /api/posts/:id (自分の投稿のみ、soft delete)
// ----------------------------------------------------------

func (h *Handler) Delete(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid post id"})
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
