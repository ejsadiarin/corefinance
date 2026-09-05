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
| `APP_ENV` | `development` for debug logs, anything else for production |
| `BLUEPRINT_DB_HOST` | PostgreSQL host |
| `BLUEPRINT_DB_PORT` | PostgreSQL port |
| `BLUEPRINT_DB_DATABASE` | Database name |
| `BLUEPRINT_DB_USERNAME` | Database user |
| `BLUEPRINT_DB_PASSWORD` | Database password |
| `BLUEPRINT_DB_SCHEMA` | Schema name (default: `corefinance`) |

## API Routes

All routes are prefixed with `/budget`.

| Method | Path | Description |
|---|---|---|
| | `/category/*` | Expense/income category CRUD |
| | `/tag/*` | Tag CRUD + associations |
| | `/expense/*` | Expense CRUD, search, skip, check-skipped |
| | `/income/*` | Income CRUD, skip, check-skipped, occurrences |
| | `/recurring/*` | Recurring expense/income rule CRUD |
| | `/stats/summary` | Financial summary |
| | `/stats/trends` | Trend data |
| | `/stats/category-breakdown` | Breakdown by category |
| | `/stats/savings-rate` | Savings rate |
| | `/analysis/50-30-20` | 50/30/20 analysis |
| | `/analytics/velocity` | Spending velocity |
| | `/analytics/forecast` | Financial forecast |
| | `/analytics/current-total-money` | Current total |
| | `/analytics/month-over-month` | Month-over-month comparison |
| | `/budget/remaining` | Remaining budget |
| | `/budget/export` | Export data |
| | `/budget/import` | Import data |

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
