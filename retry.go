package database

import (
	"database/sql"
	"log"
	"math/rand"
	"strings"
	"time"
)

// Default retry configuration
const (
	DefaultMaxRetryDuration = 30 * time.Second
	DefaultBaseDelay        = 10 * time.Millisecond
	DefaultMaxDelay         = 1 * time.Second
	DefaultJitterPercent    = 0.25 // ±25%
)

// RetryConfig holds configuration for database retry operations
type RetryConfig struct {
	MaxRetryDuration time.Duration
	BaseDelay        time.Duration
	MaxDelay         time.Duration
	JitterPercent    float64
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetryDuration: DefaultMaxRetryDuration,
		BaseDelay:        DefaultBaseDelay,
		MaxDelay:         DefaultMaxDelay,
		JitterPercent:    DefaultJitterPercent,
	}
}

// retryDatabaseOperation executes a database operation with exponential backoff retry and jitter
func retryDatabaseOperation(operation func() error, config RetryConfig) error {
	var err error
	startTime := time.Now()
	attempt := 0

	for {
		err = operation()
		if err == nil {
			if attempt > 0 {
				log.Printf("✅ SQLite operation succeeded after %d retries in %v", attempt, time.Since(startTime))
			}
			return nil
		}

		// Check if it's a SQLite BUSY error
		if !strings.Contains(err.Error(), "database is locked") && !strings.Contains(err.Error(), "SQLITE_BUSY") {
			// Non-retryable error
			log.Printf("❌ Non-retryable SQLite error: %v", err)
			return err
		}

		// Check if we've exceeded max retry duration
		elapsed := time.Since(startTime)
		if elapsed >= config.MaxRetryDuration {
			log.Printf("❌ SQLite operation failed after %v (max retry duration exceeded)", elapsed)
			return err
		}

		// Calculate exponential backoff with jitter
		// Base delay: baseDelay * 2^attempt, capped at maxDelay
		multiplier := 1 << uint(attempt)
		baseDelay := time.Duration(int64(config.BaseDelay) * int64(multiplier))
		if baseDelay > config.MaxDelay {
			baseDelay = config.MaxDelay
		}

		// Add jitter: ±jitterPercent of base delay
		jitterRange := float64(baseDelay) * config.JitterPercent
		jitter := time.Duration(rand.Float64()*jitterRange*2 - jitterRange) // ±jitterPercent
		delay := baseDelay + jitter

		// Ensure delay is positive and doesn't exceed remaining time
		if delay < 0 {
			delay = config.BaseDelay
		}

		remaining := config.MaxRetryDuration - elapsed
		if delay > remaining {
			delay = remaining
		}

		if delay <= 0 {
			log.Printf("❌ SQLite operation failed after %v (no time remaining for retry)", elapsed)
			return err
		}

		attempt++
		log.Printf("🔄 SQLite BUSY - retrying in %v (attempt %d, elapsed %v)", delay, attempt, elapsed)
		time.Sleep(delay)
	}
}

// ExecWithRetry executes a database Exec operation with retry logic
func ExecWithRetry(db *sql.DB, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error

	retryErr := retryDatabaseOperation(func() error {
		result, err = db.Exec(query, args...)
		return err
	}, DefaultRetryConfig())

	return result, retryErr
}

// QueryWithRetry executes a database Query operation with retry logic
func QueryWithRetry(db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	retryErr := retryDatabaseOperation(func() error {
		rows, err = db.Query(query, args...)
		return err
	}, DefaultRetryConfig())

	return rows, retryErr
}

// QueryRowWithRetry executes a database QueryRow operation with retry logic
// Returns a wrapper that will retry the entire QueryRow+Scan operation on SQLITE_BUSY
func QueryRowWithRetry(db *sql.DB, query string, args ...interface{}) *RetryRow {
	return &RetryRow{
		db:    db,
		query: query,
		args:  args,
	}
}

// RetryRow wraps sql.Row to provide retry functionality
type RetryRow struct {
	db    *sql.DB
	query string
	args  []interface{}
}

// Scan executes the query and scans the result with retry logic
func (r *RetryRow) Scan(dest ...interface{}) error {
	var err error

	retryErr := retryDatabaseOperation(func() error {
		row := r.db.QueryRow(r.query, r.args...)
		err = row.Scan(dest...)
		return err
	}, DefaultRetryConfig())

	return retryErr
}

// TxExecWithRetry executes a transaction Exec operation with retry logic
func TxExecWithRetry(tx *sql.Tx, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error

	retryErr := retryDatabaseOperation(func() error {
		result, err = tx.Exec(query, args...)
		return err
	}, DefaultRetryConfig())

	return result, retryErr
}

// TxQueryWithRetry executes a transaction Query operation with retry logic
func TxQueryWithRetry(tx *sql.Tx, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	retryErr := retryDatabaseOperation(func() error {
		rows, err = tx.Query(query, args...)
		return err
	}, DefaultRetryConfig())

	return rows, retryErr
}

// TxQueryRowWithRetry executes a transaction QueryRow operation with retry logic
func TxQueryRowWithRetry(tx *sql.Tx, query string, args ...interface{}) *TxRetryRow {
	return &TxRetryRow{
		tx:    tx,
		query: query,
		args:  args,
	}
}

// TxRetryRow wraps sql.Row for transaction queries to provide retry functionality
type TxRetryRow struct {
	tx    *sql.Tx
	query string
	args  []interface{}
}

// Scan executes the transaction query and scans the result with retry logic
func (r *TxRetryRow) Scan(dest ...interface{}) error {
	var err error

	retryErr := retryDatabaseOperation(func() error {
		row := r.tx.QueryRow(r.query, r.args...)
		err = row.Scan(dest...)
		return err
	}, DefaultRetryConfig())

	return retryErr
}

// WithTransactionRetry executes a function within a database transaction with retry logic
// This creates its own transaction and doesn't use the nested WithTransaction to avoid double-retry issues
func WithTransactionRetry(fn func(*sql.Tx) error) error {
	return retryDatabaseOperation(func() error {
		// Get fresh database connection
		db, err := GetDB()
		if err != nil {
			return err
		}
		defer db.Close()

		// Begin transaction (without nested retry to avoid conflicts)
		tx, err := db.Begin()
		if err != nil {
			return err
		}

		// Execute the function
		if err := fn(tx); err != nil {
			// Rollback on error (simple rollback without retry)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("❌ Failed to rollback transaction: %v", rollbackErr)
			}
			return err
		}

		// Commit the transaction (simple commit without retry)
		return tx.Commit()
	}, DefaultRetryConfig())
}

// Custom retry functions for specific configurations

// ExecWithRetryConfig executes a database Exec operation with custom retry config
func ExecWithRetryConfig(db *sql.DB, config RetryConfig, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error

	retryErr := retryDatabaseOperation(func() error {
		result, err = db.Exec(query, args...)
		return err
	}, config)

	return result, retryErr
}

// QueryWithRetryConfig executes a database Query operation with custom retry config
func QueryWithRetryConfig(db *sql.DB, config RetryConfig, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	retryErr := retryDatabaseOperation(func() error {
		rows, err = db.Query(query, args...)
		return err
	}, config)

	return rows, retryErr
}
