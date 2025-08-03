// Package database provides SQLite retry logic and migration registry for Go applications.
//
// Key features:
// - SQLite retry logic with exponential backoff for SQLITE_BUSY errors
// - Migration registry system for packages to register their migrations
// - Simple database connection management
package database

// Version of the go-database package
const Version = "1.0.0"

// RunAllMigrations runs all migrations from registered sources
func RunAllMigrations() error {
	return UpAll()
}
