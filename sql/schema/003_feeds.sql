-- +goose Up
CREATE TABLE feeds(
  id UUID PRIMARY KEY,
  name VARCHAR,
  url VARCHAR UNIQUE,
  user_id UUID NOT NULL,

  FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE TRIGGER feeds_updated_at_trigger
  BEFORE UPDATE ON feeds
  FOR EACH ROW
  EXECUTE PROCEDURE moddatetime(moddate);

-- +goose Down
DROP TRIGGER feeds_updated_at_trigger ON feeds;
DROP TABLE feeds;
