package database

import (
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
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

		// Handle embedded filesystem sources
		if source.EmbedFS != nil {
			log.Printf("📁 Using embedded filesystem for: %s", source.Name)
			if err := runEmbeddedMigrations(source.EmbedFS, "migrations", source.Prefix); err != nil {
				return fmt.Errorf("failed to run embedded migrations for %s: %w", source.Name, err)
			}
			log.Printf("✅ Completed embedded migrations for: %s", source.Name)
			continue
		}

		// Handle directory-based sources (legacy)
		if source.Directory != "" {
			log.Printf("📂 Using directory filesystem for: %s", source.Name)
			// Use the prefix-aware migration runner
			if err := runMigrationsFromDirectoryWithPrefix(source.Directory, source.Prefix); err != nil {
				return fmt.Errorf("failed to run directory migrations for %s: %w", source.Name, err)
			}
			log.Printf("✅ Completed directory migrations for: %s", source.Name)
			continue
		}

		log.Printf("⚠️  No migration source (directory or embed) specified for: %s", source.Name)
	}

	log.Printf("🎉 All migrations completed successfully!")
	return nil
}

// runMigrationsFromDirectory runs migrations from a specific directory (legacy, no prefix)
func runMigrationsFromDirectory(migrationsDir string) error {
	return runMigrationsFromDirectoryWithPrefix(migrationsDir, "")
}

// runMigrationsFromDirectoryWithPrefix runs migrations from a directory with prefix support
func runMigrationsFromDirectoryWithPrefix(migrationsDir string, prefix string) error {
	// Get database file path
	databaseFile := os.Getenv("DATABASE_FILE")
	if databaseFile == "" {
		databaseFile = "app.db"
	}

	// If no prefix, use the legacy approach
	if prefix == "" {
		return runLegacyMigrations(migrationsDir, databaseFile)
	}

	// Use prefix-aware migration runner
	return runPrefixedMigrations(migrationsDir, prefix, databaseFile)
}

// runLegacyMigrations runs migrations without prefix (backward compatibility)
func runLegacyMigrations(migrationsDir string, databaseFile string) error {
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

// runPrefixedMigrations runs migrations with prefix support using separate schema table
func runPrefixedMigrations(migrationsDir string, prefix string, databaseFile string) error {
	// Create database and migrations URLs
	databaseURL := fmt.Sprintf("sqlite://%s?x-migrations-table=%sschema_migrations", databaseFile, prefix)
	migrationsURL := fmt.Sprintf("file://%s", migrationsDir)

	log.Printf("🏷️  Using prefixed schema table: %sschema_migrations", prefix)

	// Initialize migrate instance with prefixed schema table
	m, err := migrate.New(migrationsURL, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate with prefix %s: %w", prefix, err)
	}
	defer m.Close()

	// Run migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations with prefix %s: %w", prefix, err)
	}

	return nil
}

// runEmbeddedMigrations runs migrations from an embedded filesystem with prefix support
func runEmbeddedMigrations(embedFS *embed.FS, subpath string, prefix string) error {
	// Get database file path
	databaseFile := os.Getenv("DATABASE_FILE")
	if databaseFile == "" {
		databaseFile = "app.db"
	}

	// Create database URL with optional prefix for schema table
	var databaseURL string
	if prefix != "" {
		databaseURL = fmt.Sprintf("sqlite://%s?x-migrations-table=%sschema_migrations", databaseFile, prefix)
		log.Printf("🏷️  Using prefixed schema table: %sschema_migrations", prefix)
	} else {
		databaseURL = fmt.Sprintf("sqlite://%s", databaseFile)
	}

	// Create iofs driver from embedded filesystem
	driver, err := iofs.New(*embedFS, subpath)
	if err != nil {
		return fmt.Errorf("failed to create iofs driver: %w", err)
	}

	// Initialize migrate instance with embedded source
	m, err := migrate.NewWithSourceInstance("iofs", driver, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate with embedded FS: %w", err)
	}
	defer m.Close()

	// Run migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run embedded migrations: %w", err)
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
