package database

import (
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// UpAll runs all migrations from all registered sources
func UpAll() error {
	log.Printf("🚀 Running all migrations from registered sources...")

	sources := GetRegisteredSources()
	if len(sources) == 0 {
		log.Printf("⚠️  No migration sources registered")
		return nil
	}

	// For each registered source, run its migrations using golang-migrate
	for _, source := range sources {
		log.Printf("📦 Processing migrations from: %s", source.Name)

		if source.Directory == "" {
			log.Printf("⚠️  No directory specified for source: %s", source.Name)
			continue
		}

		// Use the simplified migration runner
		if err := runMigrationsFromDirectory(source.Directory); err != nil {
			return err
		}

		log.Printf("✅ Completed migrations for: %s", source.Name)
	}

	log.Printf("🎉 All migrations completed successfully!")
	return nil
}

// runMigrationsFromDirectory runs migrations from a specific directory
func runMigrationsFromDirectory(migrationsDir string) error {
	// Get database file path
	databaseFile := os.Getenv("DATABASE_FILE")
	if databaseFile == "" {
		databaseFile = "app.db"
	}

	// Create database and migrations URLs
	databaseURL := fmt.Sprintf("sqlite://%s", databaseFile)
	migrationsURL := fmt.Sprintf("file://%s", migrationsDir)

	// Initialize migrate instance
	m, err := migrate.New(migrationsURL, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate: %w", err)
	}
	defer m.Close()

	// Run migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// GetMigrationStatus returns status information about migrations from all sources
func GetMigrationStatus() (map[string]interface{}, error) {
	sources := GetRegisteredSources()
	status := make(map[string]interface{})

	status["registered_sources"] = len(sources)
	status["sources"] = make(map[string]interface{})

	for _, source := range sources {
		sourceStatus := make(map[string]interface{})
		sourceStatus["name"] = source.Name
		sourceStatus["has_directory"] = source.Directory != ""
		sourceStatus["has_embed"] = source.EmbedFS != nil

		// Try to get migration count
		count, err := countMigrationFiles(source)
		if err != nil {
			sourceStatus["error"] = err.Error()
			sourceStatus["migration_count"] = 0
		} else {
			sourceStatus["migration_count"] = count
		}

		status["sources"].(map[string]interface{})[source.Name] = sourceStatus
	}

	return status, nil
}
