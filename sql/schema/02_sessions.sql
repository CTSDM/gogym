-- +goose Up
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    date DATE NOT NULL,
    start_timestamp TIMESTAMP DEFAULT timezone('utc', now()),
    duration_minutes SMALLINT,
    user_id UUID NOT NULL,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id)
    REFERENCES users(id)
    ON DELETE CASCADE
);

-- +goose Down
DROP TABLE sessions;
