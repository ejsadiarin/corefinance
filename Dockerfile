FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api

FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 6969

CMD ["./server"]
