-- The zone corefinance last saw per user (#18, coregateway ADR-0005), kept
-- to notice a change and recompute the user's next due times.

-- From a request's token: writes only when the zone is new or changed, and
-- then returns the user's row, so callers know to recompute.
-- name: ObserveUserTimezone :many
INSERT INTO user_timezones (user_id, timezone, seen_at)
VALUES (sqlc.arg(user_id), sqlc.arg(timezone), sqlc.arg(seen_at)::timestamptz)
ON CONFLICT (user_id) DO UPDATE SET timezone = EXCLUDED.timezone, seen_at = EXCLUDED.seen_at
WHERE user_timezones.timezone IS DISTINCT FROM EXCLUDED.timezone
RETURNING user_id;

-- Users with an active rule whose zone hasn't been checked since cutoff: the
-- worker asks the gateway again, so a change there reaches the schedule
-- without the user making a request.
-- name: ListUsersToRefreshZone :many
SELECT DISTINCT r.user_id FROM rules r
LEFT JOIN user_timezones z ON z.user_id = r.user_id
WHERE r.is_active = true
  AND (z.seen_at IS NULL OR z.seen_at < sqlc.arg(cutoff)::timestamptz);

-- From the gateway: always records the check, and returns the zone before
-- it ('' when there was none).
-- name: RefreshUserTimezone :one
WITH prev AS (SELECT timezone FROM user_timezones WHERE user_id = sqlc.arg(user_id))
INSERT INTO user_timezones (user_id, timezone, seen_at)
VALUES (sqlc.arg(user_id), sqlc.arg(timezone), sqlc.arg(seen_at)::timestamptz)
ON CONFLICT (user_id) DO UPDATE SET timezone = EXCLUDED.timezone, seen_at = EXCLUDED.seen_at
RETURNING COALESCE((SELECT timezone FROM prev), '')::text AS previous;
