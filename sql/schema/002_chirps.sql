-- +goose Up
CREATE TABLE chirps (
  id uuid PRIMARY KEY,
  created_at TIMESTAMP not null,
  updated_at TIMESTAMP not null,
  body text not null,
  user_id UUID not NULL,
  CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE chirps;