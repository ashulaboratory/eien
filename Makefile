# Eien 開発用 Makefile
# 使い方: make <target>
# 全コマンド一覧: make help

# .env を読み込んで環境変数として export
# 各コマンド内で $(EIEN_DB_URL) のように参照できる
ifneq (,$(wildcard .env))
    include .env
    export
endif

.PHONY: help
help: ## このヘルプを表示
	@echo "Eien 開発コマンド一覧:"
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ==============================================
# 開発用コンテナ (PostgreSQL + Mailhog)
# ==============================================
.PHONY: dev-up
dev-up: ## 開発用コンテナを起動 (DB + Mailhog)
	docker compose up -d

.PHONY: dev-down
dev-down: ## 開発用コンテナを停止
	docker compose down

.PHONY: dev-logs
dev-logs: ## 開発用コンテナのログを追跡 (Ctrl+Cで終了)
	docker compose logs -f

.PHONY: dev-status
dev-status: ## 開発用コンテナの状態を表示
	docker compose ps

# ==============================================
# データベース (PostgreSQL)
# ==============================================
.PHONY: db-shell
db-shell: ## PostgreSQL に psql で接続 (\q で抜ける)
	docker compose exec postgres psql -U eien -d eien

.PHONY: db-tables
db-tables: ## テーブル一覧を表示
	docker compose exec postgres psql -U eien -d eien -c "\dt"

# ==============================================
# マイグレーション
# ==============================================
.PHONY: migrate-up
migrate-up: ## マイグレーションを全て適用
	migrate -database "$(EIEN_DB_URL)" -path backend/db/migrations up

.PHONY: migrate-down
migrate-down: ## マイグレーションを1段戻す
	migrate -database "$(EIEN_DB_URL)" -path backend/db/migrations down 1

.PHONY: migrate-version
migrate-version: ## 現在のマイグレーションバージョンを表示
	migrate -database "$(EIEN_DB_URL)" -path backend/db/migrations version

.PHONY: migrate-create
migrate-create: ## 新規マイグレーション作成 (例: make migrate-create name=add_indexes)
	@if [ -z "$(name)" ]; then \
		echo "ERROR: name= が必要です。例: make migrate-create name=add_posts_table"; \
		exit 1; \
	fi
	migrate create -ext sql -dir backend/db/migrations -seq $(name)

# ==============================================
# バックエンド (Go + Echo)
# ==============================================
.PHONY: backend-run
backend-run: ## バックエンドサーバを起動 (Ctrl+Cで終了)
	cd backend && go run cmd/server/main.go

.PHONY: backend-build
backend-build: ## バックエンドをビルドしてバイナリ生成
	cd backend && go build -o ../bin/server cmd/server/main.go

.PHONY: backend-test
backend-test: ## バックエンドのテストを実行
	cd backend && go test ./...

.PHONY: backend-tidy
backend-tidy: ## go.mod を整理 (未使用依存削除など)
	cd backend && go mod tidy
