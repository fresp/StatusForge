package database

import (
	"database/sql"
	"log"
	"net/url"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func ConnectSQLite(path string) (*sql.DB, error) {
	if err := ensureSQLiteParent(path); err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	u := &url.URL{Scheme: "file", Path: absPath}
	query := u.Query()
	query.Set("_pragma", "busy_timeout(5000)")
	u.RawQuery = query.Encode()
	dsn := u.String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	log.Printf("Connected to SQLite: %s", path)
	return db, nil
}
