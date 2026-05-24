package notification

import (
	"fmt"
	"net/smtp"
)

// Mailer はメール送信の抽象 (インターフェース)。
// 実装を差し替えるだけで送信先を変更できる設計:
//   - 開発: SMTPMailer (Mailhog にローカル送信、Web UI で確認)
//   - 本番: 将来 ResendMailer などを実装予定
//
// 本番リリース前に Resend など実メール送信プロバイダへの差し替えが必要。
type Mailer interface {
	Send(to, subject, body string) error
}

// SMTPMailer は SMTP 経由でメール送信する開発用実装。
// 認証なしで送信する単純な実装で、Mailhog 向け。
// 本番では認証付き SMTP が必要なため、別実装に差し替える。
type SMTPMailer struct {
	Host string // SMTP サーバホスト (例: "localhost:1025" for Mailhog)
	From string // 送信元アドレス (例: "noreply@eien.local")
}

// Send はメールを送信する。
// MIME ヘッダ + 本文を組み立てて smtp.SendMail に渡す。
func (m *SMTPMailer) Send(to, subject, body string) error {
	// MIME 形式のメッセージを組み立てる
	// Content-Type: UTF-8 のテキストとして送信
	msg := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		m.From, to, subject, body,
	))

	// auth=nil で認証なし送信 (Mailhog はこれで OK)
	// 本番では smtp.PlainAuth などで認証情報を渡す必要あり
	return smtp.SendMail(m.Host, nil, m.From, []string{to}, msg)
}
