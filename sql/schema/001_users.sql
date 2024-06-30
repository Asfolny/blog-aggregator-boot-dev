-- +goose Up
CREATE TABLE users (
  id UUID PRIMARY KEY,
  name VARCHAR NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER users_updated_at_trigger
  BEFORE UPDATE ON users
  FOR EACH ROW
  EXECUTE PROCEDURE moddatetime(moddate); -- moddatetime is it's own extension and must be enabled before it can be used

-- +goose Down
DROP TRIGGER users_updated_at_trigger ON users;
DROP TABLE users;
