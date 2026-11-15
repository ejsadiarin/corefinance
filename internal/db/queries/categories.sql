-- Categories are one fixed, system-defined list per kind (ADR-0002): rows
-- with no owner, seeded by migration 00015. Owners can't write them.

-- name: ListExpenseCategories :many
SELECT * FROM expense_categories
WHERE user_id IS NULL AND is_active = true
ORDER BY id;

-- name: ListIncomeCategories :many
SELECT * FROM income_categories
WHERE user_id IS NULL AND is_active = true
ORDER BY id;
