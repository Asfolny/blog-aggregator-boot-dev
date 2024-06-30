-- name: CreateFeed :one
INSERT INTO feeds(id, name, url, user_id)
VALUES($1, $2, $3, $4)
RETURNING *;

-- name: GetFeeds :many
SELECT * FROM feeds;


-- name: GetNextFeedsToFetch :many
SELECT * FROM feeds ORDER BY last_fetched_at NULLS FIRST;

-- name: MarkFeedFetched :exec
UPDATE feeds SET last_fetched_at = $1 WHERE id = $2;
