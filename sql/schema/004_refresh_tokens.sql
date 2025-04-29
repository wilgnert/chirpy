-- +goose Up
create table refresh_tokens (
  token text PRIMARY KEY,
  created_at TIMESTAMP not null,
  updated_at TIMESTAMP not null,
  user_id UUID NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  revoked_at TIMESTAMP,
  CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);

-- +goose Down
drop table refresh_tokens;