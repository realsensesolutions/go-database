package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrator handles database migrations with registry support
type Migrator struct {
	migrate      *migrate.Migrate
	databasePath string
	db           *sql.DB
}

// Config holds the configuration for creating a migrator
type Config struct {
	DatabasePath     string
	MigrationsPath   string
	DatabaseURL      string
	MigrationsSource string
}

// NewMigrator creates a new migrator instance
func NewMigrator(config Config) (*Migrator, error) {
	// Determine database URL and migrations source
	var databaseURL, migrationsSource string

	if config.DatabaseURL != "" {
		databaseURL = config.DatabaseURL
	} else if config.DatabasePath != "" {
		databaseURL = fmt.Sprintf("sqlite://%s", config.DatabasePath)
	} else {
		return nil, fmt.Errorf("either DatabaseURL or DatabasePath must be provided")
	}

	if config.MigrationsSource != "" {
		migrationsSource = config.MigrationsSource
	} else if config.MigrationsPath != "" {
		migrationsSource = fmt.Sprintf("file://%s", config.MigrationsPath)
	} else {
		return nil, fmt.Errorf("either MigrationsSource or MigrationsPath must be provided")
	}

	// Initialize migrate instance
	m, err := migrate.New(migrationsSource, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrate: %w", err)
	}

	// Create direct database connection for validation (optional)
	var db *sql.DB
	if config.DatabasePath != "" {
		db, err = sql.Open("sqlite", config.DatabasePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
	}

	return &Migrator{
		migrate:      m,
		databasePath: config.DatabasePath,
		db:           db,
	}, nil
}

// NewRegistryMigrator creates a migrator that uses the global migration registry
func NewRegistryMigrator() (*Migrator, error) {
	databaseFile := ""
	if dbFile := GetDatabaseFile(); dbFile != "" {
		databaseFile = dbFile
	} else {
		return nil, fmt.Errorf("DATABASE_FILE environment variable or default database file required")
	}

	// For now, we'll create a standard migrator
	// In a full implementation, this would create a custom source driver
	// that reads from the migration registry
	config := Config{
		DatabasePath: databaseFile,
		// This will be replaced with registry-based source
		MigrationsPath: "migrations", // Placeholder
	}

	return NewMigrator(config)
}

// GetDatabaseFile returns the database file path from environment or default
func GetDatabaseFile() string {
	databaseFile := ""
	if dbFile := GetEnvDatabaseFile(); dbFile != "" {
		databaseFile = dbFile
	} else {
		databaseFile = "app.db" // default fallback
	}
	return databaseFile
}

// GetEnvDatabaseFile returns the DATABASE_FILE environment variable
func GetEnvDatabaseFile() string {
	return os.Getenv("DATABASE_FILE")
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	err := m.migrate.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}
	return nil
}

// UpAll runs all migrations from all registered sources
func UpAll() error {
	log.Printf("🚀 Running all migrations from registered sources...")
	
	sources := GetRegisteredSources()
	if len(sources) == 0 {
		log.Printf("⚠️  No migration sources registered")
		return nil
	}

	// For each registered source, run its migrations
	for _, source := range sources {
		log.Printf("📦 Processing migrations from: %s", source.Name)
		
		// Create migrator for this source
		var config Config
		if source.Directory != "" {
			config = Config{
				DatabasePath:   GetDatabaseFile(),
				MigrationsPath: source.Directory,
			}
		} else {
			// Handle embedded FS sources - placeholder for now
			log.Printf("⚠️  Embedded FS migrations not yet implemented for: %s", source.Name)
			continue
		}

		migrator, err := NewMigrator(config)
		if err != nil {
			return fmt.Errorf("failed to create migrator for source %s: %w", source.Name, err)
		}

		// Run migrations for this source
		if err := migrator.Up(); err != nil {
			migrator.Close()
			return fmt.Errorf("failed to run migrations for source %s: %w", source.Name, err)
		}

		migrator.Close()
		log.Printf("✅ Completed migrations for: %s", source.Name)
	}

	log.Printf("🎉 All migrations completed successfully!")
	return nil
}

// Down rolls back N migrations
func (m *Migrator) Down(steps int) error {
	err := m.migrate.Steps(-steps)
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	return nil
}

// Goto migrates to a specific version
func (m *Migrator) Goto(version uint) error {
	err := m.migrate.Migrate(version)
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}
	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(version int) error {
	err := m.migrate.Force(version)
	if err != nil {
		return fmt.Errorf("failed to force version %d: %w", version, err)
	}
	return nil
}

// Drop drops the entire database
func (m *Migrator) Drop() error {
	err := m.migrate.Drop()
	if err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}
	return nil
}

// ValidateSchema validates that the expected tables and columns exist
func (m *Migrator) ValidateSchema() error {
	if m.db == nil {
		return fmt.Errorf("database connection not available for validation")
	}

	// Check expected tables exist - this will be customizable based on registered sources
	expectedTables := []string{"users", "folders", "boards", "topics", "sentiments", "members", "schema_migrations"}

	for _, table := range expectedTables {
		var count int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
		err := m.db.QueryRow(query, table).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
		if count == 0 {
			log.Printf("⚠️  Expected table '%s' does not exist (may be from unregistered source)", table)
		}
	}

	return nil
}

// validateTableColumns checks if a table has all expected columns
func (m *Migrator) validateTableColumns(tableName string, expectedColumns []string) error {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := m.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to get table info for %s: %w", tableName, err)
	}
	defer rows.Close()

	var existingColumns []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue interface{}

		err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
		if err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}
		existingColumns = append(existingColumns, name)
	}

	// Check all expected columns exist
	for _, expected := range expectedColumns {
		found := false
		for _, existing := range existingColumns {
			if existing == expected {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("table '%s' missing expected column '%s'", tableName, expected)
		}
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
		migrations, err := loadMigrationsFromSource(source)
		if err != nil {
			sourceStatus["error"] = err.Error()
			sourceStatus["migration_count"] = 0
		} else {
			sourceStatus["migration_count"] = len(migrations)
		}
		
		status["sources"].(map[string]interface{})[source.Name] = sourceStatus
	}
	
	return status, nil
}

// Close closes the migrator and database connections
func (m *Migrator) Close() error {
	var errs []string

	if m.db != nil {
		if err := m.db.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("database: %v", err))
		}
	}

	if m.migrate != nil {
		sourceErr, databaseErr := m.migrate.Close()
		if sourceErr != nil {
			errs = append(errs, fmt.Sprintf("migrate source: %v", sourceErr))
		}
		if databaseErr != nil {
			errs = append(errs, fmt.Sprintf("migrate database: %v", databaseErr))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing migrator: %s", strings.Join(errs, ", "))
	}

	return nil
}