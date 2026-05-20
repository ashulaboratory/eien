# 通知 (notification) パッケージ

## 責務
誕生日通知メールの送信。cron スケジューラと、手動実行用CLIから利用。

## ファイル構成
- `mailer.go` - `Mailer` インターフェースと `SMTPMailer` 実装
- `birthday.go` - `BirthdayService` (今日の誕生日ユーザーを取得 → メール送信)
- `scheduler.go` - cron 形式の定期実行（毎朝7:00 JST）
- `helpers.go` - pgtype変換などのヘルパー
- `SPEC.md` - 本ファイル

## 動作

### 1. 自動実行 (本番)
- アプリ起動時に `Scheduler.Start()` が呼ばれ、cron ジョブが登録される
- 毎朝 7:00 JST に `BirthdayService.SendTodaysNotifications(ctx)` が走る

### 2. 手動実行 (開発・テスト)
- `make backend-birthday` で `cmd/birthday/main.go` を実行
- DB に接続して `SendTodaysNotifications()` を1回だけ呼ぶ

## メール送信フロー

1. 今日が誕生日のユーザーを `ListTodayBirthdayUsersJST` で取得
2. 各誕生日ユーザーについて:
   - `birthday_self_notify=true` なら本人に「お誕生日おめでとう」メール
   - 同じグループの仲間 (`ListGroupMatesForUser`) のうち `birthday_member_notify=true` の人に「仲間の誕生日」メール
3. 各送信前に `HasSentEmailToday` で重複チェック (べき等性)
4. 送信成功/失敗を `email_logs` に記録

## べき等性
同じ日に同じ種類のメール (`birthday_self` / `birthday_member`) を同じユーザーに2回送らないように、`email_logs` で送信状況を確認してから送る。

これにより、アプリの再起動やバッチの複数回実行があっても、ユーザーに重複メールが届かない。

## Mailer の差し替え

### 開発: SMTPMailer + Mailhog
```go
mailer := &notification.SMTPMailer{
    Host: "localhost:1025",
    From: "noreply@eien.local",
}
```
→ Mailhog の Web UI (http://localhost:8025) で受信メール確認可能

### 本番: 将来 Resend など
`Mailer` インターフェースを満たす別実装を作って差し替える。

## メール本文

### 本人向け (birthday_self)
```
🎉 Eien — お誕生日おめでとうございます！

今日はあなたの誕生日。素敵な一日になりますように。
誕生日の今日は Eien で特別な表示が出ています。よかったらアプリを開いてみてください。

-- Eien
```

### 仲間向け (birthday_member)
```
🎂 Eien — 今日は仲間の誕生日です

今日は Eien の仲間の誕生日です。
アプリを開いて、お祝いの一言を投稿しませんか？

-- Eien
```

具体的な「誰の」誕生日かはメール本文に出していない（マルチプロフィール設計上、グループによって表示名が違うため）。アプリ内の豪華表示で確認する想定。

## 依存
- `db/sqlc` (Queries: ListTodayBirthdayUsersJST, ListGroupMatesForUser, CreateEmailLog, MarkEmailLogSent/Failed, HasSentEmailToday)
- `pgxpool.Pool`
- `github.com/robfig/cron/v3` (scheduler)
- `net/smtp` (Go標準)

## 関連SQL
- `db/queries/birthday.sql` - ListTodayBirthdayUsersJST, ListGroupMatesForUser
- `db/queries/email_logs.sql` - CreateEmailLog, MarkEmailLogSent, MarkEmailLogFailed, HasSentEmailToday

## 関連マイグレーション
- `db/migrations/000008_create_email_logs.up.sql`
