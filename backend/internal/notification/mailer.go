package notification

import (
	"fmt"
	"net/smtp"
)

// Mailer はメール送信の抽象。
// 開発: SMTPMailer (Mailhog)
// 本番: 将来 ResendMailer などを実装予定
type Mailer interface {
	Send(to, subject, body string) error
}

// SMTPMailer は SMTP 経由でメール送信する (認証なし、Mailhog 向け)
type SMTPMailer struct {
	Host string // 例: "localhost:1025"
	From string // 例: "noreply@eien.local"
}

func (m *SMTPMailer) Send(to, subject, body string) error {
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

	// auth=nil で認証なし (Mailhog はこれでOK)
	return smtp.SendMail(m.Host, nil, m.From, []string{to}, msg)
}
