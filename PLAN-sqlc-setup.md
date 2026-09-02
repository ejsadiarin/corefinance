# Plan: sqlc Schema + Queries Setup

## Goal
Set up sqlc for the corefinance service: extract the corefinance schema, configure sqlc, and write CRUD + basic aggregation queries for all existing tables.

## Scope
- **Include:** expenses, incomes, expense_categories, income_categories, tags, expense_tags, income_tags, recurring_expense_rules, recurring_income_rules
- **Exclude:** goose_db_version (migration tracking), coregateway tables, category-budgets (table doesn't exist), subscriptions (table doesn't exist)
- **Queries:** CRUD operations + search/skip/check-skipped for expenses/incomes + basic aggregations (SUM, COUNT by category)

---

## Step 1: Extract corefinance schema

Create `internal/db/schema.sql` by extracting only `corefinance.*` CREATE TABLE statements from `20260718-coredb.sql`.

**Tables to extract (lines from dump):**
- `expense_categories` (line 62)
- `expense_tags` (line 80)
- `expenses` (line 92)
- `income_categories` (line 152)
- `income_tags` (line 170)
- `incomes` (line 182)
- `recurring_expense_rules` (line 213)
- `recurring_income_rules` (line 241)
- `tags` (line 264)

**Also extract:**
- The `pg_uuidv7` extension creation (line 44)
- All ALTER TABLE / CONSTRAINT / INDEX statements for these tables
- All sequence definitions for these tables

**Skip:** goose_db_version, coregateway.* tables, coreban.* schema

---

## Step 2: Create sqlc.yaml

Location: `/home/ejs/personal/coregateway/services/corefinance/sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/db/queries"
    schema: "internal/db/schema.sql"
    gen:
      go:
        package: "db"
        out: "internal/db/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_db_tags: true
        emit_empty_slices: true
        overrides:
          - db_type: "uuid"
            go_type: "github.com/google/uuid.UUID"
          - db_type: "uuid"
            go_type: "*github.com/google/uuid.UUID"
            nullable: true
```

---

## Step 3: Create query files

### 3.1 `internal/db/queries/categories.sql`

| Query | SQL Command | Description |
|-------|-------------|-------------|
| `CreateExpenseCategory` | INSERT | Create new expense category |
| `ListExpenseCategories` | SELECT | List all active expense categories for a user |
| `UpdateExpenseCategory` | UPDATE | Update expense category by ID |
| `DeleteExpenseCategory` | DELETE | Soft-delete (set is_active=false) or hard delete |
| `CreateIncomeCategory` | INSERT | Create new income category |
| `ListIncomeCategories` | SELECT | List all active income categories for a user |
| `UpdateIncomeCategory` | UPDATE | Update income category by ID |
| `DeleteIncomeCategory` | DELETE | Soft-delete or hard delete |

### 3.2 `internal/db/queries/tags.sql`

| Query | SQL Command | Description |
|-------|-------------|-------------|
| `CreateTag` | INSERT | Create new tag |
| `ListTags` | SELECT | List all tags for a user |
| `UpdateTag` | UPDATE | Update tag by ID |
| `DeleteTag` | DELETE | Delete tag by ID |

### 3.3 `internal/db/queries/expenses.sql`

| Query | SQL Command | Description |
|-------|-------------|-------------|
| `CreateExpense` | INSERT | Create new expense |
| `GetExpense` | SELECT | Get expense by ID with category and tags |
| `ListExpenses` | SELECT | List expenses with pagination, filters (date range, category, priority, status) |
| `SearchExpenses` | SELECT | Full-text search on description/notes |
| `UpdateExpense` | UPDATE | Update expense by ID |
| `DeleteExpense` | DELETE | Delete expense by ID |
| `SkipExpense` | UPDATE | Set status='skipped' |
| `CheckSkippedExpense` | SELECT | Check if expense is skipped for a date range |
| `GetExpenseStatsByCategory` | SELECT | SUM/COUNT grouped by category (basic aggregation) |
| `GetTotalExpensesByDateRange` | SELECT | SUM of expenses in a date range |

### 3.4 `internal/db/queries/incomes.sql`

