-- +goose Up
-- When each rule's next occurrence starts, in the owner's timezone (#16,
-- ADR-0001): Tuesday 00:00 Asia/Manila is Monday 16:00 UTC. The worker fills
-- rules whose time has passed and wakes for the earliest one. NULL means not
-- computed yet (picked up on the next pass); 'infinity' means the rule has
-- ended. materialized_through stays the record of what has been filled: it is
-- a date, so it doesn't move when the owner changes timezone.

ALTER TABLE rules ADD COLUMN IF NOT EXISTS next_due_at timestamptz;
CREATE INDEX IF NOT EXISTS idx_rules_next_due_at ON rules (next_due_at) WHERE is_active;

-- +goose Down
DROP INDEX IF EXISTS idx_rules_next_due_at;
ALTER TABLE rules DROP COLUMN IF EXISTS next_due_at;
