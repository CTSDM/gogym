-- +goose Up
CREATE TABLE exercises (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT
);

-- +goose Down
DROP TABLE exercises;
