package database

import (
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
)

// RunMigrations discovers and applies pending SQL migrations from the
// embedded filesystem. It uses a schema_migrations table to track which
// files have been applied. On an existing database (detected by the
// presence of the "tickers" table), all current migration files are
// seeded as already applied to avoid re-running them.
//
// A PostgreSQL advisory lock prevents concurrent pods from racing.
func RunMigrations(migrationsFS fs.FS) error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// Acquire advisory lock to prevent concurrent migration runs
	// (e.g., two pods starting simultaneously in production).
	// The lock is session-scoped and auto-released on disconnect.
	_, err := DB.Exec("SELECT pg_advisory_lock(1001001001)")
	if err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	defer DB.Exec("SELECT pg_advisory_unlock(1001001001)") //nolint:errcheck

	if err := ensureMigrationsTable(); err != nil {
		return err
	}

	allFiles, err := discoverMigrations(migrationsFS)
	if err != nil {
		return err
	}
	if len(allFiles) == 0 {
		log.Println("No migration files found")
		return nil
	}

	applied, err := getAppliedMigrations()
	if err != nil {
		return err
	}

	// Bootstrap: if schema_migrations is empty and database already has
	// tables (existing production DB), seed all filenames as applied.
	if len(applied) == 0 {
		existing, err := hasExistingTables()
		if err != nil {
			return err
		}
		if existing {
			log.Printf("Bootstrapping: seeding %d existing migrations", len(allFiles))
			return seedMigrations(allFiles)
		}
	}

	pending := findPending(allFiles, applied)
	if len(pending) == 0 {
		log.Println("No pending migrations")
		return nil
	}

	log.Printf("Found %d pending migration(s)", len(pending))
	for _, filename := range pending {
		if err := executeMigration(migrationsFS, filename); err != nil {
			return fmt.Errorf("migration %s failed: %w", filename, err)
		}
		log.Printf("Applied migration: %s", filename)
	}

	return nil
}

func ensureMigrationsTable() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}

func hasExistingTables() (bool, error) {
	var exists bool
	err := DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'tickers'
		)
	`).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check for existing tables: %w", err)
	}
	return exists, nil
}

func discoverMigrations(migrationsFS fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(files)
	return files, nil
}

func getAppliedMigrations() (map[string]bool, error) {
	rows, err := DB.Query("SELECT filename FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		applied[filename] = true
	}
	return applied, rows.Err()
}

func findPending(allFiles []string, applied map[string]bool) []string {
	var pending []string
	for _, f := range allFiles {
		if !applied[f] {
			pending = append(pending, f)
		}
	}
	return pending
}

func seedMigrations(filenames []string) error {
	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin seed transaction: %w", err)
	}

	stmt, err := tx.Prepare("INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING")
	if err != nil {
		tx.Rollback() //nolint:errcheck
		return fmt.Errorf("failed to prepare seed statement: %w", err)
	}
	defer stmt.Close()

	for _, f := range filenames {
		if _, err := stmt.Exec(f); err != nil {
			tx.Rollback() //nolint:errcheck
			return fmt.Errorf("failed to seed migration %s: %w", f, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit seed transaction: %w", err)
	}

	log.Printf("Seeded %d existing migrations into schema_migrations", len(filenames))
	return nil
}

func executeMigration(migrationsFS fs.FS, filename string) error {
	content, err := fs.ReadFile(migrationsFS, "migrations/"+filename)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	tx, err := DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if _, err := tx.Exec(string(content)); err != nil {
		tx.Rollback() //nolint:errcheck
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	if _, err := tx.Exec("INSERT INTO schema_migrations (filename) VALUES ($1)", filename); err != nil {
		tx.Rollback() //nolint:errcheck
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}
