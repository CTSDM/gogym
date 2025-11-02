-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    hashed_password TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT timezone('utc', now()),
    country TEXT,
    birthday DATE
);

-- +goose Down
DROP TABLE users;
