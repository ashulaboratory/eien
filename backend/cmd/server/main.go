package main

import (
	"context"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/ashulaboratory/eien/backend/db/sqlc"
	"github.com/ashulaboratory/eien/backend/internal/config"
	"github.com/ashulaboratory/eien/backend/internal/db"
	"github.com/ashulaboratory/eien/backend/internal/handler"
)

func main() {
	// 設定ロード
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// DB接続
	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("✓ database connected")

	// sqlc Queries（DB操作のラッパー）
	queries := sqlc.New(pool)

	// ハンドラ作成（依存注入）
	authHandler := handler.NewAuth(queries)

	// Echo
	e := echo.New()

	// ルート登録
	e.POST("/api/auth/register", authHandler.Register)

	// 疎通確認用（残しておく）
	e.GET("/api/ping", func(c echo.Context) error {
		var now string
		err := pool.QueryRow(c.Request().Context(), "SELECT NOW()::text").Scan(&now)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusOK, map[string]string{
			"message": "pong",
			"db_time": now,
		})
	})

	e.Logger.Fatal(e.Start(":" + cfg.Port))
}