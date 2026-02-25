package database

import (
	"testing"
	"testing/fstest"
)

func TestDiscoverMigrations(t *testing.T) {
	mockFS := fstest.MapFS{
		"migrations/003_third.sql":           {},
		"migrations/001_first.sql":           {},
		"migrations/002_second.sql":          {},
		"migrations/README.md":               {},
		"migrations/007_dup_a.sql":           {},
		"migrations/007_dup_b.sql":           {},
		"migrations/010_with spaces.sql.bak": {},
	}

	files, err := discoverMigrations(mockFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"001_first.sql",
		"002_second.sql",
		"003_third.sql",
		"007_dup_a.sql",
		"007_dup_b.sql",
	}

	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}

	for i, f := range files {
		if f != expected[i] {
			t.Errorf("file[%d]: expected %q, got %q", i, expected[i], f)
		}
	}
}

func TestDiscoverMigrations_Empty(t *testing.T) {
	mockFS := fstest.MapFS{
		"migrations/.gitkeep": {},
	}

	files, err := discoverMigrations(mockFS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d: %v", len(files), files)
	}
}

func TestFindPending_NoneApplied(t *testing.T) {
	allFiles := []string{"001_a.sql", "002_b.sql", "003_c.sql"}
	applied := map[string]bool{}

	pending := findPending(allFiles, applied)
	if len(pending) != 3 {
		t.Fatalf("expected 3 pending, got %d", len(pending))
	}
}

func TestFindPending_SomeApplied(t *testing.T) {
	allFiles := []string{"001_a.sql", "002_b.sql", "003_c.sql"}
	applied := map[string]bool{"001_a.sql": true, "002_b.sql": true}

	pending := findPending(allFiles, applied)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
	if pending[0] != "003_c.sql" {
		t.Errorf("expected 003_c.sql, got %s", pending[0])
	}
}

func TestFindPending_AllApplied(t *testing.T) {
	allFiles := []string{"001_a.sql", "002_b.sql"}
	applied := map[string]bool{"001_a.sql": true, "002_b.sql": true}

	pending := findPending(allFiles, applied)
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending, got %d", len(pending))
	}
}

func TestFindPending_PreservesOrder(t *testing.T) {
	allFiles := []string{"001_a.sql", "003_c.sql", "005_e.sql", "007_g.sql"}
	applied := map[string]bool{"001_a.sql": true, "005_e.sql": true}

	pending := findPending(allFiles, applied)
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}
	if pending[0] != "003_c.sql" || pending[1] != "007_g.sql" {
		t.Errorf("expected [003_c.sql, 007_g.sql], got %v", pending)
	}
}
