-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT timezone('utc', now()),
    age INTEGER,
    birthday DATE
);

-- +goose Down
DROP TABLE users;
