# Database Tracing with Datadog

This package provides optional Datadog APM tracing for SQLite database operations.

## Features

- ✅ **Zero overhead when disabled** - Direct passthrough to `*sql.DB` methods
- ✅ **Automatic when enabled** - Just use the wrapper functions
- ✅ **Full parameter visibility** - Captures query parameters for debugging
- ✅ **Environment-driven** - Enabled via `DD_API_KEY_SECRET_ARN` env var
- ✅ **Configurable service name** - Uses `DD_SERVICE` env var

## Usage

### Without Tracing (Standard)

```go
import (
    "database/sql"
    database "github.com/realsensesolutions/go-database"
)

db, err := database.GetDB()

// Standard *sql.DB methods - no tracing
rows, err := db.QueryContext(ctx, "SELECT * FROM users WHERE id = ?", userID)
```

### With Tracing (Traced)

```go
import (
    database "github.com/realsensesolutions/go-database"
)

db, err := database.GetDB()

// Use database package wrappers - automatic tracing when enabled
rows, err := database.QueryContext(ctx, db, "SELECT * FROM users WHERE id = ?", userID)
```

## API

### QueryContext

```go
func QueryContext(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error)
```

Executes a query that returns rows. Creates a Datadog span if tracing is enabled.

### QueryRowContext

```go
func QueryRowContext(ctx context.Context, db *sql.DB, query string, args ...interface{}) *sql.Row
```

Executes a query that returns a single row. Creates a Datadog span if tracing is enabled.

### ExecContext

```go
func ExecContext(ctx context.Context, db *sql.DB, query string, args ...interface{}) (sql.Result, error)
```

Executes a query without returning rows (INSERT, UPDATE, DELETE). Creates a Datadog span if tracing is enabled.

## Configuration

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DD_API_KEY_SECRET_ARN` | Enables tracing when set | `arn:aws:secretsmanager:...` |
| `DD_SERVICE` | Base service name | `grantpulse` → `grantpulse-sqlite` |
| `DATABASE_FILE` | Database file path (for span tags) | `/tmp/app.db` |

### Span Tags

Each traced query includes:

- `span.type`: `sql`
- `service.name`: `${DD_SERVICE}-sqlite` (e.g., `grantpulse-sqlite`)
- `resource.name`: The SQL query
- `db.type`: `sqlite`
- `db.instance`: Database file path
- `db.statement.params`: Query parameters (raw, for debugging)
- `error`: Set if query fails
- `error.message`: Error details if query fails

## Migration Guide

### Gradual Migration (Recommended)

Migrate queries one file at a time:

```go
// Before
rows, err := db.QueryContext(ctx, query, args...)

// After
rows, err := database.QueryContext(ctx, db, query, args...)
```

### Example Migration

**Before:**
```go
func GetUserByID(ctx context.Context, db *sql.DB, userID int) (*User, error) {
    query := "SELECT id, name, email FROM users WHERE id = ?"
    
    var user User
    err := db.QueryRowContext(ctx, query, userID).Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        return nil, err
    }
    return &user, nil
}
```

**After:**
```go
import database "github.com/realsensesolutions/go-database"

func GetUserByID(ctx context.Context, db *sql.DB, userID int) (*User, error) {
    query := "SELECT id, name, email FROM users WHERE id = ?"
    
    var user User
    err := database.QueryRowContext(ctx, db, query, userID).Scan(&user.ID, &user.Name, &user.Email)
    if err != nil {
        return nil, err
    }
    return &user, nil
}
```

## Security Note

⚠️ **Query parameters are captured in raw form** for debugging purposes. This may expose sensitive data (passwords, tokens, PII) in your Datadog traces.

**Recommendations:**
- Don't pass sensitive data as query parameters
- Use application-level redaction if needed
- Review Datadog retention policies for sensitive data

## Performance

- **No tracing**: Zero overhead - direct passthrough
- **With tracing**: Minimal overhead (~0.1-0.5ms per query for span creation)
- Spans are sent asynchronously to Datadog agent

## Compatibility

- Requires `gopkg.in/DataDog/dd-trace-go.v1 v1.74.6` or later
- Works with `modernc.org/sqlite` (pure Go, CGO-free)
- Compatible with Datadog Lambda Extension
- Backward compatible - no breaking changes to existing API

