-- +goose Up
CREATE TABLE IF NOT EXISTS collections (
    type    TEXT PRIMARY KEY,
    schema  JSONB NOT NULL DEFAULT '{}',
    weights JSONB NOT NULL DEFAULT '{}'
);

-- +goose Down
DROP TABLE IF EXISTS collections;
