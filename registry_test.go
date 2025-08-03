package database

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMigrationRegistryCountsFiles verifies that the migration registry
// correctly counts SQL files from registered directories
func TestMigrationRegistryCountsFiles(t *testing.T) {
	// Create temporary directory with test migration files
	tempDir := t.TempDir()

	// Create test migration files
	upSQL := "CREATE TABLE test_table (id TEXT PRIMARY KEY);"
	downSQL := "DROP TABLE test_table;"

	os.WriteFile(filepath.Join(tempDir, "001_test_migration.up.sql"), []byte(upSQL), 0644)
	os.WriteFile(filepath.Join(tempDir, "001_test_migration.down.sql"), []byte(downSQL), 0644)

	// Create a test migration source
	source := MigrationSource{
		Name:      "test-source",
		Directory: tempDir,
		Prefix:    "test_",
	}

	// Test counting migrations from the source
	count, err := countMigrationFiles(source)
	if err != nil {
		t.Fatalf("Expected countMigrationFiles to succeed, got error: %v", err)
	}

	// Verify that we found the migration files
	if count != 2 {
		t.Fatalf("Expected to find 2 migration files (.up.sql and .down.sql), got %d", count)
	}
}

// TestSchemaConsistencyAfterMigrations verifies that the database schema
// is consistent after running all registered migrations
func TestSchemaConsistencyAfterMigrations(t *testing.T) {
	// Setup temporary database
	tempDir := t.TempDir()
	dbFile := filepath.Join(tempDir, "test.db")
	os.Setenv("DATABASE_FILE", dbFile)
	defer os.Unsetenv("DATABASE_FILE")

	// Create temporary migration directory with user table migration
	migrationDir := filepath.Join(tempDir, "migrations")
	os.MkdirAll(migrationDir, 0755)

	// Create a test user migration
	userMigrationSQL := `CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    given_name TEXT NOT NULL,
    family_name TEXT NOT NULL,
    picture TEXT,
    role TEXT NOT NULL CHECK (role IN ('admin', 'user')) DEFAULT 'user',
    api_key TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_api_key ON users(api_key) WHERE api_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);`

	os.WriteFile(filepath.Join(migrationDir, "001_create_users_table.up.sql"), []byte(userMigrationSQL), 0644)
	os.WriteFile(filepath.Join(migrationDir, "001_create_users_table.down.sql"), []byte("DROP TABLE users;"), 0644)

	// Register test migration source
	RegisterMigrations(MigrationSource{
		Name:      "test-user-management",
		Directory: migrationDir,
		Prefix:    "test_",
	})

	// Run all migrations
	err := RunAllMigrations()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Get database connection to verify schema
	db, err := GetDB()
	if err != nil {
		t.Fatalf("Failed to get database connection: %v", err)
	}
	defer db.Close()

	// Test 1: Verify users table exists with correct schema
	var userTableExists bool
	err = db.QueryRow(`
		SELECT EXISTS(
			SELECT name FROM sqlite_master 
			WHERE type='table' AND name='users'
		)
	`).Scan(&userTableExists)
	if err != nil {
		t.Fatalf("Failed to check if users table exists: %v", err)
	}
	if !userTableExists {
		t.Fatal("Expected users table to exist after migrations")
	}

	// Test 2: Verify users table has expected columns including email and api_key
	expectedColumns := []string{"id", "email", "given_name", "family_name", "role", "api_key"}
	for _, column := range expectedColumns {
		var columnExists bool
		err = db.QueryRow(`
			SELECT COUNT(*) > 0 FROM pragma_table_info('users') 
			WHERE name = ?
		`, column).Scan(&columnExists)
		if err != nil {
			t.Fatalf("Failed to check if column %s exists: %v", column, err)
		}
		if !columnExists {
			t.Errorf("Expected column '%s' to exist in users table", column)
		}
	}

	// Test 3: Verify email constraint exists (should be unique and not null)
	var emailInfo struct {
		notNull bool
		unique  bool
	}

	// Check if email is not null
	err = db.QueryRow(`
		SELECT [notnull] FROM pragma_table_info('users') 
		WHERE name = 'email'
	`).Scan(&emailInfo.notNull)
	if err != nil {
		t.Fatalf("Failed to check email column constraints: %v", err)
	}
	if !emailInfo.notNull {
		t.Error("Expected email column to have NOT NULL constraint")
	}

	// Check if unique index exists on email
	var emailUniqueIndexExists bool
	err = db.QueryRow(`
		SELECT EXISTS(
			SELECT name FROM sqlite_master 
			WHERE type='index' AND (
				name LIKE '%email%' OR 
				sql LIKE '%email%'
			)
		)
	`).Scan(&emailUniqueIndexExists)
	if err != nil {
		t.Fatalf("Failed to check email unique index: %v", err)
	}
	if !emailUniqueIndexExists {
		t.Error("Expected unique index on email column to exist")
	}
}
