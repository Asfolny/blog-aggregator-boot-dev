-- +goose Up
CREATE TABLE posts (
  id UUID PRIMARY KEY,
  title VARCHAR NOT NULL,
  url VARCHAR NOT NULL UNIQUE,
  description TEXT,
  feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE ON UPDATE CASCADE,
  publised_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
);

CREATE TRIGGER posts_updated_at_trigger
  BEFORE UPDATE ON posts
  FOR EACH ROW
  EXECUTE PROCEDURE moddatetime(moddate);

-- +goose Down
DROP TRIGGER posts_updated_at_trigger ON posts;
DROP TABLE posts;
