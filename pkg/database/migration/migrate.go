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
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Migrations completed successfully!")
	return nil
}
