package database

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

// Connect establishes a PostgreSQL connection using the provided DSN and applies
// some sane pooling defaults.
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
