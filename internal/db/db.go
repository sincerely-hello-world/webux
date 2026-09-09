package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

//go:embed migrations/001_init.sql
var migration001 string

//go:embed migrations/002_ai.sql
var migration002 string

//go:embed migrations/003_configmgmt.sql
var migration003 string

//go:embed migrations/004_terminal.sql
var migration004 string

//go:embed migrations/005_ansible_ai.sql
var migration005 string

//go:embed migrations/006_auth.sql
var migration006 string

//go:embed migrations/007_health.sql
var migration007 string

//go:embed migrations/008_hardening.sql
var migration008 string

//go:embed migrations/009_sessions.sql
var migration009 string

//go:embed migrations/010_fix_sessions.sql
var migration010 string

// Open opens (or creates) the Webux SQLite database.
func Open(path string) (*sql.DB, error) {
	// The driver is github.com/ncruces/go-sqlite3, whose DSN syntax is NOT
	// mattn/go-sqlite3's: it only reads options when the name starts with
	// "file:", and PRAGMAs are spelled _pragma=name(value).
	//
	// The previous DSN used mattn's "_journal/_timeout/_fk" parameters, which
	// this driver does not recognise at all. They were silently taken as part
	// of the file name, so it created a literal "webux.db?_journal=WAL&..."
	// file and WAL, the busy timeout and foreign key enforcement were never
	// actually enabled — with no error to show for it.
	//
	// PRAGMA order matters: busy_timeout and the locking-mode pragmas must be
	// applied first (see the driver documentation).
	dsn := "file:" + path +
		"?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite: one writer at a time
	return db, nil
}

// Migrate applies all pending migrations idempotently.
func Migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version     INTEGER PRIMARY KEY,
		applied_at  TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	migrations := []struct {
		version int
		sql     string
	}{
		{1, migration001},
		{2, migration002},
		{3, migration003},
		{4, migration004},
		{5, migration005},
		{6, migration006},
		{7, migration007},
		{8, migration008},
		{9, migration009},
		{10, migration010},
	}

	for _, m := range migrations {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", m.version).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("migration %d: %w", m.version, err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
	}
	return nil
}

// RunSetup creates initial admin user and default settings.
func RunSetup(db *sql.DB) error {
	if err := Migrate(db); err != nil {
		return err
	}
	// Insert default settings — idempotent
	settings := map[string]string{
		"learn_mode_enabled": "true",
		"ai_enabled":         "false",
	}
	for k, v := range settings {
		if _, err := db.Exec(
			`INSERT OR IGNORE INTO webux_settings (key, value) VALUES (?, ?)`, k, v,
		); err != nil {
			return err
		}
	}
	return nil
}
