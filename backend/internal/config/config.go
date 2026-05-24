package config

import (
	"fmt"
	"os"
)

// Config はアプリケーション全体の設定を保持する構造体。
// 環境変数から値を読み込んで保持する。
type Config struct {
	DatabaseURL string // PostgreSQL の接続URL
	Port        string // サーバが listen するポート
	PublicURL   string // 外部公開URL (例: https://eien.fly.dev / http://localhost:8080)
}

// Load は環境変数から設定を読み込んで Config を返す。
// 環境変数が未設定の場合は、デフォルト値を使うか、必須項目ならエラーを返す。
func Load() (*Config, error) {
	// EIEN_DB_URL を優先、なければ DATABASE_URL (Fly Postgres attach 時の自動設定) を使う
	// 両方未設定ならエラー (DB なしでは動かないため)
	dbURL := os.Getenv("EIEN_DB_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		return nil, fmt.Errorf("neither EIEN_DB_URL nor DATABASE_URL is set")
	}

	// PORT 未設定なら 8080 をデフォルトに
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// EIEN_PUBLIC_URL 未設定ならローカル開発用のURLをデフォルトに
	// 画像のURL生成 (LocalImageStore.PublicBase) などに使われる
	publicURL := os.Getenv("EIEN_PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:" + port
	}

	return &Config{
		DatabaseURL: dbURL,
		Port:        port,
		PublicURL:   publicURL,
	}, nil
}
