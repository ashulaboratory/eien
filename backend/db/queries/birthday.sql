-- name: ListTodayBirthdayUsersJST :many
SELECT * FROM users
WHERE EXTRACT(MONTH FROM birthday) = EXTRACT(MONTH FROM (NOW() AT TIME ZONE 'Asia/Tokyo'))
  AND EXTRACT(DAY FROM birthday) = EXTRACT(DAY FROM (NOW() AT TIME ZONE 'Asia/Tokyo'));

-- name: ListGroupMatesForUser :many
SELECT DISTINCT u.*
FROM users u
INNER JOIN group_members gm ON gm.user_id = u.id
WHERE gm.group_id IN (SELECT gm_self.group_id FROM group_members gm_self WHERE gm_self.user_id = $1)
  AND u.id != $1;
