package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// isTracingEnabled checks if Datadog tracing should be enabled
func isTracingEnabled() bool {
	return os.Getenv("DD_API_KEY_SECRET_ARN") != ""
}

// getServiceName returns the service name for database spans
func getServiceName() string {
	if svc := os.Getenv("DD_SERVICE"); svc != "" {
		return svc + "-sqlite"
	}
	return "sqlite-db"
}

// getDatabasePath returns the database file path from environment
func getDatabasePath() string {
	return os.Getenv("DATABASE_FILE")
}

// QueryContext executes a query with optional Datadog tracing
// Use this instead of db.QueryContext() when you want automatic tracing
func QueryContext(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	if !isTracingEnabled() {
		return db.QueryContext(ctx, query, args...)
	}

	span, ctx := tracer.StartSpanFromContext(ctx, "sqlite.query",
		tracer.SpanType(ext.SpanTypeSQL),
		tracer.ServiceName(getServiceName()),
		tracer.ResourceName(query),
		tracer.Tag(ext.DBType, "sqlite"),
		tracer.Tag(ext.DBInstance, getDatabasePath()),
		tracer.Tag("db.statement.params", fmt.Sprintf("%v", args)), // Raw parameters for debugging
	)
	defer span.Finish()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		span.SetTag(ext.Error, err)
		span.SetTag("error.message", err.Error())
	}
	return rows, err
}

// QueryRowContext executes a query that returns a single row with optional Datadog tracing
// Use this instead of db.QueryRowContext() when you want automatic tracing
func QueryRowContext(ctx context.Context, db *sql.DB, query string, args ...interface{}) *sql.Row {
	if !isTracingEnabled() {
		return db.QueryRowContext(ctx, query, args...)
	}

	span, ctx := tracer.StartSpanFromContext(ctx, "sqlite.query",
		tracer.SpanType(ext.SpanTypeSQL),
		tracer.ServiceName(getServiceName()),
		tracer.ResourceName(query),
		tracer.Tag(ext.DBType, "sqlite"),
		tracer.Tag(ext.DBInstance, getDatabasePath()),
		tracer.Tag("db.statement.params", fmt.Sprintf("%v", args)), // Raw parameters for debugging
	)
	defer span.Finish()

	return db.QueryRowContext(ctx, query, args...)
}

// ExecContext executes a query without returning rows with optional Datadog tracing
// Use this instead of db.ExecContext() when you want automatic tracing
func ExecContext(ctx context.Context, db *sql.DB, query string, args ...interface{}) (sql.Result, error) {
	if !isTracingEnabled() {
		return db.ExecContext(ctx, query, args...)
	}

	span, ctx := tracer.StartSpanFromContext(ctx, "sqlite.exec",
		tracer.SpanType(ext.SpanTypeSQL),
		tracer.ServiceName(getServiceName()),
		tracer.ResourceName(query),
		tracer.Tag(ext.DBType, "sqlite"),
		tracer.Tag(ext.DBInstance, getDatabasePath()),
		tracer.Tag("db.statement.params", fmt.Sprintf("%v", args)), // Raw parameters for debugging
	)
	defer span.Finish()

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		span.SetTag(ext.Error, err)
		span.SetTag("error.message", err.Error())
	}
	return result, err
}
