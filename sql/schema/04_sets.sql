-- +goose Up
CREATE TABLE sets (
    id BIGSERIAL PRIMARY KEY,
    set_order INTEGER NOT NULL,
    rest_time INTEGER,
    session_id UUID NOT NULL,
    exercise_id INTEGER NOT NULL,
    CONSTRAINT fk_session_id FOREIGN KEY (session_id)
    REFERENCES sessions(id)
    ON DELETE CASCADE,
    CONSTRAINT fk_exercise_id FOREIGN KEY (exercise_id)
    REFERENCES exercises(id)
);

-- +goose Down
DROP TABLE sets;
