-- name: CreateFeedFollow :one
INSERT INTO feed_follow(feed_id, user_id)
VALUES($1, $2)
RETURNING *;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follow WHERE feed_id = $1 AND user_id = $2;

-- name: AllFeedFollowsByUser :many
SELECT * FROM feed_follow WHERE user_id = $1;
