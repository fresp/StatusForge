package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConnectSQLite_AllowsPathWithSpaces(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "sqlite data", "statusforge.db")

	db, err := ConnectSQLite(path)
	if err == nil {
		_ = db.Close()
	}

	if err != nil {
		t.Fatalf("expected sqlite path with spaces to open, got error: %v", err)
	}
}

func TestConnectSQLite_AllowsPathWithQuestionMark(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "status?forge.db")

	db, err := ConnectSQLite(path)
	if err == nil {
		_ = db.Close()
	}

	if err != nil {
		t.Fatalf("expected sqlite path with question mark to open, got error: %v", err)
	}
}

func TestConnectSQLite_AllowsPathWithHashCharacter(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "status#forge.db")

	db, err := ConnectSQLite(path)
	if err == nil {
		_ = db.Close()
	}

	if err != nil {
		t.Fatalf("expected sqlite path with hash character to open, got error: %v", err)
	}

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected sqlite file to exist at requested path %q, got: %v", path, statErr)
	}
}

func TestConnectSQLite_AllowsRelativePathWithHashCharacter(t *testing.T) {
	t.Chdir(t.TempDir())
	path := "./data/status#forge.db"

	db, err := ConnectSQLite(path)
	if err == nil {
		_ = db.Close()
	}

	if err != nil {
		t.Fatalf("expected relative sqlite path with hash character to open, got error: %v", err)
	}

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected sqlite file to exist at requested path %q, got: %v", path, statErr)
	}
}

func TestConnectSQLite_AllowsRepoRelativePathWithHashCharacter(t *testing.T) {
	t.Parallel()

	path := "./data/status#forge-test.db"

	db, err := ConnectSQLite(path)
	if err == nil {
		_ = db.Close()
	}

	defer os.Remove(path)

	if err != nil {
		t.Fatalf("expected repo-relative sqlite path with hash character to open, got error: %v", err)
	}

	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("expected sqlite file to exist at requested path %q, got: %v", path, statErr)
	}
}
