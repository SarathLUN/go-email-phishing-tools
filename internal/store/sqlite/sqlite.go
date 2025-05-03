package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/pressly/goose/v3"
)

// ConnectDB establishes a connection to the SQLite database and runs migrations.
func ConnectDB(dbPath string) (*sql.DB, error) {
	log.Printf("Connecting to database: %s", dbPath)

	// Ensure the directory for the database file exists
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		log.Printf("Database directory not found, creating: %s", dbDir)
		if err := os.MkdirAll(dbDir, 0755); err != nil { // Use 0755 for directory permissions
			return nil, fmt.Errorf("failed to create database directory '%s': %w", dbDir, err)
		}
	}

	// Connect to the database. DSN options can improve performance/safety.
	// _busy_timeout increases wait time if DB is locked.
	// _journal_mode=WAL enables Write-Ahead Logging for better concurrency.
	dsn := fmt.Sprintf("file:%s?cache=shared&_journal_mode=WAL&_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Check the connection
	if err = db.Ping(); err != nil {
		db.Close() // Close the connection if ping fails
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established successfully.")

	// Run migrations
	log.Println("Applying database migrations...")
	goose.SetBaseFS(nil) // Use filesystem migrations
	// Note: Consider making migrations directory configurable if needed
	if err := goose.SetDialect("sqlite3"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}
	if err := goose.Up(db, "db/migrations"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply database migrations: %w", err)
	}
	log.Println("Database migrations applied successfully.")

	return db, nil
}
