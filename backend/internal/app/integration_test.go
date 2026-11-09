package app

import (
	"context"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"hooklab/migrations"
	"os"
	"testing"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_DSN to run MySQL integration tests")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = migrations.Run(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}
func TestMigrationIdempotent(t *testing.T) {
	db := testDB(t)
	if err := migrations.Run(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&n); err != nil || n != 1 {
		t.Fatalf("versions %d: %v", n, err)
	}
}

// Tests below use the real router and MySQL; no repository mocks.
