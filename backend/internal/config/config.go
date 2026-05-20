package config

import (
	"fmt"
	"os"
)

// Config はアプリケーション全体の設定を保持する
type Config struct {
	DatabaseURL string
	Port        string
	PublicURL   string // 例: https://eien.fly.dev (本番), http://localhost:8080 (開発)
}

// Load は環境変数から設定を読み込む
func Load() (*Config, error) {
	// EIEN_DB_URL を優先、なければ DATABASE_URL (Fly Postgres attach の自動設定) を使う
	dbURL := os.Getenv("EIEN_DB_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		return nil, fmt.Errorf("neither EIEN_DB_URL nor DATABASE_URL is set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

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
