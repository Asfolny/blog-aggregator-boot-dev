-- name: CreateFeedFollow :one
INSERT INTO feed_follow(feed_id, user_id)
VALUES($1, $2)
RETURNING *;

