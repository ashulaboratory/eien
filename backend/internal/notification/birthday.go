package notification

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ashulaboratory/eien/backend/db/sqlc"
)

const (
	emailTypeSelf   = "birthday_self"
	emailTypeMember = "birthday_member"
)

// BirthdayService は誕生日通知の送信ロジック
type BirthdayService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	mailer  Mailer
}

func NewBirthdayService(pool *pgxpool.Pool, queries *sqlc.Queries, mailer Mailer) *BirthdayService {
	return &BirthdayService{pool: pool, queries: queries, mailer: mailer}
}

// SendTodaysNotifications は今日が誕生日のユーザーと、その仲間に通知を送る
func (s *BirthdayService) SendTodaysNotifications(ctx context.Context) error {
	log.Println("[birthday] starting today's notifications")

	birthdayUsers, err := s.queries.ListTodayBirthdayUsersJST(ctx)
	if err != nil {
		return fmt.Errorf("list birthday users: %w", err)
	}

	log.Printf("[birthday] found %d birthday users today", len(birthdayUsers))

	sentCount := 0
	failedCount := 0
	skippedCount := 0

	for _, user := range birthdayUsers {
		// 1. 本人に送る (birthday_self_notify = true なら)
		if user.BirthdaySelfNotify {
			result := s.sendIfNotAlreadySent(ctx, user.ID, user.Email, emailTypeSelf,
				"🎉 Eien — お誕生日おめでとうございます！",
				"Eien からのお祝いメールです。\n\n"+
					"今日はあなたの誕生日。素敵な一日になりますように。\n"+
					"誕生日の今日は Eien で特別な表示が出ています。よかったらアプリを開いてみてください。\n\n"+
					"-- Eien")
			switch result {
			case "sent":
				sentCount++
			case "failed":
				failedCount++
			case "skipped":
				skippedCount++
			}
		}

		// 2. 同じグループの仲間に送る (各仲間の birthday_member_notify = true なら)
		mates, err := s.queries.ListGroupMatesForUser(ctx, user.ID)
		if err != nil {
			log.Printf("[birthday] failed to list group mates for %s: %v", user.Email, err)
			continue
		}

		for _, mate := range mates {
			if !mate.BirthdayMemberNotify {
				continue
			}
			result := s.sendIfNotAlreadySent(ctx, mate.ID, mate.Email, emailTypeMember,
				"🎂 Eien — 今日は仲間の誕生日です",
				"Eien からのお知らせです。\n\n"+
					"今日は Eien の仲間の誕生日です。\n"+
					"アプリを開いて、お祝いの一言を投稿しませんか？\n\n"+
					"-- Eien")
			switch result {
			case "sent":
				sentCount++
			case "failed":
				failedCount++
			case "skipped":
				skippedCount++
			}
		}
	}

	log.Printf("[birthday] done. sent=%d, failed=%d, skipped=%d", sentCount, failedCount, skippedCount)
	return nil
}

// sendIfNotAlreadySent は今日同じ種類のメールを送ってなければ送る (べき等性)
func (s *BirthdayService) sendIfNotAlreadySent(ctx context.Context, userID uuid.UUID, email, emailType, subject, body string) string {
	// 今日既に送ってないかチェック
	hasSent, err := s.queries.HasSentEmailToday(ctx, sqlc.HasSentEmailTodayParams{
		RecipientUserID: userID,
		EmailType:       emailType,
	})
	if err != nil {
		log.Printf("[birthday] HasSentEmailToday error: %v", err)
		return "failed"
	}
	if hasSent {
		log.Printf("[birthday] already sent %s to %s today, skipping", emailType, email)
		return "skipped"
	}

	// pending log を作成
	logRec, err := s.queries.CreateEmailLog(ctx, sqlc.CreateEmailLogParams{
		RecipientUserID: userID,
		EmailType:       emailType,
		RecipientEmail:  email,
		Subject:         subject,
	})
	if err != nil {
		log.Printf("[birthday] CreateEmailLog error: %v", err)
		return "failed"
	}

	// 実際に送信
	if err := s.mailer.Send(email, subject, body); err != nil {
		log.Printf("[birthday] mailer.Send error: %v", err)
		_ = s.queries.MarkEmailLogFailed(ctx, sqlc.MarkEmailLogFailedParams{
			ID:           logRec.ID,
			ErrorMessage: pgTextOrNil(err.Error()),
		})
		return "failed"
	}

	if err := s.queries.MarkEmailLogSent(ctx, logRec.ID); err != nil {
		log.Printf("[birthday] MarkEmailLogSent error: %v", err)
		// メールは送れてるので sent としてカウント
	}
	return "sent"
}
