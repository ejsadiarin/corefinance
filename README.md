# CoreFinance

Financial management microservice for Core. Handles expenses, incomes, categories, tags, recurring rules, and analytics.

## Project Structure

```
cmd/api/main.go              — entrypoint
internal/server/              — HTTP server, routes, handlers (remaining/export/import)
internal/expense/             — expense CRUD, search, skip
internal/income/              — income CRUD, skip, occurrences
internal/category/            — expense/income categories (soft delete)
internal/tag/                 — tags + tag associations
internal/recurring/           — recurring expense/income rules
internal/stats/               — analytics (summary, trends, 50/30/20, velocity)
internal/helper/              — shared utilities
internal/auth/                — user ID extraction from headers
internal/db/sqlc/             — generated sqlc code
internal/db/queries/          — SQL queries
internal/db/schema.sql        — database schema
```

## Environment Variables

| Variable | Description |
|---|---|
| `PORT` | Server port (default: `8080`) |
| `ENV` | `development` for debug logs, anything else for production |
| `DB_HOST` | PostgreSQL host |
| `DB_PORT` | PostgreSQL port |
| `DB_DATABASE` | Database name |
| `DB_USERNAME` | Database user |
| `DB_PASSWORD` | Database password |
| `DB_SCHEMA` | Schema name (default: `corefinance`) |

## API Routes

All routes are prefixed with `/budget`.

| Method | Path | Description |
|---|---|---|
| | `/expense-categories/*` | Expense category CRUD (soft delete) |
| | `/income-categories/*` | Income category CRUD (soft delete) |
| | `/tags/*` | Tag CRUD + tag associations for expenses/incomes |
| | `/expenses/*` | Expense CRUD, search, skip, check-skipped, tag management |
| | `/incomes/*` | Income CRUD, skip, check-skipped, occurrences, tag management |
| | `/recurring-expenses/*` | Recurring expense rule CRUD |
| | `/recurring-incomes/*` | Recurring income rule CRUD |
| | `/stats/summary` | Financial summary |
| | `/stats/trends` | Trend data |
| | `/stats/category-breakdown` | Breakdown by category |
| | `/stats/savings-rate` | Savings rate |
| | `/analysis/503020` | 50/30/20 analysis by priority |
| | `/velocity` | Average monthly spending |
| | `/forecast/upcoming` | Upcoming recurring expenses |
| | `/current-total-money` | Total income minus total expenses |
| | `/trends/month-over-month` | Month-over-month comparison |
| | `/priority-groups` | Reference data (need/want/savings) |
| | `/remaining` | Current month budget remaining |
| | `/export` | Export all budget data as JSON |
| | `/import` | Import budget data (with ID remapping) |
| `/` | `/health` | Health check (pgxpool ping + stats) |

## Auth

All requests require the `X-User-ID` header, set by the coregateway.

## Requirements

- Go 1.26+
- PostgreSQL with `corefinance` schema
- [sqlc](https://sqlc.dev) for code generation

## Development

```sh
# Run
go run cmd/api/main.go

# Generate sqlc code
sqlc generate
```
