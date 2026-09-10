package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const TestUserID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"

func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("corefinance_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		t.Fatalf("failed to parse connection string: %v", err)
	}
	// search_path must be a per-connection setting: a bare `SET search_path`
	// only affects the single pooled session that runs it, so unqualified
	// queries on other pooled connections would not see the corefinance schema.
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = "corefinance,public"

	conn, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	defer conn.Close()

	_, err = conn.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS corefinance")
	if err != nil {
		t.Fatalf("failed to create corefinance schema: %v", err)
	}

	_, err = conn.Exec(ctx, "SET search_path TO corefinance, public")
	if err != nil {
		t.Fatalf("failed to set search path: %v", err)
	}

	// Enable uuid-ossp extension (built-in) as replacement for pg_uuidv7
	_, err = conn.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	if err != nil {
		t.Fatalf("failed to create uuid-ossp extension: %v", err)
	}

	schemaBytes, err := loadSchema()
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	schema := adaptSchemaForTests(schemaBytes)

	_, err = conn.Exec(ctx, schema)
	if err != nil {
		t.Fatalf("failed to execute schema: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func loadSchema() ([]byte, error) {
	// Single source of truth: the goose migrations directory. Concatenate the
	// Up section of every migration in lexicographic (zero-padded) order,
	// mirroring what `goose up` and `sqlc generate` apply.
	dirs := []string{
		"../../db/migrations",
		"internal/db/migrations",
		"../db/migrations",
		"db/migrations",
	}
	var dir string
	for _, d := range dirs {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			dir = d
			break
		}
	}
	if dir == "" {
		return nil, fmt.Errorf("db/migrations not found in any of the expected locations")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no migration files found in %s", dir)
	}
	var sb strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		// Keep only the Up section: goose Down sections start here.
		up := string(data)
		if idx := strings.Index(up, "-- +goose Down"); idx >= 0 {
			up = up[:idx]
		}
		sb.WriteString(up)
		sb.WriteString("\n")
	}
	return []byte(sb.String()), nil
}

func adaptSchemaForTests(schema []byte) string {
	s := string(schema)
	s = strings.ReplaceAll(s, `CREATE EXTENSION IF NOT EXISTS pg_uuidv7`, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	s = strings.ReplaceAll(s, `uuid_generate_v7()`, `uuid_generate_v4()`)
	return s
}

func StrPtr(s string) *string {
	return &s
}

func TestUserIDUUID() uuid.UUID {
	return uuid.MustParse(TestUserID)
}
