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

// ImageStore は画像保存の抽象。
// 開発: LocalImageStore (ローカルディスク保存)
// 本番: 将来 R2ImageStore (Cloudflare R2) を実装予定
type ImageStore interface {
	Save(ctx context.Context, content io.Reader, contentType string) (publicURL string, err error)
}

// LocalImageStore はローカルディスクに画像を保存する実装
type LocalImageStore struct {
	UploadDir  string // 例: "uploads"
	PublicBase string // 例: "http://localhost:8080/uploads"
}

func (s *LocalImageStore) Save(ctx context.Context, content io.Reader, contentType string) (string, error) {
	// content type から拡張子を決定
	exts, _ := mime.ExtensionsByType(contentType)
	ext := ".bin"
	for _, e := range exts {
		// 短い拡張子を優先 (".jpg" > ".jpeg")
		if ext == ".bin" || len(e) < len(ext) {
			ext = e
		}
	}

	filename := uuid.NewString() + ext
	fullPath := filepath.Join(s.UploadDir, filename)

	if err := os.MkdirAll(s.UploadDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir upload dir: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, content); err != nil {
		return "", fmt.Errorf("copy content: %w", err)
	}

	return s.PublicBase + "/" + filename, nil
}
