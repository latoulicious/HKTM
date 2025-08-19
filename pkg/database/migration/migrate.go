package migration

import (
	"log"

	"github.com/latoulicious/HKTM/pkg/database/models"
	"gorm.io/gorm"
)

func RunMigration(db *gorm.DB) error {

	log.Println("Starting migrations...")

	// Create postgres extension for uuid
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		log.Fatalf("Failed to create uuid-ossp extension: %v", err)
	}

	log.Println("Running database migrations...")
	// Auto-migrate the models
	if err := db.AutoMigrate(
		&models.Character{},
		&models.CharacterImage{},
		&models.SupportCard{},
		&models.AudioError{},
		&models.AudioMetric{},
		&models.AudioLog{},
		&models.QueueTimeout{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Run specific migrations for centralized logging
	log.Println("Running centralized logging migrations...")
	if err := AddCentralizedLoggingColumns(db); err != nil {
		log.Fatalf("Failed to run centralized logging migration: %v", err)
	}

	log.Println("Migrations completed successfully!")
	return nil
}
