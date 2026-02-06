CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    current_rev BIGINT NOT NULL DEFAULT 0
);

CREATE TYPE item_type AS ENUM ('login/pass', 'text', 'binary', 'card');

CREATE TABLE items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    type item_type NOT NULL,
    ciphertext BYTEA,
    nonce BYTEA NOT NULL,
    aad BYTEA NULL,
    deleted BOOLEAN NOT NULL DEFAULT false,
    updated_rev BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_items_updated_rev ON items(user_id, updated_rev);
CREATE INDEX idx_items_deleted_false ON items(user_id, deleted) WHERE deleted = false;