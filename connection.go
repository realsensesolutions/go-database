package database

import (
	"database/sql"
	"log"
	"os"
)

// Database connection management with retry support

// GetDB returns a database connection with optional Datadog tracing
func GetDB() (*DB, error) {
	databaseFile := os.Getenv("DATABASE_FILE")
	if databaseFile == "" {
		panic("DATABASE_FILE environment variable is required but not set")
	}

	sqlDB, err := sql.Open("sqlite", databaseFile)
	if err != nil {
		return nil, err
	}

	// Test the connection with retry logic for SQLITE_BUSY errors
	err = retryDatabaseOperation(func() error {
		return sqlDB.Ping()
	}, DefaultRetryConfig())

	if err != nil {
		sqlDB.Close()
		return nil, err
	}

	// Wrap with tracing support
	db := &DB{
		DB:          sqlDB,
		traced:      isTracingEnabled(),
		serviceName: getTracingServiceName(),
		dbPath:      databaseFile,
	}

	if db.traced {
		log.Printf("🔍 Database tracing enabled (service: %s, db: %s)", db.serviceName, databaseFile)
	}

	return db, nil
}

// WithTransaction executes a function within a database transaction
func WithTransaction(fn func(*sql.Tx) error) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Begin transaction with retry logic
	var tx *sql.Tx
	err = retryDatabaseOperation(func() error {
		tx, err = db.Begin()
		return err
	}, DefaultRetryConfig())
	if err != nil {
		return err
	}

	// Execute the function
	if err := fn(tx); err != nil {
		// Rollback with retry logic
		rollbackErr := retryDatabaseOperation(func() error {
			return tx.Rollback()
		}, DefaultRetryConfig())
		if rollbackErr != nil {
			// Log rollback error but return original error
			log.Printf("❌ Failed to rollback transaction: %v", rollbackErr)
		}
		return err
	}

	// Commit with retry logic
	return retryDatabaseOperation(func() error {
		return tx.Commit()
	}, DefaultRetryConfig())
}
