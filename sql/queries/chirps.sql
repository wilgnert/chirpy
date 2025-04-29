-- name: CreateChirp :one
INSERT INTO chirps(id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING *;

-- name: DeleteAllChirps :exec
delete from chirps where 1 = 1;

-- name: GetAllChirps :many
select * from chirps order by created_at;

-- name: GetAllChirpsFromAuthorID :many
select * from chirps where user_id = $1 order by created_at;

-- name: GetChirpByID :one
select * from chirps where id = $1;

-- name: DeleteChirpByID :exec
delete from chirps where id = $1;