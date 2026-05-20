-- name: CreateInviteLink :one
INSERT INTO invite_links (group_id, token, created_by_user_id, expires_at, max_uses)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetInviteLink :one
SELECT * FROM invite_links
WHERE token = $1
  AND expires_at > NOW()
  AND current_uses < max_uses
LIMIT 1;

-- name: IncrementInviteLinkUses :exec
UPDATE invite_links
SET current_uses = current_uses + 1
WHERE token = $1;

-- name: DeleteExpiredInviteLinks :exec
DELETE FROM invite_links
WHERE expires_at <= NOW() OR current_uses >= max_uses;
