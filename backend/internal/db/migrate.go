package db

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // pgx v5 driver の副作用 import
	_ "github.com/golang-migrate/migrate/v4/source/file"    // ファイルソース driver の副作用 import
)

// RunMigrations は指定パスのマイグレーションをすべて適用する。
// main.go の起動時に呼ばれて、未適用のマイグレーションを自動実行する。
//
// 処理の流れ:
// 1. migrate インスタンスを作成 (ファイルソース + DB)
// 2. m.Up() で未適用のマイグレーションを順次適用
// 3. 既に最新ならエラーではなく ErrNoChange が返るので、これは正常終了として扱う
//
// 単一インスタンス運用前提。複数インスタンスで同時起動すると競合する可能性がある。
func RunMigrations(databaseURL string, migrationsPath string) error {
	// "file://path/to/migrations" 形式と DB URL から migrate インスタンスを生成
	m, err := migrate.New("file://"+migrationsPath, normalizeMigrateURL(databaseURL))
	if err != nil {
		return fmt.Errorf("migrate.New: %w", err)
	}
	defer m.Close()

	// 未適用のマイグレーションを順次実行
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("m.Up: %w", err)
	}

	// ErrNoChange は「適用すべきマイグレーションがない」を意味する正常状態
	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("✓ migrations: no change (already up to date)")
	} else {
		log.Println("✓ migrations: applied")
	}
	return nil
}

// normalizeMigrateURL は DB URL を golang-migrate が期待する形式に変換する。
// Fly Postgres の attach 等で渡されるのは "postgres://..." 形式だが、
// golang-migrate の pgx5 ドライバは "pgx5://..." スキームを要求する。
func normalizeMigrateURL(databaseURL string) string {
	switch {
	case strings.HasPrefix(databaseURL, "postgres://"):
		return "pgx5://" + strings.TrimPrefix(databaseURL, "postgres://")
	case strings.HasPrefix(databaseURL, "postgresql://"):
		return "pgx5://" + strings.TrimPrefix(databaseURL, "postgresql://")
	default:
		return databaseURL
	}
}
