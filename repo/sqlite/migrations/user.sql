-- Enable WAL mode once; persists in the DB file header across all future connections.
PRAGMA journal_mode=WAL;

CREATE TABLE IF NOT EXISTS users (
    id         TEXT     PRIMARY KEY NOT NULL,
    name       TEXT     NOT NULL,
    email      TEXT     UNIQUE NOT NULL,
    created_at DATETIME NOT NULL
);
