package notification

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ashulaboratory/eien/backend/db/sqlc"
)

// メールの種類タグ (email_logs.email_type に保存される)
const (
	emailTypeSelf   = "birthday_self"   // 本人への誕生日通知
	emailTypeMember = "birthday_member" // 仲間への誕生日通知
)

// BirthdayService は誕生日通知の送信ロジック。
// scheduler から定期的に呼ばれて、その日が誕生日のユーザーに通知メールを送る。
type BirthdayService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	mailer  Mailer // メーラーは抽象 (開発:Mailhog / 本番:Resend など)
}

func NewBirthdayService(pool *pgxpool.Pool, queries *sqlc.Queries, mailer Mailer) *BirthdayService {
	return &BirthdayService{pool: pool, queries: queries, mailer: mailer}
}

// SendTodaysNotifications は今日が誕生日のユーザーと、その仲間に通知メールを送る。
// 処理の流れ:
// 1. 今日(JST)が誕生日のユーザーを取得
// 2. 各ユーザーに対して:
//    a. 本人に通知 (birthday_self_notify = true の場合)
//    b. 同じグループの仲間にも通知 (各仲間の birthday_member_notify = true の場合)
// 3. 各送信は sendIfNotAlreadySent を経由し、冪等性を担保 (同日に2回送らない)
//
// cron が事故で複数回起動しても、email_logs を見て送信済みなら skip するので安全。
func (s *BirthdayService) SendTodaysNotifications(ctx context.Context) error {
	log.Println("[birthday] starting today's notifications")

	// 今日が誕生日のユーザーを取得 (JST 基準で月日が一致する人)
	birthdayUsers, err := s.queries.ListTodayBirthdayUsersJST(ctx)
	if err != nil {
		return fmt.Errorf("list birthday users: %w", err)
	}

	log.Printf("[birthday] found %d birthday users today", len(birthdayUsers))

	// 統計用カウンタ
	sentCount := 0
	failedCount := 0
	skippedCount := 0

	for _, user := range birthdayUsers {
		// 1. 本人に通知 (birthday_self_notify が ON の場合)
		if user.BirthdaySelfNotify {
			result := s.sendIfNotAlreadySent(ctx, user.ID, user.Email, emailTypeSelf,
				"🎉 Eien — お誕生日おめでとうございます!",
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

		// 2. 同じグループの仲間に通知 (各仲間の birthday_member_notify が ON の場合)
		mates, err := s.queries.ListGroupMatesForUser(ctx, user.ID)
		if err != nil {
			log.Printf("[birthday] failed to list group mates for %s: %v", user.Email, err)
			continue
		}

		for _, mate := range mates {
			// 各仲間ごとに通知設定をチェック (ユーザー側のオプトアウトを尊重)
			if !mate.BirthdayMemberNotify {
				continue
			}
			result := s.sendIfNotAlreadySent(ctx, mate.ID, mate.Email, emailTypeMember,
				"🎂 Eien — 今日は仲間の誕生日です",
				"Eien からのお知らせです。\n\n"+
					"今日は Eien の仲間の誕生日です。\n"+
					"アプリを開いて、お祝いの一言を投稿しませんか?\n\n"+
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

// sendIfNotAlreadySent は冪等性を担保しながらメールを送信する。
// 処理の流れ:
// 1. email_logs を確認し、今日同じ種類のメールを既に送ったかチェック
// 2. 送信済みなら skip (冪等性: cron 複数回実行でも1日1通)
// 3. 未送信なら、email_logs に pending レコードを INSERT
// 4. mailer.Send() で実送信
// 5. 結果に応じて email_logs を sent or failed に UPDATE
//
// 返り値: "sent" / "failed" / "skipped" のいずれか
func (s *BirthdayService) sendIfNotAlreadySent(ctx context.Context, userID uuid.UUID, email, emailType, subject, body string) string {
	// 今日既に同じタイプのメールを送ってないか確認 (冪等性チェック)
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

	// 送信予定として email_logs に pending レコードを INSERT
	// (送信時のメアドをスナップショット保存することで監査ログを守る)
	logRec, err := s.queries.CreateEmailLog(ctx, sqlc.CreateEmailLogParams{
		RecipientUserID: userID,
		EmailType:       emailType,
		RecipientEmail:  email,   // スナップショット (将来メアド変更があっても残る)
		Subject:         subject, // 件名もスナップショット
	})
	if err != nil {
		log.Printf("[birthday] CreateEmailLog error: %v", err)
		return "failed"
	}

	// 実際にメールを送信
	if err := s.mailer.Send(email, subject, body); err != nil {
		// 送信失敗 → email_logs を failed に更新
		log.Printf("[birthday] mailer.Send error: %v", err)
		_ = s.queries.MarkEmailLogFailed(ctx, sqlc.MarkEmailLogFailedParams{
			ID:           logRec.ID,
			ErrorMessage: pgTextOrNil(err.Error()),
		})
		return "failed"
	}

	// 送信成功 → email_logs を sent に更新 (sent_at もセット)
	if err := s.queries.MarkEmailLogSent(ctx, logRec.ID); err != nil {
		log.Printf("[birthday] MarkEmailLogSent error: %v", err)
		// メールは送れてるので sent としてカウント (DB更新失敗は記録のみ)
	}
	return "sent"
}
