FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o /app/worker ./cmd/worker
# Pinned goose CLI for the migrate stage. Keep in sync with the
# `tool github.com/pressly/goose/v3/cmd/goose` directive in go.mod.
RUN --mount=type=cache,target=/go/pkg/mod GOBIN=/app/bin go install github.com/pressly/goose/v3/cmd/goose@v3.28.0

FROM cgr.dev/chainguard/static:latest AS api

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 6969

CMD ["./server"]

FROM cgr.dev/chainguard/static:latest AS worker

WORKDIR /app

COPY --from=builder /app/worker .

CMD ["./worker"]

# Short-lived migrate job: stock goose CLI over the migrations dir.
# Runs as the owner on a DIRECT (unpooled) URL. Owner runs need explicit
# schema scope (owner has no search_path default); app-role runs already
# carry it — appended only when absent.
FROM golang:alpine AS migrate

WORKDIR /app

COPY --from=builder /app/bin/goose /usr/local/bin/goose
COPY --from=builder /app/internal/db/migrations ./migrations

CMD sh -c 'case "$DATABASE_URL" in *search_path*) ;; *) case "$DATABASE_URL" in *"?"*) S="&";; *) S="?";; esac; export DATABASE_URL="$DATABASE_URL${S}options=-c%20search_path%3Dcorefinance";; esac; exec goose -dir ./migrations postgres "$DATABASE_URL" up'
