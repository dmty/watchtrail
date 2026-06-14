package store

import (
	"path/filepath"
	"testing"
)

func openTemp(t *testing.T) *SQLiteRepo {
	t.Helper()
	repo, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestMigrationsCreateTables(t *testing.T) {
	repo := openTemp(t)
	for _, table := range []string{"media_item", "watch_event", "watch_session", "schema_migrations"} {
		var name string
		err := repo.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing: %v", table, err)
		}
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	r1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	_ = r1.Close()
	r2, err := Open(path) // re-running migrate must not error
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	_ = r2.Close()
}
