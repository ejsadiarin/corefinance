package testutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

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
				WithStartupTimeout(60),
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

	conn, err := pgxpool.New(ctx, connStr)
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

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func loadSchema() ([]byte, error) {
	paths := []string{
		"../../db/schema.sql",
		"internal/db/schema.sql",
		"../db/schema.sql",
		"db/schema.sql",
	}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("schema.sql not found in any of the expected locations")
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
