package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// DB wraps sql.DB with optional Datadog tracing
type DB struct {
	*sql.DB
	traced      bool
	serviceName string
	dbPath      string
}

// isTracingEnabled checks if Datadog tracing should be enabled
func isTracingEnabled() bool {
	return os.Getenv("DD_API_KEY_SECRET_ARN") != ""
}

// getTracingServiceName returns the service name for database spans
func getTracingServiceName() string {
	// Use DD_SERVICE environment variable with -sqlite suffix
	if svc := os.Getenv("DD_SERVICE"); svc != "" {
		return svc + "-sqlite"
	}
	// Fallback to generic name
	return "sqlite-db"
}

// QueryContext executes a query that returns rows with optional tracing
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if !db.traced {
		return db.DB.QueryContext(ctx, query, args...)
	}

	span, ctx := tracer.StartSpanFromContext(ctx, "sqlite.query",
		tracer.SpanType(ext.SpanTypeSQL),
		tracer.ServiceName(db.serviceName),
		tracer.ResourceName(query),
		tracer.Tag(ext.DBType, "sqlite"),
		tracer.Tag(ext.DBInstance, db.dbPath),
		tracer.Tag("db.statement.params", fmt.Sprintf("%v", args)),
	)
	defer span.Finish()

	rows, err := db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		span.SetTag(ext.Error, err)
		span.SetTag(ext.ErrorMsg, err.Error())
	}
	return rows, err
}

// QueryRowContext executes a query that returns a single row with optional tracing
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if !db.traced {
		return db.DB.QueryRowContext(ctx, query, args...)
	}

	span, ctx := tracer.StartSpanFromContext(ctx, "sqlite.query",
		tracer.SpanType(ext.SpanTypeSQL),
		tracer.ServiceName(db.serviceName),
		tracer.ResourceName(query),
		tracer.Tag(ext.DBType, "sqlite"),
		tracer.Tag(ext.DBInstance, db.dbPath),
		tracer.Tag("db.statement.params", fmt.Sprintf("%v", args)),
	)
	defer span.Finish()

	row := db.DB.QueryRowContext(ctx, query, args...)
	// Note: Can't check for errors here as sql.Row defers error until Scan()
	return row
}

// ExecContext executes a query without returning rows with optional tracing
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if !db.traced {
		return db.DB.ExecContext(ctx, query, args...)
	}

	span, ctx := tracer.StartSpanFromContext(ctx, "sqlite.exec",
		tracer.SpanType(ext.SpanTypeSQL),
		tracer.ServiceName(db.serviceName),
		tracer.ResourceName(query),
		tracer.Tag(ext.DBType, "sqlite"),
		tracer.Tag(ext.DBInstance, db.dbPath),
		tracer.Tag("db.statement.params", fmt.Sprintf("%v", args)),
	)
	defer span.Finish()

	result, err := db.DB.ExecContext(ctx, query, args...)
	if err != nil {
		span.SetTag(ext.Error, err)
		span.SetTag(ext.ErrorMsg, err.Error())
	}
	return result, err
}

// PrepareContext creates a prepared statement with optional tracing context
func (db *DB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if !db.traced {
		return db.DB.PrepareContext(ctx, query)
	}

	span, ctx := tracer.StartSpanFromContext(ctx, "sqlite.prepare",
		tracer.SpanType(ext.SpanTypeSQL),
		tracer.ServiceName(db.serviceName),
		tracer.ResourceName(query),
		tracer.Tag(ext.DBType, "sqlite"),
		tracer.Tag(ext.DBInstance, db.dbPath),
	)
	defer span.Finish()

	stmt, err := db.DB.PrepareContext(ctx, query)
	if err != nil {
		span.SetTag(ext.Error, err)
		span.SetTag(ext.ErrorMsg, err.Error())
	}
	return stmt, err
}

// BeginTx starts a transaction with optional tracing
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if !db.traced {
		return db.DB.BeginTx(ctx, opts)
	}

	span, ctx := tracer.StartSpanFromContext(ctx, "sqlite.begin",
		tracer.SpanType(ext.SpanTypeSQL),
		tracer.ServiceName(db.serviceName),
		tracer.ResourceName("BEGIN TRANSACTION"),
		tracer.Tag(ext.DBType, "sqlite"),
		tracer.Tag(ext.DBInstance, db.dbPath),
	)
	defer span.Finish()

	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		span.SetTag(ext.Error, err)
		span.SetTag(ext.ErrorMsg, err.Error())
	}
	return tx, err
}
