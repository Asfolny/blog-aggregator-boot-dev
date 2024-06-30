-- +goose Up
CREATE TABLE feed_follow(
  feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE ON UPDATE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (feed_id, user_id)
);

CREATE TRIGGER feed_follow_updated_at_trigger
  BEFORE UPDATE ON feed_follow
  FOR EACH ROW
  EXECUTE PROCEDURE moddatetime(moddate);

-- +goose Down
DROP TRIGGER feed_follow_updated_at_trigger ON feed_follow;
DROP TABLE feed_follow;
