package database

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// MigrationSource represents a source of database migrations
type MigrationSource struct {
	Name      string    // Human-readable name (e.g., "user-management", "sentipulse-core")
	Directory string    // File system path to migrations (for file-based sources)
	EmbedFS   *embed.FS // Embedded filesystem (for embedded migrations)
	Prefix    string    // Optional prefix for migration files (e.g., "user_", "app_")
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

// GetMigrationStats returns statistics about registered migrations
func GetMigrationStats() (map[string]int, error) {
	sources := GetRegisteredSources()
	stats := make(map[string]int)

	for _, source := range sources {
		count, err := countMigrationFiles(source)
		if err != nil {
			return nil, err
		}
		stats[source.Name] = count
	}

	return stats, nil
}

// countMigrationFiles counts the number of migration files in a source
func countMigrationFiles(source MigrationSource) (int, error) {
	if source.Directory != "" {
		return countMigrationFilesInDirectory(source.Directory)
	}

	if source.EmbedFS != nil {
		log.Printf("📦 Embedded FS migrations not yet implemented for: %s", source.Name)
		return 0, nil
	}

	return 0, fmt.Errorf("migration source %s has neither Directory nor EmbedFS specified", source.Name)
}

// countMigrationFilesInDirectory counts SQL files in a directory
func countMigrationFilesInDirectory(directory string) (int, error) {
	// Check if directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return 0, fmt.Errorf("migration directory does not exist: %s", directory)
	}

	// List SQL files
	entries, err := filepath.Glob(filepath.Join(directory, "*.sql"))
	if err != nil {
		return 0, fmt.Errorf("failed to list migration files: %w", err)
	}

	return len(entries), nil
}
