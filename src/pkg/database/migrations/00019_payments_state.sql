-- +goose Up
-- Single-row record of whether payments are paused (paywall lifted). It
-- survives restarts so a pause raised before a redeploy is not silently
-- forgotten; the backend re-reads it on startup.
CREATE TABLE payments_state (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    paused BOOLEAN NOT NULL DEFAULT FALSE,
    reason TEXT NOT NULL DEFAULT '',
    detail TEXT NOT NULL DEFAULT '',
    changed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO payments_state (id, paused) VALUES (1, FALSE);

-- +goose Down
DROP TABLE IF EXISTS payments_state;
