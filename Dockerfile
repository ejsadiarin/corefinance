FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o /app/worker ./cmd/worker
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o /app/migrate ./cmd/migrate

FROM cgr.dev/chainguard/static:latest AS api

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 6969

CMD ["./server"]

FROM cgr.dev/chainguard/static:latest AS worker

WORKDIR /app

COPY --from=builder /app/worker .

CMD ["./worker"]

FROM cgr.dev/chainguard/static:latest AS migrate

WORKDIR /app

COPY --from=builder /app/migrate .

CMD ["./migrate"]
