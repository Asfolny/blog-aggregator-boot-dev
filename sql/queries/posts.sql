-- name: CreatePost :one
INSERT INTO posts(id, title, url, description, published_at, feed_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPostsByUser :many
SELECT * FROM posts 
WHERE
  feed_id IN (SELECT feed_id FROM feed_follow WHERE user_id = $1)
ORDER BY published_at;
