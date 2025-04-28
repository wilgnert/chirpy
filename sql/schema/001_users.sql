-- +goose Up
CREATE TABLE users (
  id uuid PRIMARY key DEFAULT gen_random_uuid(),
  created_at timestamp not null,
  updated_at timestamp not null,
  email text UNIQUE not null
);

-- +goose Down
DROP TABLE users;