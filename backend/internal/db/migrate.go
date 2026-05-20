package db

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // pgx v5 driver
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations は指定パスのマイグレーションをすべて適用する。
// 既に適用済みなら何もしない。
func RunMigrations(databaseURL string, migrationsPath string) error {
	m, err := migrate.New("file://"+migrationsPath, normalizeMigrateURL(databaseURL))
	if err != nil {
		return fmt.Errorf("migrate.New: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("m.Up: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("✓ migrations: no change (already up to date)")
	} else {
		log.Println("✓ migrations: applied")
	}
	return nil
}

// normalizeMigrateURL converts Fly/Postgres URLs to pgx5 scheme for golang-migrate.
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
