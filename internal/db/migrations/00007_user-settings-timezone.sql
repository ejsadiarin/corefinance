-- +goose Up
-- Per-user settings (#9). timezone is the IANA zone (e.g. Asia/Manila) the
-- worker and the API use to decide a user's "today" for rule entries. A user
-- with no row keeps the server clock's date.

CREATE TABLE IF NOT EXISTS user_settings (
    user_id    uuid PRIMARY KEY,
    timezone   text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS user_settings;
