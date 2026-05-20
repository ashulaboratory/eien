package notification

import "github.com/jackc/pgx/v5/pgtype"

// pgTextOrNil は string を pgtype.Text に変換するヘルパー
func pgTextOrNil(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}
