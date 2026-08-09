-- +goose Up
BEGIN;
CREATE TABLE access_tokens (
  id BIGSERIAL PRIMARY KEY,
  token_hash CHAR(64) NOT NULL UNIQUE,
  user_id UUID NOT NULL,
  client_id TEXT,
  scopes TEXT,
  issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  revoked BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_access_tokens_user_id ON access_tokens(user_id);
COMMIT;

-- +goose Down
BEGIN;
DROP INDEX IF EXISTS idx_access_tokens_user_id;
DROP TABLE IF EXISTS access_tokens;
COMMIT;
