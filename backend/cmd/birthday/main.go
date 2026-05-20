// Package main は誕生日通知バッチを手動実行するCLI。
// 開発/テスト用途。本番では server 内の scheduler が cron で自動実行する。
//
// 使い方:
//   make backend-birthday
//   (内部的に cd backend && go run cmd/birthday/main.go)
package main

import (
	"context"
	"log"

	"github.com/ashulaboratory/eien/backend/db/sqlc"
	"github.com/ashulaboratory/eien/backend/internal/config"
	"github.com/ashulaboratory/eien/backend/internal/db"
	"github.com/ashulaboratory/eien/backend/internal/notification"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	mailer := &notification.SMTPMailer{
		Host: "localhost:1025",
		From: "noreply@eien.local",
	}

	service := notification.NewBirthdayService(pool, queries, mailer)
	if err := service.SendTodaysNotifications(ctx); err != nil {
		log.Fatalf("send notifications: %v", err)
	}
	log.Println("✓ done")
}
