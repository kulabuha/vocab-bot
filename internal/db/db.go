package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return db, nil
}

func RunMigrations(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (name TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	// Backfill schema_version for DBs that were migrated before we had tracking (already have chat_id on collocations).
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM schema_version`).Scan(&count)
	if count == 0 {
		var dummy int64
		err := db.QueryRow(`SELECT chat_id FROM collocations LIMIT 1`).Scan(&dummy)
		if err == nil || err == sql.ErrNoRows {
			for _, name := range names {
				_, _ = db.Exec(`INSERT OR IGNORE INTO schema_version (name) VALUES (?)`, name)
			}
			slog.Info("schema_version backfilled for existing DB")
		}
	}

	for _, name := range names {
		var applied int
		if err := db.QueryRow(`SELECT 1 FROM schema_version WHERE name = ?`, name).Scan(&applied); err == nil {
			continue
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := db.Exec(string(body)); err != nil {
			return fmt.Errorf("exec %s: %w", name, err)
		}
		if _, err := db.Exec(`INSERT INTO schema_version (name) VALUES (?)`, name); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		slog.Info("migration applied", "file", name)
	}
	return nil
}
