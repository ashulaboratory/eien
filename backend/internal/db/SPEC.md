# DB接続 (db) パッケージ

## 責務
PostgreSQL への接続プールを作成し、ヘルスチェック（Ping）を行う。

## ファイル構成
- `db.go` - `New()` 関数（接続プール作成 + Ping）
- `SPEC.md` - 本ファイル

## 使い方
```go
pool, err := db.New(ctx, cfg.DatabaseURL)
if err != nil {
    log.Fatalf("...")
}
defer pool.Close()

queries := sqlc.New(pool) // sqlc経由のDB操作はここから
```

## 設計判断
- **pgxpool** を使用（pgx 標準の接続プール）
- 起動時に `Ping` で接続確認 → 失敗したらアプリ起動失敗（早期失敗 fail-fast）
- 接続パラメータは pgxpool のデフォルト（最大25接続）。必要なら後で調整
- pgx は PostgreSQL 専用ドライバ。`database/sql` 標準より高速かつ PG固有機能を活用可能

## 関連
- マイグレーションは `db/migrations/` (golang-migrate)
- クエリは `db/queries/` (sqlc 入力)
- 生成コードは `db/sqlc/` (sqlc 出力、コミット対象)
