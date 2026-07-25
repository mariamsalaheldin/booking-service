package migrate

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mariamsalaheldin/booking-service/internal/repository"
)

func Run(db *repository.Postgres) error {
	migrationDir := "migrations"
	files, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}

	sort.Strings(migrationFiles)

	for _, filename := range migrationFiles {
		log.Printf("Running migration: %s", filename)
		content, err := os.ReadFile(filepath.Join(migrationDir, filename))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		_, err = db.DB.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		log.Printf("Migration completed: %s", filename)
	}

	log.Println("All migrations completed successfully")
	return nil
}
