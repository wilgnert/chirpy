-- +goose Up
ALTER TABLE users 
add column hashed_password text DEFAULT 'unset' not null;

-- +goose Down
ALTER table users drop COLUMN hashed_password;