| Query | SQL Command | Description |
|-------|-------------|-------------|
| `CreateIncome` | INSERT | Create new income |
| `GetIncome` | SELECT | Get income by ID with category and tags |
| `ListIncomes` | SELECT | List incomes with pagination, filters (date range, category, status) |
| `UpdateIncome` | UPDATE | Update income by ID |
| `DeleteIncome` | DELETE | Delete income by ID |
| `SkipIncome` | UPDATE | Set status='skipped' |
| `CheckSkippedIncome` | SELECT | Check if income is skipped for a date range |
| `GetIncomeOccurrences` | SELECT | Get occurrences of recurring income in a date range |
| `GetTotalIncomesByDateRange` | SELECT | SUM of incomes in a date range |

### 3.5 `internal/db/queries/expense_tags.sql`

| Query | SQL Command | Description |
|-------|-------------|-------------|
| `AddTagToExpense` | INSERT | Link tag to expense |
| `RemoveTagFromExpense` | DELETE | Unlink tag from expense |
| `GetTagsByExpenseID` | SELECT | Get all tags for an expense |

### 3.6 `internal/db/queries/income_tags.sql`

| Query | SQL Command | Description |
|-------|-------------|-------------|
| `AddTagToIncome` | INSERT | Link tag to income |
| `RemoveTagFromIncome` | DELETE | Unlink tag from income |
| `GetTagsByIncomeID` | SELECT | Get all tags for an income |

### 3.7 `internal/db/queries/recurring_expense_rules.sql`

| Query | SQL Command | Description |
|-------|-------------|-------------|
| `CreateRecurringExpenseRule` | INSERT | Create new rule |
| `ListRecurringExpenseRules` | SELECT | List active rules for a user |
| `GetRecurringExpenseRule` | SELECT | Get rule by ID |
| `UpdateRecurringExpenseRule` | UPDATE | Update rule by ID |
| `DeleteRecurringExpenseRule` | DELETE | Delete rule by ID |

### 3.8 `internal/db/queries/recurring_income_rules.sql`

| Query | SQL Command | Description |
|-------|-------------|-------------|
| `CreateRecurringIncomeRule` | INSERT | Create new rule |
| `ListRecurringIncomeRules` | SELECT | List active rules for a user |
| `GetRecurringIncomeRule` | SELECT | Get rule by ID |
| `UpdateRecurringIncomeRule` | UPDATE | Update rule by ID |
| `DeleteRecurringIncomeRule` | DELETE | Delete rule by ID |

---

## Step 4: Generate sqlc code

Run:
```bash
cd /home/ejs/personal/coregateway/services/corefinance
sqlc generate
```

This will produce:
- `internal/db/sqlc/models.go` - Go structs for all tables
- `internal/db/sqlc/querier.go` - Query interface
- `internal/db/sqlc/*.sql.go` - Generated query implementations

---

## Step 5: Verify

- Run `go build ./...` to verify generated code compiles
- Run `go vet ./...` for static analysis

---

## Files Created/Modified

| File | Action |
|------|--------|
| `internal/db/schema.sql` | **Create** - Extracted corefinance schema |
| `sqlc.yaml` | **Create** - sqlc configuration |
| `internal/db/queries/categories.sql` | **Create** - Category CRUD queries |
| `internal/db/queries/tags.sql` | **Create** - Tag CRUD queries |
| `internal/db/queries/expenses.sql` | **Create** - Expense CRUD + search + stats queries |
| `internal/db/queries/incomes.sql` | **Create** - Income CRUD + skip + occurrences queries |
| `internal/db/queries/expense_tags.sql` | **Create** - Expense-tag junction queries |
| `internal/db/queries/income_tags.sql` | **Create** - Income-tag junction queries |
| `internal/db/queries/recurring_expense_rules.sql` | **Create** - Recurring expense rule CRUD |
| `internal/db/queries/recurring_income_rules.sql` | **Create** - Recurring income rule CRUD |
| `internal/db/sqlc/` | **Generated** - sqlc output (models.go, querier.go, *.sql.go) |

---

## Notes

- All queries use `$1, $2, ...` positional params (pgx style)
- All list queries filter by `user_id` (multi-tenant)
- `DeleteExpenseCategory`/`DeleteIncomeCategory` use soft delete (`is_active = false`) to match the schema design
- `DeleteTag` is hard delete since tags are lightweight reference data
- Expense/income list queries support optional filters via sqlc `:onig` or `IFNULL` patterns
- The `priority_groups` endpoint returns hardcoded `['need', 'want', 'savings']` - no SQL needed, handled in Go handler
