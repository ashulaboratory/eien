// Package main は誕生日通知バッチを手動実行する CLI。
// 開発 / テスト用途。本番では server 内の scheduler が cron で自動実行する。
//
// 使い方:
//   make backend-birthday
//   (内部的に cd backend && go run cmd/birthday/main.go)
//
// server の cron と同じ BirthdayService を呼ぶので、本番と同じロジックを単発で
// 動かして確認できる。
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
	// 設定をロード
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// DB接続プールを作成
	ctx := context.Background()
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	// メーラーを初期化 (開発時は Mailhog)
	mailer := &notification.SMTPMailer{
		Host: "localhost:1025",
		From: "noreply@eien.local",
	}

	// 誕生日通知サービスを作成して、今日分の通知を送信
	// server 起動時の scheduler が呼ぶのと同じメソッドを使う
	service := notification.NewBirthdayService(pool, queries, mailer)
	if err := service.SendTodaysNotifications(ctx); err != nil {
		log.Fatalf("send notifications: %v", err)
	}
	log.Println("✓ done")
}
