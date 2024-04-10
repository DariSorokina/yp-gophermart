-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS content;

CREATE TABLE IF NOT EXISTS content.users (
    user_login TEXT NOT NULL UNIQUE, 
    encrypted_password TEXT NOT NULL,
    user_id SERIAL PRIMARY KEY);

CREATE TABLE IF NOT EXISTS content.orders (
    user_id INTEGER NOT NULL REFERENCES content.users (user_id) ON DELETE CASCADE,
    order_id TEXT,
    accrual FLOAT,
    status TEXT,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
    flag TEXT CHECK (flag IN ('accrue', 'withdraw')));

CREATE UNIQUE INDEX order_flag_idx ON content.orders (order_id, flag);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS content.orders;

DROP TABLE IF EXISTS content.users;
-- +goose StatementEnd
