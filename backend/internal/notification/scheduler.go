package notification

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler は cron 形式の定期実行ジョブを管理する。
// アプリ内スケジューラとして動き、main.go の起動時に Start() が呼ばれて常駐する。
// 別プロセスのバッチに切り出さず、Fly.io 上で常駐プロセス1つで完結させる方針。
type Scheduler struct {
	cron     *cron.Cron       // robfig/cron のスケジューラ本体
	birthday *BirthdayService // 誕生日通知サービスへの参照
}

// NewScheduler は Scheduler を初期化する。
// JST タイムゾーンで cron を動かすことで、日付の境目を JST 基準で扱う。
func NewScheduler(birthday *BirthdayService) *Scheduler {
	// JST タイムゾーンでスケジュール (本番が UTC のサーバでも JST の感覚で動く)
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		loc = time.UTC // tzdata が無い環境でも動くフォールバック
	}
	c := cron.New(cron.WithLocation(loc))
	return &Scheduler{
		cron:     c,
		birthday: birthday,
	}
}

// Start は cron ジョブを登録して開始する。
// 毎朝 7:00 JST に誕生日通知ジョブが走るように設定する。
func (s *Scheduler) Start(ctx context.Context) error {
	// "0 7 * * *" は cron 形式で「毎日 7:00」を意味する
	// 分 時 日 月 曜日 の5つのフィールド
	_, err := s.cron.AddFunc("0 7 * * *", func() {
		if err := s.birthday.SendTodaysNotifications(ctx); err != nil {
			log.Printf("[scheduler] birthday notification error: %v", err)
		}
	})
	if err != nil {
		return err
	}

	// cron スケジューラを開始 (goroutine 内で常駐する)
	s.cron.Start()
	log.Println("✓ scheduler started (birthday notification at 07:00 JST daily)")
	return nil
}

// Stop は cron ジョブを停止する。main.go の defer scheduler.Stop() で呼ばれる。
func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}
