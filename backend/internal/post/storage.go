package post

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// ImageStore は画像保存の抽象 (インターフェース)。
// 実装を差し替えるだけで保存先を変更できる設計:
//   - 開発: LocalImageStore (ローカルディスク保存)
//   - 本番: 将来 R2ImageStore (Cloudflare R2) を実装予定
//
// 本番リリース前に LocalImageStore → R2ImageStore に差し替えが必須。
// 理由: Fly.io のインスタンス再起動でローカルディスクの内容は失われるため。
type ImageStore interface {
	Save(ctx context.Context, content io.Reader, contentType string) (publicURL string, err error)
}

// LocalImageStore はローカルディスクに画像を保存する開発用実装。
// 本番環境では使わない (Fly.io 再起動でデータが消えるため)。
type LocalImageStore struct {
	UploadDir  string // 保存先ディレクトリ (例: "uploads")
	PublicBase string // 公開URLのベース (例: "http://localhost:8080/uploads")
}

// Save は画像コンテンツをディスクに保存して、公開URLを返す。
// 処理の流れ:
// 1. Content-Type から拡張子を決定 ("image/jpeg" → ".jpg")
// 2. UUID v4 でファイル名を生成 (衝突しない)
// 3. uploads ディレクトリがなければ作成
// 4. ファイルに書き込み
// 5. 公開URL (PublicBase + "/" + filename) を返す
func (s *LocalImageStore) Save(ctx context.Context, content io.Reader, contentType string) (string, error) {
	// Content-Type から拡張子の候補を取得
	// 例: "image/jpeg" → [".jpe", ".jpeg", ".jpg"]
	exts, _ := mime.ExtensionsByType(contentType)
	ext := ".bin" // 不明な場合の fallback
	for _, e := range exts {
		// 短い拡張子を優先 (".jpg" を ".jpeg" より優先)
		if ext == ".bin" || len(e) < len(ext) {
			ext = e
		}
	}

	// UUID v4 でユニークなファイル名を生成 (例: "abc-def-...-xyz.jpg")
	filename := uuid.NewString() + ext
	fullPath := filepath.Join(s.UploadDir, filename)

	// uploads ディレクトリがなければ作成 (0o755 はディレクトリの権限)
	if err := os.MkdirAll(s.UploadDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir upload dir: %w", err)
	}

	// ファイルを作成 → 関数終了時に確実にクローズ (defer)
	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	// content (io.Reader) の中身をファイルにコピー
	if _, err := io.Copy(f, content); err != nil {
		return "", fmt.Errorf("copy content: %w", err)
	}

	// 公開URLを返す (例: "http://localhost:8080/uploads/abc-def-...-xyz.jpg")
	return s.PublicBase + "/" + filename, nil
}
