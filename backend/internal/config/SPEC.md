# 設定 (config) パッケージ

## 責務
環境変数からアプリケーション設定を読み込み、型付き構造体として提供する。

## ファイル構成
- `config.go` - `Config` 構造体と `Load()` 関数
- `SPEC.md` - 本ファイル

## 読み込む環境変数

| 変数名 | 必須 | デフォルト | 説明 |
| --- | --- | --- | --- |
| EIEN_DB_URL | 必須 | - | PostgreSQL接続URL |
| PORT | 任意 | 8080 | バックエンドサーバが listen するポート |

## 使い方
```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("failed to load config: %v", err)
}
// cfg.DatabaseURL, cfg.Port を使う
```

## 拡張時の手順
新しい環境変数を追加する場合：
1. `Config` 構造体にフィールド追加
2. `Load()` で `os.Getenv` を呼んで設定
3. `.env.example` に変数名を追記
4. 本ファイル（SPEC.md）の表を更新
