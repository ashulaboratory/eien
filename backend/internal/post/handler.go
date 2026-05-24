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

// 投稿に関する制約値
const (
	maxBodyLength = 1000             // 投稿本文の最大文字数
	maxImageCount = 4                // 1投稿あたりの最大画像数
	maxImageSize  = 10 * 1024 * 1024 // 画像1枚あたりの最大サイズ(10MB)
)

// Handler は投稿関連の HTTP ハンドラ
type Handler struct {
	pool    *pgxpool.Pool  // トランザクション用に接続プール直接保持
	queries *sqlc.Queries  // sqlc 生成のクエリ実行用
	images  ImageStore     // 画像保存(インターフェース、開発:Local / 本番:R2)
}

func NewHandler(pool *pgxpool.Pool, queries *sqlc.Queries, images ImageStore) *Handler {
	return &Handler{pool: pool, queries: queries, images: images}
}

// ----------------------------------------------------------
// 共通ヘルパー
// ----------------------------------------------------------

// 画像のMIMEタイプが許可されているか判定
func isValidImageContentType(ct string) bool {
	switch ct {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	}
	return false
}

// クエリパラメータから limit / offset を取得 (ページネーション用)
// limit: 1〜100、デフォルトは defaultLimit
// offset: 0以上、デフォルトは 0
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

// 投稿に紐づく画像URL一覧を取得 (タイムライン・詳細表示用)
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
// 投稿を作成 (multipart form: body テキスト + images ファイル配列、画像はオプション)
// ----------------------------------------------------------

// Create は新規投稿のハンドラ。
// 処理の流れ:
// 1. URL パラメータ :id からグループIDを取り出し
// 2. ユーザーがそのグループのメンバーか確認 (権限チェック)
// 3. 本文と画像のバリデーション
// 4. 画像があれば ImageStore に保存し、URL一覧を取得
// 5. トランザクションで posts + post_images を一括 INSERT
// 6. 成功時は 201 Created で投稿情報を返す
func (h *Handler) Create(c echo.Context) error {
	// 認証ミドルウェアが設定した user_id を取得
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	// URL の :id をパースしてグループIDに
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid group id"})
	}

	// メンバーシップ確認 (このユーザーがこのグループに所属しているか)
	isMember, err := h.queries.IsGroupMember(c.Request().Context(), sqlc.IsGroupMemberParams{
		UserID:  userID,
		GroupID: groupID,
	})
	if err != nil || !isMember {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "not a member of this group"})
	}

	// multipart form の "body" フィールドから本文を取得
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
		files := form.File["images"] // "images" フィールドの全ファイル
		if len(files) > maxImageCount {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "too many images (max 4)"})
		}
		// 各画像をバリデーションして ImageStore に保存
		for _, fh := range files {
			if fh.Size > maxImageSize {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "image too large (max 10MB)"})
			}
			ct := fh.Header.Get("Content-Type")
			if !isValidImageContentType(ct) {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "unsupported image type (allowed: jpeg, png, gif, webp)"})
			}
			// アップロードされたファイルを開く
			src, err := fh.Open()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read uploaded file"})
			}
			// ImageStore に保存 → 公開URLを取得
			url, err := h.images.Save(ctx, src, ct)
			src.Close()
			if err != nil {
				log.Printf("image save error: %v", err)
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to store image"})
			}
			imageURLs = append(imageURLs, url)
		}
	}

	// posts と post_images をトランザクションで一括 INSERT
	// 片方失敗したら全部ロールバックして整合性を保つ
	tx, err := h.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		log.Printf("BeginTx error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	defer tx.Rollback(ctx) // Commit 成功時の Rollback は無視されるので安全

	// トランザクション内で実行する Queries オブジェクト
	q := h.queries.WithTx(tx)

	// posts テーブルに投稿本体を INSERT
	post, err := q.CreatePost(ctx, sqlc.CreatePostParams{
		GroupID:      groupID,
		AuthorUserID: userID,
		Body:         body,
	})
	if err != nil {
		log.Printf("CreatePost error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create post"})
	}

	// 各画像URLを post_images テーブルに INSERT (sort_order で順序を保持)
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

	// トランザクションをコミット (ここで初めて DB に反映)
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
// 自分が所属する全グループの投稿を、新しい順に取得 (タイムライン)
// ----------------------------------------------------------

// Timeline は所属グループ横断のタイムラインを返すハンドラ。
// 複合インデックス idx_posts_group_id_created_at により、新しい順の取得が高速。
func (h *Handler) Timeline(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	// ページネーション (デフォルト 20件)
	limit, offset := parseLimitOffset(c, 20)

	// 所属グループ横断で投稿を取得 (JOIN を内部で行う)
	rows, err := h.queries.ListTimeline(c.Request().Context(), sqlc.ListTimelineParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		log.Printf("ListTimeline error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch timeline"})
	}

	// 各投稿に画像URLを付加してレスポンス用に整形
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
				"display_name": r.AuthorDisplayName, // グループ単位の表示名
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
// 特定グループの投稿一覧を取得
// ----------------------------------------------------------

func (h *Handler) ListByGroup(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	// URL の :id からグループIDを取得
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

	// ページネーション付きで投稿一覧を取得
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

	// 各投稿に画像URLを付加
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
// 投稿を1件取得 (グループメンバーのみ閲覧可)
// ----------------------------------------------------------

func (h *Handler) Get(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid post id"})
	}

	// 投稿を取得 (存在しなければ 404)
	post, err := h.queries.GetPost(c.Request().Context(), postID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "post not found"})
		}
		log.Printf("GetPost error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	// 投稿先グループのメンバーかチェック (権限)
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
// DELETE /api/posts/:id
// 自分の投稿のみ削除可能 (論理削除: deleted_at にタイムスタンプをセット)
// ----------------------------------------------------------

// Delete は投稿の論理削除ハンドラ。
// SoftDeletePost の SQL 内で WHERE author_user_id = $2 の条件付きなので、
// 他人の投稿を削除しようとしても影響なし (権限チェック兼ねる)。
func (h *Handler) Delete(c echo.Context) error {
	userID, _ := auth.UserIDFromContext(c.Request().Context())

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid post id"})
	}

	// SoftDeletePost は UPDATE posts SET deleted_at = NOW() WHERE id = $1 AND author_user_id = $2
	// → DELETE 文ではなく UPDATE 文を使うことで、レコードは残り deleted_at で削除済みと判定
	if err := h.queries.SoftDeletePost(c.Request().Context(), sqlc.SoftDeletePostParams{
		ID:           postID,
		AuthorUserID: userID,
	}); err != nil {
		log.Printf("SoftDeletePost error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return c.NoContent(http.StatusNoContent)
}
