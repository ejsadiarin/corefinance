# CoreFinance

Financial management microservice for Core. Handles expenses, incomes, categories, tags, recurring rules, and analytics.

It runs behind [coregateway](https://github.com/ejsadiarin/coregateway/blob/main/README.md), which signs every request it
forwards; corefinance never sees a client directly. Two programs ship from
this repo: the API (`cmd/api`, port 6969) and the worker (`cmd/worker`),
which writes rule entries as they fall due (ADR-0001).

## Quick start

**The easy way: run it with everything else.** From a coregateway checkout,
its `local-up` target builds and starts corefinance, its worker, the gateway, the
other services and the dashboard, with their own Postgres. See coregateway's
[Quick start](https://github.com/ejsadiarin/coregateway/blob/main/README.md#quick-start-the-whole-stack-locally).

**Run this repo's code against that stack.** With coregateway's `local-up` target
running, a native corefinance can use the same database and gateway. Put
this in `.env`:

```sh
DATABASE_URL=postgresql://corefinance_role:cfpw@localhost:25432/coredb?sslmode=disable
MIGRATION_URL=postgresql://neondb_owner:owner@localhost:25432/coredb?sslmode=disable&options=-c%20search_path%3Dcorefinance
JWT_JWKS_URL=http://localhost:28080/.well-known/jwks.json
PORT=16969
ENV=development
```

```sh
make run                   # the API on :16969
go run ./cmd/worker        # the worker; add --once for one pass
```

The stack's gateway keeps sending dashboard traffic to its own corefinance
container, so use this to run code against real tables (and for the tests
below), not to replace the container.

## Production prerequisites

- **Database:** the `corefinance` schema and `corefinance_role` in Neon's
  `coredb`, created once with `internal/db/bootstrap.sql`, then the
  migrations. The steps, including the order relative to the gateway, are
  in coregateway's [Setting up Neon](https://github.com/ejsadiarin/coregateway/blob/main/README.md#setting-up-neon).
- **The gateway:** running and reachable at `JWT_JWKS_URL`. corefinance
  refuses to start without its signing keys.
- **`.env`:** copy `.env.example`; `DATABASE_URL` (pooled host,
  `corefinance_role`), `MIGRATION_URL` (direct host, `neondb_owner`) and
  `JWT_JWKS_URL` are required.
- **Two images:** `Dockerfile.corefinance-api` and
  `Dockerfile.corefinance-cron-worker`. Run `make migrate-up` before
  deploying a new image; nothing migrates at start.

No other accounts are needed. The user's timezone comes from the gateway's
token (coregateway ADR-0005).

## Project Structure

```
cmd/api/main.go              — entrypoint
internal/server/              — HTTP server, routes, handlers (remaining/export/import)
internal/expense/             — expense CRUD, search, skip
internal/income/              — income CRUD, skip, occurrences
internal/category/            — the fixed expense/income category lists (ADR-0002)
internal/tag/                 — tags + tag associations
internal/recurring/           — recurring expense/income rules
internal/stats/               — analytics (summary, trends, 50/30/20, velocity)
internal/rates/               — fixed exchange rates (frankfurter.dev/ECB, ADR-0003)
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
| `DATABASE_URL` | Runtime connection: pooled host, `corefinance_role`, no `search_path` |
| `MIGRATION_URL` | Migrations only: direct host, `neondb_owner`, `options=-c search_path=corefinance` |
| `PORT` | Server port (default: `6969`) |
| `ENV` | `development` for debug logs, anything else for production |
| `DB_HOST` | PostgreSQL host (the `DB_*` variables are used only when `DATABASE_URL` is empty) |
| `DB_PORT` | PostgreSQL port |
| `DB_DATABASE` | Database name |
| `DB_USERNAME` | Database user |
| `DB_PASSWORD` | Database password |
| `DB_SCHEMA` | Schema name (default: `corefinance`) |
| `JWT_ISSUER` | Expected internal JWT issuer (default: `https://gateway.internal`) |
| `JWT_AUDIENCE` | Expected internal JWT audience (default: `corefinance`) |
| `JWT_JWKS_URL` | Gateway JWKS URL (default: `https://gateway.internal/.well-known/jwks.json`) |
| `RATE_URL` | Exchange-rate provider base URL (default: `https://api.frankfurter.dev/v1`) |

## API Routes

All routes are prefixed with `/budget`.

| Method | Path | Description |
|---|---|---|
| | `/expense-categories` | The fixed expense categories (GET only) |
| | `/income-categories` | The fixed income categories (GET only) |
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
| | `/current-total-money` | Money on hand, committed, average daily spend and runway |
| | `/period-budget` | Day/week/month budget: funded, spent, bills still due, left, safe to spend per day, previous period |
| | `/cash-flow` | Last N day/week/month buckets: income, spent and net, ending at today |
| | `/trends/month-over-month` | Month-over-month comparison |
| | `/priority-groups` | Reference data (need/want/savings) |
| | `/rule-entries` | Entries recurring rules created (expenses and incomes), newest-inserted first; filters `rule_id`, `start_date`/`end_date`, `created_since` (RFC 3339), paged with `page`/`page_size` |
| | `/rule-entries/feed` | "What's new" feed of rule entries: `{entries, next}`; pass `next` back as `after`, with the same filters (`rule_id`, `start_date`/`end_date`): a cursor is only valid with the filters that produced it. Never skips an entry from a pass that commits late (prefer it over `created_since`) |
| | `/remaining` | Current month budget remaining |
| | `/export` | Export all budget data as JSON |
| | `/import` | Import budget data (with ID remapping) |
| `/` | `/health` | Liveness: `{"status":"up"}`, never touches the database |
| | `/ready` | Readiness: database ping + pool stats; 503 when the ping fails |

## Timezone

Rule entries are dated in the user's timezone, which the gateway owns
(coregateway ADR-0005) and sends in the internal token's `tz` claim. Each
request records the zone its token carries as the user's last-seen zone
(`user_timezones`); a change recomputes the user's rule schedule. A token
without `tz` records nothing and uses the last-seen zone. The worker has no
token: it uses the last-seen zone too. A user no token has carried a zone for
is on UTC. corefinance never calls the gateway.

## Auth

All requests (except listed public paths) require a gateway-minted internal
JWT (`Authorization: Bearer ...`, Ed25519, verified against the gateway
JWKS). `X-User-ID` is stripped on every request and never trusted.

## Development

Go 1.26+, with [sqlc](https://sqlc.dev) and
[goose](https://github.com/pressly/goose) installed. Docker is needed for
the integration tests.

```sh
make run              # the API (make dev: live reload, needs air)
make sqlc             # after editing SQL in internal/db/queries/
make migrate-status   # what MIGRATION_URL has applied
make migrate-up
make help             # every target
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
