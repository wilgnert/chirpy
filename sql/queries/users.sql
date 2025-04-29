-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users WHERE 1 = 1;

-- name: GetUserByEmail :one
select * from users where email = $1;

-- name: UpdateUserEmailAndPassword :one
UPDATE users 
set email = $2, hashed_password = $3, updated_at = NOW()
where id = $1
RETURNING id, created_at, updated_at, email, chirpy_red_expires_at;

-- name: UpdateUserChirpyRed :one
update users
set updated_at=NOW(), chirpy_red_expires_at=$2
where id = $1
RETURNING id, created_at, updated_at, email, chirpy_red_expires_at;
