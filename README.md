# Go Database Package

A robust database abstraction layer for Go applications with built-in retry logic, migration support, and connection management.

## Features

- **🔄 Retry Logic**: Exponential backoff retry mechanism for handling `SQLITE_BUSY` errors
- **📦 Migration System**: Comprehensive database migration management with versioning
- **🔗 Connection Management**: Efficient database connection handling and pooling
- **📊 Registry Support**: Migration registry for multi-package database schemas
- **⚡ Performance**: Optimized for concurrent database operations

## Installation

```bash
go get github.com/realsensesolutions/go-database
```

## Quick Start

### Basic Database Operations with Retry

```go
package main

import (
    "database/sql"
    database "github.com/realsensesolutions/go-database"
)

func main() {
    db, err := sql.Open("sqlite", "example.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // Execute with automatic retry on SQLITE_BUSY
    result, err := database.ExecWithRetry(db, 
        "INSERT INTO users (name, email) VALUES (?, ?)", 
        "John Doe", "john@example.com")
    if err != nil {
        panic(err)
    }
}
```

### Database Migrations

```go
package main

import (
    database "github.com/realsensesolutions/go-database"
)

func main() {
    config := database.Config{
        DatabasePath:    "app.db",
        MigrationsPath: "./migrations",
    }

    migrator, err := database.NewMigrator(config)
    if err != nil {
        panic(err)
    }

    // Run all pending migrations
    err = migrator.Up()
    if err != nil {
        panic(err)
    }
}
```

### Custom Retry Configuration

```go
package main

import (
    "time"
    database "github.com/realsensesolutions/go-database"
)

func main() {
    config := database.RetryConfig{
        MaxRetryDuration: 60 * time.Second,
        BaseDelay:       20 * time.Millisecond,
        MaxDelay:        2 * time.Second,
        JitterPercent:   0.3,
    }

    db, _ := sql.Open("sqlite", "example.db")
    
    result, err := database.ExecWithRetryConfig(db, config,
        "UPDATE users SET last_login = ? WHERE id = ?",
        time.Now(), userID)
}
```

## Core Components

### Retry System (`retry.go`)
Handles SQLite database locking with exponential backoff:
- Automatic retry on `SQLITE_BUSY` errors
- Configurable retry duration and delays
- Jitter to prevent thundering herd
- Support for transactions

### Migration System (`migrator.go`)
Comprehensive database migration management:
- Version tracking and rollback support
- Multiple migration sources
- Safe concurrent migration execution
- Integration with `golang-migrate/migrate`

### Registry System (`registry.go`)
Multi-package schema management:
- Register migrations from multiple packages
- Conflict detection and resolution
- Ordered migration execution
- Version tracking per registry

### Connection Management (`connection.go`)
Efficient database connection handling:
- Connection pooling and lifecycle management
- Error handling and recovery
- Integration with retry system

## API Reference

### Retry Functions

```go
// Basic retry functions with default configuration
func ExecWithRetry(db *sql.DB, query string, args ...interface{}) (sql.Result, error)
func QueryWithRetry(db *sql.DB, query string, args ...interface{}) (*sql.Rows, error)
func QueryRowWithRetry(db *sql.DB, query string, args ...interface{}) *RetryRow

// Custom retry configuration
func ExecWithRetryConfig(db *sql.DB, config RetryConfig, query string, args ...interface{}) (sql.Result, error)
func QueryWithRetryConfig(db *sql.DB, config RetryConfig, query string, args ...interface{}) (*sql.Rows, error)

// Transaction support
func WithTransactionRetry(fn func(*sql.Tx) error) error
```

### Migration Functions

```go
// Create new migrator
func NewMigrator(config Config) (*Migrator, error)
func NewSimpleMigrator(databasePath, migrationsPath string) (*Migrator, error)

// Migration operations
func (m *Migrator) Up() error
func (m *Migrator) Down() error
func (m *Migrator) Version() (uint, error)
func (m *Migrator) Drop() error
```

### Registry Functions

```go
// Registry management
func NewRegistry() *Registry
func (r *Registry) RegisterMigrations(name string, migrations []Migration) error
func (r *Registry) RunAllMigrations(databasePath string) error
```

## Configuration

### Retry Configuration

```go
type RetryConfig struct {
    MaxRetryDuration time.Duration  // Maximum total retry time
    BaseDelay        time.Duration  // Initial delay between retries
    MaxDelay         time.Duration  // Maximum delay between retries
    JitterPercent    float64        // Jitter percentage (0.0-1.0)
}
```

### Migration Configuration

```go
type Config struct {
    DatabaseURL       string  // Database connection URL
    DatabasePath      string  // Local database file path
    MigrationsSource  string  // Migration source URL
    MigrationsPath    string  // Local migrations directory
}
```

## Default Values

- **MaxRetryDuration**: 30 seconds
- **BaseDelay**: 10 milliseconds
- **MaxDelay**: 1 second
- **JitterPercent**: 25%

## Requirements

- Go 1.24.5 or later
- SQLite support via `modernc.org/sqlite`
- Migration support via `golang-migrate/migrate`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

- 📖 [Documentation](https://github.com/realsensesolutions/go-database)
- 🐛 [Issues](https://github.com/realsensesolutions/go-database/issues)
- 💬 [Discussions](https://github.com/realsensesolutions/go-database/discussions)