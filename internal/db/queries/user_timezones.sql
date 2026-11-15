-- The zone corefinance last saw per user (#18, coregateway ADR-0005), kept
-- to notice a change and recompute the user's next due times.

-- name: GetSeenTimezone :one
SELECT timezone FROM user_timezones WHERE user_id = sqlc.arg(user_id);

-- Records a zone seen at seen_at and, in the same statement, clears the
-- user's next due times when it differs from the one before (each is
-- midnight in the old zone; the next pass recomputes them). One statement,
-- so a change can't be recorded without its schedule being reset.
-- name: RecordUserTimezone :one
WITH prev AS (
    SELECT timezone FROM user_timezones WHERE user_id = sqlc.arg(owner)::uuid
), recorded AS (
    INSERT INTO user_timezones (user_id, timezone, seen_at)
    VALUES (sqlc.arg(owner)::uuid, sqlc.arg(timezone)::text, sqlc.arg(seen_at)::timestamptz)
    ON CONFLICT (user_id) DO UPDATE SET timezone = EXCLUDED.timezone, seen_at = EXCLUDED.seen_at
    RETURNING user_id
), cleared AS (
    UPDATE rules SET next_due_at = NULL
    WHERE user_id = sqlc.arg(owner)::uuid
      AND (SELECT timezone FROM prev) IS DISTINCT FROM sqlc.arg(timezone)::text
    RETURNING id
)
SELECT ((SELECT timezone FROM prev) IS DISTINCT FROM sqlc.arg(timezone)::text)::bool AS changed;

-- Users with an active rule whose zone hasn't been checked since cutoff: the
-- worker asks the gateway again, so a change there reaches the schedule
-- without the user making a request.
-- name: ListUsersToRefreshZone :many
SELECT DISTINCT r.user_id FROM rules r
LEFT JOIN user_timezones z ON z.user_id = r.user_id
WHERE r.is_active = true
  AND (z.seen_at IS NULL OR z.seen_at < sqlc.arg(cutoff)::timestamptz);
