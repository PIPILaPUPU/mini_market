-- +goose Up
CREATE TABLE users (
    id            UUID PRIMARY KEY,
    username      VARCHAR(32) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name    VARCHAR(100) NOT NULL DEFAULT '',
    last_name     VARCHAR(100) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX users_username_unique_ci ON users (LOWER(username));
CREATE UNIQUE INDEX users_email_unique_ci ON users (LOWER(email));

CREATE TABLE refresh_sessions (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX refresh_sessions_user_id_idx ON refresh_sessions(user_id);
CREATE INDEX refresh_sessions_expires_at_idx ON refresh_sessions(expires_at);

-- +goose Down
DROP TABLE IF EXISTS refresh_sessions;
DROP TABLE IF EXISTS users;