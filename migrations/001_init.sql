-- +goose Up
-- +goose envsub on

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS vecsim_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS items (
    id         TEXT PRIMARY KEY,
    label      TEXT        NOT NULL,
    type       TEXT        NOT NULL,
    fields     JSONB       NOT NULL DEFAULT '{}',
    tags       TEXT[]      NOT NULL DEFAULT '{}',
    embedding  vector(${VECSIM_DIMENSIONS}),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS items_type_idx           ON items (type);
CREATE INDEX IF NOT EXISTS items_embedding_hnsw_idx ON items USING hnsw (embedding vector_cosine_ops);

-- +goose envsub off

-- +goose Down
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS vecsim_meta;
