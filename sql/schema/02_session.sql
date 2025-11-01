-- +goose Up
CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    date DATE NOT NULL,
    time_start TIMESTAMP DEFAULT timezone('utc', now()),
    duration_minutes SMALLINT NOT NULL,
    user_id UUID NOT NULL,
    CONSTRAINT fk_user_id FOREIGN KEY(user_id)
    REFERENCES users(id)
    ON DELETE CASCADE
);

-- +goose Down
DROP TABLE sessions;
