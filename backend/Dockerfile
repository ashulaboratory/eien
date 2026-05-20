# ============================================================
# Eien 本番用 Dockerfile (マルチステージビルド)
# - frontend を Vite で build
# - backend を Go で build
# - 最小 alpine イメージに2つの成果物を配置
# ============================================================

# --- Stage 1: フロントエンドビルド ---
FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# --- Stage 2: バックエンドビルド ---
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -o /app/eien-server ./cmd/server

# --- Stage 3: 最終ランタイムイメージ ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Tokyo
WORKDIR /app

# バックエンドバイナリ
COPY --from=backend-builder /app/eien-server /app/eien-server

# DB マイグレーション (Goから自動実行するため)
COPY --from=backend-builder /app/db/migrations /app/db/migrations

# フロントエンドビルド成果物
COPY --from=frontend-builder /app/dist /app/frontend-dist

# 画像保存ディレクトリ (Fly volumes でマウントされる)
RUN mkdir -p /app/uploads

EXPOSE 8080
CMD ["/app/eien-server"]
