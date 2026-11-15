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
internal/auth/                — internal JWT verification (JWKS), identity injection
internal/db/sqlc/             — generated sqlc code
internal/db/queries/          — SQL queries
internal/db/migrations/       — goose migrations (source of truth)
internal/db/bootstrap.sql     — one-time role/grants bootstrap (not a migration)
```

## Environment Variables

| Variable | Description |
|---|---|
| `PORT` | Server port (default: `6969`) |
| `ENV` | `development` for debug logs, anything else for production |
| `DB_HOST` | PostgreSQL host |
| `DB_PORT` | PostgreSQL port |
| `DB_DATABASE` | Database name |
| `DB_USERNAME` | Database user |
| `DB_PASSWORD` | Database password |
| `DB_SCHEMA` | Schema name (default: `corefinance`) |
| `JWT_ISSUER` | Expected internal JWT issuer (default: `https://gateway.internal`) |
| `JWT_AUDIENCE` | Expected internal JWT audience (default: `corefinance`) |
| `JWT_JWKS_URL` | Gateway JWKS URL (default: `https://gateway.internal/.well-known/jwks.json`) |
| `GATEWAY_URL` | Worker only, required: the gateway's base URL, for timezone lookups (see Timezone) |
| `GATEWAY_API_KEY` | Worker only, required: an admin-owned gateway API key with the `corefinance:admin` scope |

## API Routes

All routes are prefixed with `/budget`.

| Method | Path | Description |
|---|---|---|
| | `/expense-categories/*` | Expense category CRUD (soft delete) |
| | `/income-categories/*` | Income category CRUD (soft delete) |
| | `/tags/*` | Tag CRUD + tag associations for expenses/incomes |
| | `/expenses/*` | Expense CRUD, search, skip, check-skipped, tag management; `GET /expenses?ids=a,b,c` fetches up to 100 of yours by id, leaving out ids that don't exist or aren't yours |
| | `/incomes/*` | Income CRUD, skip, check-skipped, occurrences, tag management; `GET /incomes?ids=a,b,c` likewise |
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
| | `/rule-entries` | Entries recurring rules created (expenses and incomes), newest-inserted first; filters `rule_id`, `start_date`/`end_date`, `created_since` (RFC 3339), paged with `page`/`page_size` |
| | `/rule-entries/feed` | "What's new" feed of rule entries: `{entries, next}`; pass `next` back as `after`. Never skips an entry from a pass that commits late (prefer it over `created_since`) |
| | `/remaining` | Current month budget remaining |
| | `/export` | Export all budget data as JSON |
| | `/import` | Import budget data (with ID remapping) |
| `/` | `/health` | Health check (pgxpool ping + stats) |

## Timezone

Rule entries are dated in the user's timezone, which the gateway owns
(coregateway ADR-0005). API requests read it from the internal token's `tz`
claim (a token without one is UTC). The worker looks it up from the gateway
(`GET /api/users/{id}/timezone`) with `GATEWAY_URL` and `GATEWAY_API_KEY`: an
API key owned by a gateway admin with only the `corefinance:admin` scope. It
won't start without them, and skips a user whose lookup fails rather than
guess a zone. corefinance keeps only the zone it last saw per user, to
recompute a user's schedule when it changes.

## Auth

All requests (except listed public paths) require a gateway-minted internal
JWT (`Authorization: Bearer ...`, Ed25519, verified against the gateway
JWKS). `X-User-ID` is stripped on every request and never trusted.

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

## Testing

```sh
make test             # everything (unit + integration, needs Docker)
make test-unit        # unit only, no Docker needed
make test-integration # Docker-backed integration only (testcontainers)
make test-race        # unit tests with race detector
make check-refs       # verify README/Makefile references
```

Integration tests spin ephemeral Postgres via testcontainers
(`internal/testutil/testdb.go`).
