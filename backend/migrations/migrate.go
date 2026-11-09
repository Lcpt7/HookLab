package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
)

//go:embed *.sql
var files embed.FS

func Run(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var locked int
	if err = conn.QueryRowContext(ctx, "SELECT GET_LOCK('hooklab_migrations', 30)").Scan(&locked); err != nil || locked != 1 {
		return fmt.Errorf("migration lock: %v", err)
	}
	defer conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK('hooklab_migrations')")
	if _, err = conn.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(100) PRIMARY KEY, applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"); err != nil {
		return err
	}
	entries, _ := files.ReadDir(".")
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		var count int
		if err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version=?", e.Name()).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		b, _ := files.ReadFile(e.Name())
		for _, statement := range strings.Split(string(b), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err = conn.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migration %s: %w", e.Name(), err)
			}
		}
		if _, err = conn.ExecContext(ctx, "INSERT INTO schema_migrations(version) VALUES(?)", e.Name()); err != nil {
			return err
		}
	}
	return nil
}
