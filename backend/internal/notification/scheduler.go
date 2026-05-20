package notification

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler は cron 形式の定期実行ジョブを管理する
type Scheduler struct {
	cron     *cron.Cron
	birthday *BirthdayService
}

func NewScheduler(birthday *BirthdayService) *Scheduler {
	// JST タイムゾーンでスケジュール
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		loc = time.UTC
	}
	c := cron.New(cron.WithLocation(loc))
	return &Scheduler{
		cron:     c,
		birthday: birthday,
	}
}

// Start は cron ジョブを登録して開始する
func (s *Scheduler) Start(ctx context.Context) error {
	// 毎朝7:00 JST に誕生日通知を送る
	_, err := s.cron.AddFunc("0 7 * * *", func() {
		if err := s.birthday.SendTodaysNotifications(ctx); err != nil {
			log.Printf("[scheduler] birthday notification error: %v", err)
		}
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	log.Println("✓ scheduler started (birthday notification at 07:00 JST daily)")
	return nil
}

// Stop は cron ジョブを停止する
func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}
