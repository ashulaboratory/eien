package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New は PostgreSQL の接続プールを作成し、接続できることを確認する。
// 接続プールは pgxpool が管理し、複数の goroutine から安全に使える。
// main.go の起動時に呼ばれ、defer pool.Close() で確実に解放される。
func New(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	// 接続プール作成 (この時点では実際には接続しない)
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// 接続テスト (実際に DB に ping を打って疎通確認)
	// ここで失敗したら、起動段階で気付ける
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
