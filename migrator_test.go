package database

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// TestMigrationPrefixes validates that multiple sources with different prefixes
// can have overlapping migration numbers without conflict
func TestMigrationPrefixes(t *testing.T) {
	// Create a temporary database for this test
	tempDir := t.TempDir()
	testDBFile := filepath.Join(tempDir, "test_prefixes.db")

	// Set DATABASE_FILE for this test
	originalDBFile := os.Getenv("DATABASE_FILE")
	os.Setenv("DATABASE_FILE", testDBFile)
	defer func() {
		if originalDBFile == "" {
			os.Unsetenv("DATABASE_FILE")
		} else {
			os.Setenv("DATABASE_FILE", originalDBFile)
		}
	}()

	// Create test migration directories
	userMigrationsDir := filepath.Join(tempDir, "user_migrations")
	appMigrationsDir := filepath.Join(tempDir, "app_migrations")

	if err := os.MkdirAll(userMigrationsDir, 0755); err != nil {
		t.Fatalf("Failed to create user migrations dir: %v", err)
	}
	if err := os.MkdirAll(appMigrationsDir, 0755); err != nil {
		t.Fatalf("Failed to create app migrations dir: %v", err)
	}

	// Create conflicting migration files (same numbers, different sources)

	// User source: migration 001 creates users table
	userMig001Up := `CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT);`
	userMig001Down := `DROP TABLE users;`

	// App source: migration 001 creates boards table
	appMig001Up := `CREATE TABLE boards (id TEXT PRIMARY KEY, name TEXT);`
	appMig001Down := `DROP TABLE boards;`

	// Write user migrations
	if err := ioutil.WriteFile(filepath.Join(userMigrationsDir, "001_create_users.up.sql"), []byte(userMig001Up), 0644); err != nil {
		t.Fatalf("Failed to write user migration: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(userMigrationsDir, "001_create_users.down.sql"), []byte(userMig001Down), 0644); err != nil {
		t.Fatalf("Failed to write user migration: %v", err)
	}

	// Write app migrations
	if err := ioutil.WriteFile(filepath.Join(appMigrationsDir, "001_create_boards.up.sql"), []byte(appMig001Up), 0644); err != nil {
		t.Fatalf("Failed to write app migration: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(appMigrationsDir, "001_create_boards.down.sql"), []byte(appMig001Down), 0644); err != nil {
		t.Fatalf("Failed to write app migration: %v", err)
	}

	// Clear any existing registrations for clean test
	globalRegistry.mu.Lock()
	globalRegistry.sources = []MigrationSource{}
	globalRegistry.mu.Unlock()

	// Register both sources with different prefixes
	RegisterMigrations(MigrationSource{
		Name:      "test-user-management",
		Directory: userMigrationsDir,
		Prefix:    "user_",
	})

	RegisterMigrations(MigrationSource{
		Name:      "test-app-core",
		Directory: appMigrationsDir,
		Prefix:    "app_",
	})

	// Attempt to run all migrations
	err := UpAll()
	if err != nil {
		t.Fatalf("Migration failed unexpectedly: %v", err)
	}

	// Check what actually happened in the database
	db, err := GetDB()
	if err != nil {
		t.Fatalf("Failed to get database connection: %v", err)
	}
	defer db.Close()

	// Check what tables were created
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("Failed to query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Failed to scan table name: %v", err)
		}
		tables = append(tables, name)
	}

	t.Logf("Tables created: %v", tables)

	// Check prefixed schema_migrations entries
	userMigRows, err := db.Query("SELECT version FROM user_schema_migrations ORDER BY version")
	if err != nil {
		t.Fatalf("Failed to query user_schema_migrations: %v", err)
	}
	defer userMigRows.Close()

	var userVersions []string
	for userMigRows.Next() {
		var version string
		if err := userMigRows.Scan(&version); err != nil {
			t.Fatalf("Failed to scan user version: %v", err)
		}
		userVersions = append(userVersions, version)
	}

	appMigRows, err := db.Query("SELECT version FROM app_schema_migrations ORDER BY version")
	if err != nil {
		t.Fatalf("Failed to query app_schema_migrations: %v", err)
	}
	defer appMigRows.Close()

	var appVersions []string
	for appMigRows.Next() {
		var version string
		if err := appMigRows.Scan(&version); err != nil {
			t.Fatalf("Failed to scan app version: %v", err)
		}
		appVersions = append(appVersions, version)
	}

	t.Logf("User migration versions: %v", userVersions)
	t.Logf("App migration versions: %v", appVersions)

	// SUCCESS: Each source should have its own migration tracking
	if len(userVersions) != 1 {
		t.Errorf("Expected 1 user migration to be recorded, got %d: %v", len(userVersions), userVersions)
	}
	if len(appVersions) != 1 {
		t.Errorf("Expected 1 app migration to be recorded, got %d: %v", len(appVersions), appVersions)
	}

	// Check if both tables exist (they should if no real conflict occurred)
	hasUsers := false
	hasBoards := false
	for _, table := range tables {
		if table == "users" {
			hasUsers = true
		}
		if table == "boards" {
			hasBoards = true
		}
	}

	if !hasUsers {
		t.Error("Expected 'users' table to be created")
	}
	if !hasBoards {
		t.Error("Expected 'boards' table to be created")
	}

	t.Logf("✅ Test validates the prefix solution:")
	t.Logf("   - Both migrations ran successfully with separate tracking")
	t.Logf("   - Prefixes are now properly used in separate schema tables")
	t.Logf("   - No conflicts between sources with same migration numbers")

	// TODO: After fixing the migrator, this test should:
	// 1. Succeed without error
	// 2. Create both users and boards tables
	// 3. Have prefixed entries in schema_migrations table (e.g., "user_1", "app_1")

	// Verify that we can see the conflicting sources
	sources := GetRegisteredSources()
	if len(sources) != 2 {
		t.Fatalf("Expected 2 registered sources, got %d", len(sources))
	}

	userSource := sources[0]
	appSource := sources[1]

	if userSource.Prefix != "user_" {
		t.Errorf("Expected user source prefix 'user_', got '%s'", userSource.Prefix)
	}
	if appSource.Prefix != "app_" {
		t.Errorf("Expected app source prefix 'app_', got '%s'", appSource.Prefix)
	}

	t.Logf("✅ Test demonstrates the prefix solution works")
	t.Logf("   - User source: %s (prefix: %s) → %d migrations", userSource.Name, userSource.Prefix, len(userVersions))
	t.Logf("   - App source: %s (prefix: %s) → %d migrations", appSource.Name, appSource.Prefix, len(appVersions))
	t.Logf("   - Both have migration 001, but no conflict due to separate tracking")
}
