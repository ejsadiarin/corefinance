-- +goose Up
-- Categories become one fixed, system-defined list per kind (#23, ADR-0002).
-- System rows have no owner and well-known ids that internal/category names
-- too. Every record and rule moves to the system category with the same name
-- (ignoring case and surrounding spaces), or to `other` with a tag of the old
-- name, and the per-owner rows go. Not reversible: the Down section restores
-- only the constraints.

ALTER TABLE expense_categories ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE income_categories ALTER COLUMN user_id DROP NOT NULL;

INSERT INTO expense_categories (id, user_id, name, color, icon) VALUES
    ('00000000-0000-4000-8000-000000000101', NULL, 'food', '#f97316', '🍜'),
    ('00000000-0000-4000-8000-000000000102', NULL, 'transportation', '#3b82f6', '🚌'),
    ('00000000-0000-4000-8000-000000000103', NULL, 'gifts', '#ec4899', '🎁'),
    ('00000000-0000-4000-8000-000000000104', NULL, 'subscriptions', '#8b5cf6', '🔁'),
    ('00000000-0000-4000-8000-000000000105', NULL, 'bills & installments', '#ef4444', '🧾'),
    ('00000000-0000-4000-8000-000000000106', NULL, 'personal care', '#14b8a6', '🧴'),
    ('00000000-0000-4000-8000-000000000107', NULL, 'entertainment', '#eab308', '🎬'),
    ('00000000-0000-4000-8000-000000000108', NULL, 'travel', '#0ea5e9', '✈️'),
    ('00000000-0000-4000-8000-000000000109', NULL, 'government & school', '#64748b', '🏛️'),
    ('00000000-0000-4000-8000-000000000110', NULL, 'shopping', '#d946ef', '🛍️'),
    ('00000000-0000-4000-8000-000000000111', NULL, 'communication', '#22c55e', '📱'),
    ('00000000-0000-4000-8000-000000000112', NULL, 'other', '#94a3b8', '📦')
ON CONFLICT (id) DO NOTHING;

INSERT INTO income_categories (id, user_id, name, color, icon) VALUES
    ('00000000-0000-4000-8000-000000000201', NULL, 'allowance', '#22c55e', '💵'),
    ('00000000-0000-4000-8000-000000000202', NULL, 'gift', '#ec4899', '🎁'),
    ('00000000-0000-4000-8000-000000000203', NULL, 'salary', '#3b82f6', '💼'),
    ('00000000-0000-4000-8000-000000000204', NULL, 'reimbursement', '#14b8a6', '↩️'),
    ('00000000-0000-4000-8000-000000000205', NULL, 'other', '#94a3b8', '📦')
ON CONFLICT (id) DO NOTHING;

-- 1. Records under an owner category with no system match get a tag of that
--    name: the owner's existing tag (same name ignoring case and spaces), or
--    a new one.
INSERT INTO tags (user_id, name)
SELECT DISTINCT ON (owner, lower(name)) owner, name
FROM (
    SELECT e.user_id AS owner, btrim(c.name) AS name
    FROM expenses e
    JOIN expense_categories c ON c.id = e.category_id AND c.user_id IS NOT NULL
    WHERE NOT EXISTS (SELECT 1 FROM expense_categories s
                      WHERE s.user_id IS NULL AND s.name = lower(btrim(c.name)))
    UNION ALL
    SELECT i.user_id, btrim(c.name)
    FROM incomes i
    JOIN income_categories c ON c.id = i.category_id AND c.user_id IS NOT NULL
    WHERE NOT EXISTS (SELECT 1 FROM income_categories s
                      WHERE s.user_id IS NULL AND s.name = lower(btrim(c.name)))
) unmatched
WHERE btrim(name) <> ''
  AND NOT EXISTS (SELECT 1 FROM tags t
                  WHERE t.user_id = unmatched.owner AND lower(btrim(t.name)) = lower(unmatched.name))
ORDER BY owner, lower(name), name;

INSERT INTO expense_tags (expense_id, tag_id)
SELECT e.id,
       (SELECT t.id FROM tags t
        WHERE t.user_id = e.user_id AND lower(btrim(t.name)) = lower(btrim(c.name))
        ORDER BY t.created_at, t.id LIMIT 1)
FROM expenses e
JOIN expense_categories c ON c.id = e.category_id AND c.user_id IS NOT NULL
WHERE btrim(c.name) <> ''
  AND NOT EXISTS (SELECT 1 FROM expense_categories s
                  WHERE s.user_id IS NULL AND s.name = lower(btrim(c.name)))
ON CONFLICT DO NOTHING;

INSERT INTO income_tags (income_id, tag_id)
SELECT i.id,
       (SELECT t.id FROM tags t
        WHERE t.user_id = i.user_id AND lower(btrim(t.name)) = lower(btrim(c.name))
        ORDER BY t.created_at, t.id LIMIT 1)
FROM incomes i
JOIN income_categories c ON c.id = i.category_id AND c.user_id IS NOT NULL
WHERE btrim(c.name) <> ''
  AND NOT EXISTS (SELECT 1 FROM income_categories s
                  WHERE s.user_id IS NULL AND s.name = lower(btrim(c.name)))
ON CONFLICT DO NOTHING;

-- 2. Move records and rules to the system category of the same name, else
--    `other`.
UPDATE expenses e
SET category_id = COALESCE(
    (SELECT s.id FROM expense_categories s WHERE s.user_id IS NULL AND s.name = lower(btrim(c.name))),
    '00000000-0000-4000-8000-000000000112')
FROM expense_categories c
WHERE c.id = e.category_id AND c.user_id IS NOT NULL;

UPDATE incomes i
SET category_id = COALESCE(
    (SELECT s.id FROM income_categories s WHERE s.user_id IS NULL AND s.name = lower(btrim(c.name))),
    '00000000-0000-4000-8000-000000000205')
FROM income_categories c
WHERE c.id = i.category_id AND c.user_id IS NOT NULL;

UPDATE rules r
SET category_id = COALESCE(
    (SELECT s.id FROM expense_categories s WHERE s.user_id IS NULL AND s.name = lower(btrim(c.name))),
    '00000000-0000-4000-8000-000000000112')
FROM expense_categories c
WHERE r.kind = 'expense' AND c.id = r.category_id AND c.user_id IS NOT NULL;

UPDATE rules r
SET category_id = COALESCE(
    (SELECT s.id FROM income_categories s WHERE s.user_id IS NULL AND s.name = lower(btrim(c.name))),
    '00000000-0000-4000-8000-000000000205')
FROM income_categories c
WHERE r.kind = 'income' AND c.id = r.category_id AND c.user_id IS NOT NULL;

-- 3. An id that pointed at no category of the record's kind showed no
--    category before; it shows none now, explicitly.
UPDATE expenses SET category_id = NULL
WHERE category_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM expense_categories s WHERE s.id = expenses.category_id AND s.user_id IS NULL);
UPDATE incomes SET category_id = NULL
WHERE category_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM income_categories s WHERE s.id = incomes.category_id AND s.user_id IS NULL);
UPDATE rules SET category_id = NULL
WHERE kind = 'expense' AND category_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM expense_categories s WHERE s.id = rules.category_id AND s.user_id IS NULL);
UPDATE rules SET category_id = NULL
WHERE kind = 'income' AND category_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM income_categories s WHERE s.id = rules.category_id AND s.user_id IS NULL);

-- 4. The per-owner rows go, and can't come back.
DELETE FROM expense_categories WHERE user_id IS NOT NULL;
DELETE FROM income_categories WHERE user_id IS NOT NULL;
ALTER TABLE expense_categories ADD CONSTRAINT expense_categories_system_only CHECK (user_id IS NULL);
ALTER TABLE income_categories ADD CONSTRAINT income_categories_system_only CHECK (user_id IS NULL);
ALTER TABLE expenses ADD CONSTRAINT expenses_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES expense_categories(id);
ALTER TABLE incomes ADD CONSTRAINT incomes_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES income_categories(id);

-- +goose Down
-- Records stay on the system categories, which stay too; up again is safe.
ALTER TABLE incomes DROP CONSTRAINT IF EXISTS incomes_category_id_fkey;
ALTER TABLE expenses DROP CONSTRAINT IF EXISTS expenses_category_id_fkey;
ALTER TABLE income_categories DROP CONSTRAINT IF EXISTS income_categories_system_only;
ALTER TABLE expense_categories DROP CONSTRAINT IF EXISTS expense_categories_system_only;
