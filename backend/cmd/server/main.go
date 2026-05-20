package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"

	"github.com/ashulaboratory/eien/backend/db/sqlc"
	"github.com/ashulaboratory/eien/backend/internal/auth"
	"github.com/ashulaboratory/eien/backend/internal/config"
	"github.com/ashulaboratory/eien/backend/internal/db"
	"github.com/ashulaboratory/eien/backend/internal/group"
	"github.com/ashulaboratory/eien/backend/internal/notification"
	"github.com/ashulaboratory/eien/backend/internal/post"
)

func main() {
	// 設定ロード
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 自動マイグレーション (db/migrations フォルダがあれば)
	if _, err := os.Stat("db/migrations"); err == nil {
		if err := db.RunMigrations(cfg.DatabaseURL, "db/migrations"); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
	}

	// DB接続
	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("✓ database connected")

	// sqlc Queries
	queries := sqlc.New(pool)

	// 画像ストア
	imageStore := &post.LocalImageStore{
		UploadDir:  "uploads",
		PublicBase: cfg.PublicURL + "/uploads",
	}

	// メーラー (開発: Mailhog, 本番: 後で Resend に差し替え予定)
	mailer := &notification.SMTPMailer{
		Host: "localhost:1025",
		From: "noreply@eien.local",
	}

	// スケジューラ
	birthdayService := notification.NewBirthdayService(pool, queries, mailer)
	scheduler := notification.NewScheduler(birthdayService)
	if err := scheduler.Start(ctx); err != nil {
		log.Fatalf("scheduler start: %v", err)
	}
	defer scheduler.Stop()

	// ハンドラ + ミドルウェア
	authHandler := auth.NewHandler(queries)
	authMiddleware := auth.Middleware(queries)
	groupHandler := group.NewHandler(pool, queries)
	postHandler := post.NewHandler(pool, queries, imageStore)

	// Echo
	e := echo.New()

	// フロント静的配信 (frontend-dist があれば、SPAルーティング対応)
	if _, err := os.Stat("frontend-dist"); err == nil {
		e.Use(echomiddleware.StaticWithConfig(echomiddleware.StaticConfig{
			Root:  "frontend-dist",
			Index: "index.html",
			HTML5: true,
			Skipper: func(c echo.Context) bool {
				p := c.Request().URL.Path
				return strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/uploads/")
			},
		}))
		log.Println("✓ serving frontend from frontend-dist/")
	}

	// 画像配信
	e.Static("/uploads", "uploads")

	// 認証不要のルート
	e.POST("/api/auth/register", authHandler.Register)
	e.POST("/api/auth/login", authHandler.Login)
	e.POST("/api/auth/logout", authHandler.Logout)
	e.GET("/api/invites/:token", groupHandler.GetInvite)

	// 認証必須のルート
	authenticated := e.Group("/api", authMiddleware)
	authenticated.GET("/auth/me", authHandler.Me)

	// グループ
	authenticated.POST("/groups", groupHandler.Create)
	authenticated.GET("/groups", groupHandler.ListMine)
	authenticated.GET("/groups/:id", groupHandler.Get)
	authenticated.GET("/groups/:id/members", groupHandler.ListMembers)
	authenticated.POST("/groups/:id/invites", groupHandler.CreateInvite)
	authenticated.POST("/invites/:token/accept", groupHandler.AcceptInvite)

	// 投稿
	authenticated.POST("/groups/:id/posts", postHandler.Create)
	authenticated.GET("/groups/:id/posts", postHandler.ListByGroup)
	authenticated.GET("/timeline", postHandler.Timeline)
	authenticated.GET("/posts/:id", postHandler.Get)
	authenticated.DELETE("/posts/:id", postHandler.Delete)

	// 疎通確認用
	e.GET("/api/ping", func(c echo.Context) error {
		var now string
		err := pool.QueryRow(c.Request().Context(), "SELECT NOW()::text").Scan(&now)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "pong", "db_time": now})
	})

	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
