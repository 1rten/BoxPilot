package store

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const schemaMigrationsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);`

// DB wraps *sql.DB and provides migration.
type DB struct {
	*sql.DB
}

// Open opens SQLite at path and runs migrations.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	d := &DB{DB: db}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) migrate() error {
	if _, err := d.Exec(schemaMigrationsTable); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	var versions []int
	for _, e := range entries {
		if e.IsDir() {
			continue
	}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		ver, err := strconv.Atoi(name[:4])
		if err != nil || len(name) < 5 {
			continue
		}
		versions = append(versions, ver)
	}
	sort.Ints(versions)

	for _, ver := range versions {
		var done int
		err := d.QueryRow("SELECT version FROM schema_migrations WHERE version = ?", ver).Scan(&done)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		prefix := fmt.Sprintf("%04d_", ver)
		var fname string
		for _, e := range entries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".sql") {
				fname = e.Name()
				break
			}
		}
		if fname == "" {
			continue
		}
		body, err := migrationsFS.ReadFile("migrations/" + fname)
		if err != nil {
			return errorx.New(errorx.DBMigrationFailed, "read migration").WithDetails(map[string]any{"version": ver, "err": err.Error()})
		}
		tx, err := d.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			tx.Rollback()
			return errorx.New(errorx.DBMigrationFailed, "run migration").WithDetails(map[string]any{"version": ver, "err": err.Error()})
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)", ver, util.NowRFC3339()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
