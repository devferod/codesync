package database

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
)

// Migrations embeds SQL migration files
//go:embed migrations/*.sql
var migrations embed.FS

// RunMigrations executes all SQL migration files in order
func (db *DB) RunMigrations() error {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files by name to ensure order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		migrationName := entry.Name()
		content, err := migrations.ReadFile(path.Join("migrations", migrationName))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", migrationName, err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migrationName, err)
		}

		fmt.Printf("Applied migration: %s\n", migrationName)
	}

	return nil
}
