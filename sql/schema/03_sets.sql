-- +goose Up
CREATE TABLE sets (
    id BIGSERIAL PRIMARY KEY,
    set_order INTEGER NOT NULL,
    rest_time INTEGER,
    session_id UUID NOT NULL,
    CONSTRAINT fk_session_id FOREIGN KEY (session_id)
    REFERENCES sessions(id)
    ON DELETE CASCADE
);

-- +goose Down
DROP TABLE sets;
