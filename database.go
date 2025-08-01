// Package database provides a unified database infrastructure with migration registry,
// SQLite retry logic, and connection management for Go applications.
//
// Key features:
// - Migration registry system for packages to register their migrations
// - Sophisticated SQLite retry logic with exponential backoff and jitter
// - Connection management with automatic retry on SQLITE_BUSY errors
// - Unified "up all migrations" functionality
package database

import (
	"fmt"
	"log"
)

// Version of the go-database package
const Version = "1.0.0"

// init runs package initialization
func init() {
	log.Printf("📦 go-database v%s initialized", Version)
}

// InitializeDatabase sets up the database with retry logic and runs any pending migrations
func InitializeDatabase() error {
	log.Printf("🔄 Initializing database...")
	
	// Test database connection
	db, err := GetDB()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()
	
	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	log.Printf("✅ Database connection established")
	return nil
}

// RunAllMigrations is a convenience function to run all migrations from registered sources
func RunAllMigrations() error {
	return UpAll()
}

// GetDatabaseStats returns statistics about the database and registered migrations
func GetDatabaseStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Database connection info
	db, err := GetDB()
	if err != nil {
		stats["database_connection"] = "failed"
		stats["database_error"] = err.Error()
	} else {
		defer db.Close()
		if err := db.Ping(); err != nil {
			stats["database_connection"] = "ping_failed"
			stats["database_error"] = err.Error()
		} else {
			stats["database_connection"] = "ok"
		}
	}
	
	// Migration source statistics
	migrationStats, err := GetMigrationStats()
	if err != nil {
		stats["migration_error"] = err.Error()
	} else {
		stats["migration_sources"] = migrationStats
	}
	
	// Registry information
	sources := GetRegisteredSources()
	stats["registered_sources_count"] = len(sources)
	
	sourceNames := make([]string, len(sources))
	for i, source := range sources {
		sourceNames[i] = source.Name
	}
	stats["registered_source_names"] = sourceNames
	
	return stats, nil
}