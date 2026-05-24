package notification

import "github.com/jackc/pgx/v5/pgtype"

// pgTextOrNil は通常の string を pgtype.Text に変換するヘルパー。
// 空文字なら Valid=false (DB の NULL)、それ以外は Valid=true で値を保持。
// PostgreSQL の NULL 許容カラムに値を入れる時に使う。
func pgTextOrNil(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}
