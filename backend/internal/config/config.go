package config

import (
	"fmt"
	"os"
)

// Config はアプリケーション全体の設定を保持する
type Config struct {
	DatabaseURL string
	Port        string
}

// Load は環境変数から設定を読み込む
func Load() (*Config, error) {
	dbURL := os.Getenv("EIEN_DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("EIEN_DB_URL is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // デフォルト値
	}

	return &Config{
		DatabaseURL: dbURL,
		Port:        port,
	}, nil
}