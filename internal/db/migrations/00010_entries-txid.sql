-- +goose Up
-- The transaction that inserted each entry (#13, ADR-0001). The review feed
-- pages by (txid, id) and never reads past the oldest transaction still in
-- flight, so an entry from a slow pass can't land behind a cursor that
-- already moved on. created_at (transaction start) can't promise that.
-- Existing rows get this migration's transaction id, before any new entry.

ALTER TABLE expenses ADD COLUMN IF NOT EXISTS txid xid8 NOT NULL DEFAULT pg_current_xact_id();
ALTER TABLE incomes ADD COLUMN IF NOT EXISTS txid xid8 NOT NULL DEFAULT pg_current_xact_id();
CREATE INDEX IF NOT EXISTS idx_expenses_user_txid ON expenses (user_id, txid, id) WHERE source_rule_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_incomes_user_txid ON incomes (user_id, txid, id) WHERE source_rule_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_incomes_user_txid;
DROP INDEX IF EXISTS idx_expenses_user_txid;
ALTER TABLE incomes DROP COLUMN IF EXISTS txid;
ALTER TABLE expenses DROP COLUMN IF EXISTS txid;
