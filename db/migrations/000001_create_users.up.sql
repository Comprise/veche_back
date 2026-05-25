CREATE TABLE IF NOT EXISTS users (
    id         TEXT    NOT NULL PRIMARY KEY,
    email      TEXT    NOT NULL UNIQUE,
    name       TEXT    NOT NULL,
    provider   TEXT    NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);
