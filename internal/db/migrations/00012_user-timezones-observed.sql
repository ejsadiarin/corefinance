-- +goose Up
-- The gateway owns each user's timezone (#18, coregateway ADR-0005).
-- user_settings becomes the zone corefinance last saw per user (from a
-- request's token): background work dates entries in it, and a change
-- recomputes the user's next due times. Existing rows are the zones users saved here before.

ALTER TABLE user_settings RENAME TO user_timezones;
ALTER TABLE user_timezones RENAME COLUMN updated_at TO seen_at;

-- +goose Down
ALTER TABLE user_timezones RENAME COLUMN seen_at TO updated_at;
ALTER TABLE user_timezones RENAME TO user_settings;
