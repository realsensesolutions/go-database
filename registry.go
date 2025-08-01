package database

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// MigrationSource represents a source of database migrations
type MigrationSource struct {
	Name      string    // Human-readable name (e.g., "user-management", "sentipulse-core")
	Directory string    // File system path to migrations (for file-based sources)
	EmbedFS   *embed.FS // Embedded filesystem (for embedded migrations)
	Prefix    string    // Optional prefix for migration files (e.g., "user_", "app_")
}

// Migration represents a single database migration
type Migration struct {
	Version     uint   // Migration version number
	Name        string // Migration name/description
	Source      string // Source package name
	UpContent   string // SQL content for up migration
	DownContent string // SQL content for down migration
}

// Registry manages all registered migration sources
type Registry struct {
	mu      sync.RWMutex
	sources []MigrationSource
}

// Global registry instance
var globalRegistry = &Registry{
	sources: make([]MigrationSource, 0),
}

// RegisterMigrations registers a migration source with the global registry
func RegisterMigrations(source MigrationSource) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	log.Printf("📦 Registering migration source: %s", source.Name)
	globalRegistry.sources = append(globalRegistry.sources, source)
}

// GetRegisteredSources returns all registered migration sources
func GetRegisteredSources() []MigrationSource {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	// Return a copy to prevent external modification
	sources := make([]MigrationSource, len(globalRegistry.sources))
	copy(sources, globalRegistry.sources)
	return sources
}

// GetAllMigrations discovers and returns all migrations from registered sources
func GetAllMigrations() ([]Migration, error) {
	sources := GetRegisteredSources()
	if len(sources) == 0 {
		log.Printf("⚠️  No migration sources registered")
		return []Migration{}, nil
	}

	var allMigrations []Migration

	for _, source := range sources {
		migrations, err := loadMigrationsFromSource(source)
		if err != nil {
			return nil, fmt.Errorf("failed to load migrations from source %s: %w", source.Name, err)
		}
		allMigrations = append(allMigrations, migrations...)
	}

	// Sort migrations by version number
	sort.Slice(allMigrations, func(i, j int) bool {
		return allMigrations[i].Version < allMigrations[j].Version
	})

	log.Printf("📋 Discovered %d migrations from %d sources", len(allMigrations), len(sources))
	return allMigrations, nil
}

// loadMigrationsFromSource loads migrations from a specific source
func loadMigrationsFromSource(source MigrationSource) ([]Migration, error) {
	if source.EmbedFS != nil {
		return loadMigrationsFromEmbed(source)
	} else if source.Directory != "" {
		return loadMigrationsFromDirectory(source)
	}

	return nil, fmt.Errorf("migration source %s has neither Directory nor EmbedFS specified", source.Name)
}

// loadMigrationsFromDirectory loads migrations from a filesystem directory
func loadMigrationsFromDirectory(source MigrationSource) ([]Migration, error) {
	log.Printf("📂 Loading migrations from directory: %s", source.Directory)

	// Read directory contents
	entries, err := filepath.Glob(filepath.Join(source.Directory, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to list migration files: %w", err)
	}

	// Parse migration files into pairs (up/down)
	migrationMap := make(map[uint]*Migration)

	for _, entry := range entries {
		filename := filepath.Base(entry)

		// Parse filename: "001_migration_name.up.sql" or "001_migration_name.down.sql"
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			continue // Skip files that don't match pattern
		}

		// Extract version number
		versionStr := parts[0]
		version := uint(0)
		if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
			continue // Skip files with invalid version numbers
		}

		// Extract migration name and type
		nameAndType := strings.Join(parts[1:], "_")
		if strings.HasSuffix(nameAndType, ".up.sql") {
			name := strings.TrimSuffix(nameAndType, ".up.sql")
			content, err := os.ReadFile(entry)
			if err != nil {
				return nil, fmt.Errorf("failed to read migration file %s: %w", entry, err)
			}

			// Get or create migration entry
			if migrationMap[version] == nil {
				migrationMap[version] = &Migration{
					Version: version,
					Name:    name,
					Source:  source.Name,
				}
			}
			migrationMap[version].UpContent = string(content)

		} else if strings.HasSuffix(nameAndType, ".down.sql") {
			name := strings.TrimSuffix(nameAndType, ".down.sql")
			content, err := os.ReadFile(entry)
			if err != nil {
				return nil, fmt.Errorf("failed to read migration file %s: %w", entry, err)
			}

			// Get or create migration entry
			if migrationMap[version] == nil {
				migrationMap[version] = &Migration{
					Version: version,
					Name:    name,
					Source:  source.Name,
				}
			}
			migrationMap[version].DownContent = string(content)
		}
	}

	// Convert map to slice
	var migrations []Migration
	for _, migration := range migrationMap {
		migrations = append(migrations, *migration)
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	log.Printf("✅ Loaded %d migrations from %s", len(migrations), source.Name)
	return migrations, nil
}

// loadMigrationsFromEmbed loads migrations from an embedded filesystem
func loadMigrationsFromEmbed(source MigrationSource) ([]Migration, error) {
	// This is a placeholder - in a real implementation, you would:
	// 1. Walk the embedded filesystem
	// 2. Find migration files
	// 3. Parse versions and contents

	log.Printf("📦 Loading migrations from embedded FS: %s", source.Name)

	// For now, return empty slice - this will be implemented when packages register
	return []Migration{}, nil
}

// GetMigrationStats returns statistics about registered migrations
func GetMigrationStats() (map[string]int, error) {
	sources := GetRegisteredSources()
	stats := make(map[string]int)

	for _, source := range sources {
		migrations, err := loadMigrationsFromSource(source)
		if err != nil {
			return nil, err
		}
		stats[source.Name] = len(migrations)
	}

	return stats, nil
}

// ClearRegistry clears all registered migration sources (useful for testing)
func ClearRegistry() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.sources = make([]MigrationSource, 0)
}

// validateMigrationSequence ensures migrations have sequential version numbers
func validateMigrationSequence(migrations []Migration) error {
	if len(migrations) == 0 {
		return nil
	}

	expectedVersion := uint(1)
	for _, migration := range migrations {
		if migration.Version != expectedVersion {
			return fmt.Errorf("migration sequence gap: expected version %d, found %d", expectedVersion, migration.Version)
		}
		expectedVersion++
	}

	return nil
}

// parseMigrationFilename extracts version and name from a migration filename
// Example: "000001_create_users_table.up.sql" -> version=1, name="create_users_table"
func parseMigrationFilename(filename string) (version uint, name string, isUp bool, err error) {
	base := filepath.Base(filename)

	// Remove .sql extension
	if !strings.HasSuffix(base, ".sql") {
		return 0, "", false, fmt.Errorf("migration file must have .sql extension: %s", filename)
	}
	base = strings.TrimSuffix(base, ".sql")

	// Check if it's up or down migration
	if strings.HasSuffix(base, ".up") {
		isUp = true
		base = strings.TrimSuffix(base, ".up")
	} else if strings.HasSuffix(base, ".down") {
		isUp = false
		base = strings.TrimSuffix(base, ".down")
	} else {
		return 0, "", false, fmt.Errorf("migration file must end with .up.sql or .down.sql: %s", filename)
	}

	// Split version and name
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return 0, "", false, fmt.Errorf("migration filename must be in format VERSION_name: %s", filename)
	}

	// Parse version number
	var versionNum uint
	if _, err := fmt.Sscanf(parts[0], "%d", &versionNum); err != nil {
		return 0, "", false, fmt.Errorf("invalid version number in filename %s: %w", filename, err)
	}

	return versionNum, parts[1], isUp, nil
}
