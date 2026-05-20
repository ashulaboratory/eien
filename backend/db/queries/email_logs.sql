-- name: CreateEmailLog :one
INSERT INTO email_logs (recipient_user_id, email_type, recipient_email, subject, status)
VALUES ($1, $2, $3, $4, 'pending')
RETURNING *;

-- name: MarkEmailLogSent :exec
UPDATE email_logs
SET status = 'sent', sent_at = NOW()
WHERE id = $1;

-- name: MarkEmailLogFailed :exec
UPDATE email_logs
SET status = 'failed', error_message = $2
WHERE id = $1;

-- name: HasSentEmailToday :one
SELECT EXISTS(
    SELECT 1 FROM email_logs
    WHERE recipient_user_id = $1
      AND email_type = $2
      AND status = 'sent'
      AND DATE(sent_at AT TIME ZONE 'Asia/Tokyo') = DATE(NOW() AT TIME ZONE 'Asia/Tokyo')
) AS has_sent;
