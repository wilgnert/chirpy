-- +goose Up
ALTER TABLE users
add column chirpy_red_expires_at timestamp DEFAULT null;

-- +goose Down
ALTER TABLE users
drop COLUMN chirpy_red_expires_at;