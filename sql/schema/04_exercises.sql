-- +goose Up
CREATE TABLE exercises (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    resources TEXT [],
    description TEXT,
    muscle_groups TEXT []
);

-- +goose Down
DROP TABLE exercises;
