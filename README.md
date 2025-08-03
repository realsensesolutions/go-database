# Go Database Package

A simple SQLite database library for Go with automatic retry logic and migration registry.

## ✨ Features

- **🔄 SQLite Retry Logic**: Automatic retry with exponential backoff for `SQLITE_BUSY` errors
- **📦 Migration Registry**: Simple migration management across packages
- **🔗 Connection Management**: Easy database connection handling

## 🚀 Quick Start

```bash
go get github.com/realsensesolutions/go-database
```

### Basic Usage

```go
package main

import (
    database "github.com/realsensesolutions/go-database"
)

func main() {
    // Get database connection
    db, err := database.GetDB()
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // Execute with automatic retry on SQLITE_BUSY
    result, err := database.ExecWithRetry(db, 
        "INSERT INTO users (name, email) VALUES (?, ?)", 
        "John Doe", "john@example.com")

    // Query with retry logic
    rows, err := database.QueryWithRetry(db, 
        "SELECT id, name FROM users WHERE active = ?", true)

    // Single row query with retry
    var userID string
    err = database.QueryRowWithRetry(db, 
        "SELECT id FROM users WHERE email = ?", "john@example.com").Scan(&userID)
}
```

## 🔄 Retry Functions

All standard SQL operations with automatic retry:

```go
// Database operations
result, err := database.ExecWithRetry(db, query, args...)
rows, err := database.QueryWithRetry(db, query, args...)
err := database.QueryRowWithRetry(db, query, args...).Scan(&dest)

// Transaction operations  
result, err := database.TxExecWithRetry(tx, query, args...)
rows, err := database.TxQueryWithRetry(tx, query, args...)
err := database.TxQueryRowWithRetry(tx, query, args...).Scan(&dest)

// Full transaction with retry
err := database.WithTransactionRetry(func(tx *sql.Tx) error {
    // Your transaction logic here
    return nil
})
```

## 📦 Migration System

### 1. Register Migrations

```go
package mypackage

import (
    "path/filepath"
    "runtime"
    database "github.com/realsensesolutions/go-database"
)

func init() {
    // Get current package directory
    _, filename, _, _ := runtime.Caller(0)
    packageDir := filepath.Dir(filename)
    migrationsDir := filepath.Join(packageDir, "migrations")
    
    // Register your migrations
    database.RegisterMigrations(database.MigrationSource{
        Name:      "my-package", 
        Directory: migrationsDir,
    })
}
```

### 2. Run All Migrations

```go
package main

import (
    database "github.com/realsensesolutions/go-database"
    _ "your-app/package1" // Import to register migrations
    _ "your-app/package2" // Import to register migrations
)

func main() {
    // Run all registered migrations
    if err := database.RunAllMigrations(); err != nil {
        panic(err)
    }
}
```

### 3. Migration Files

```
migrations/
├── 001_create_users.up.sql
├── 001_create_users.down.sql
├── 002_add_indexes.up.sql
└── 002_add_indexes.down.sql
```

## ⚙️ Configuration

### Environment Variables
- `DATABASE_FILE`: SQLite database file path (default: `app.db`)

### Retry Settings
- **Max Retry Duration**: 30 seconds
- **Base Delay**: 10 milliseconds  
- **Max Delay**: 1 second
- **Jitter**: 25%

## 📋 API Reference

```go
// Connection
func GetDB() (*sql.DB, error)

// Retry Operations
func ExecWithRetry(db *sql.DB, query string, args ...interface{}) (sql.Result, error)
func QueryWithRetry(db *sql.DB, query string, args ...interface{}) (*sql.Rows, error)
func QueryRowWithRetry(db *sql.DB, query string, args ...interface{}) *RetryRow
func WithTransactionRetry(fn func(*sql.Tx) error) error

// Transaction Operations
func TxExecWithRetry(tx *sql.Tx, query string, args ...interface{}) (sql.Result, error)
func TxQueryWithRetry(tx *sql.Tx, query string, args ...interface{}) (*sql.Rows, error)
func TxQueryRowWithRetry(tx *sql.Tx, query string, args ...interface{}) *TxRetryRow

// Migration Registry
func RegisterMigrations(source MigrationSource)
func RunAllMigrations() error
func GetRegisteredSources() []MigrationSource
```

## 🔧 Requirements

- Go 1.22 or later
- SQLite via `modernc.org/sqlite`
- Migrations via `golang-migrate/migrate`

## 📄 License

MIT License