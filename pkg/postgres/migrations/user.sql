CREATE TABLE IF NOT EXISTS users (
    id         TEXT        PRIMARY KEY NOT NULL,
    name       TEXT        NOT NULL,
    email      TEXT        UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
