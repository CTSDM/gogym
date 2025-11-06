-- +goose Up
CREATE TABLE logs(
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT timezone('utc', now()),
    last_modified_at TIMESTAMP DEFAULT timezone('utc', now()),
    weight FLOAT,
    reps INTEGER NOT NULL,
    logs_order INTEGER NOT NULL,
    exercise_id INTEGER NOT NULL,
    set_id BIGINT NOT NULL,
    CONSTRAINT fk_exercise_id FOREIGN KEY(exercise_id)
    REFERENCES exercises(id),
    CONSTRAINT fk_set_id FOREIGN KEY(set_id)
    REFERENCES sets(id)
    ON DELETE CASCADE
);

-- +goose Down
DROP TABLE logs;
