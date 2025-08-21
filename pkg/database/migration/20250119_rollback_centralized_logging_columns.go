package migration

import (
	"log"

	"gorm.io/gorm"
)

// RollbackCentralizedLoggingColumns removes component, user_id, and channel_id columns from audio_logs table
// This rollback migration can be used if we need to revert the centralized logging changes
func RollbackCentralizedLoggingColumns(db *gorm.DB) error {
	log.Println("Running rollback: Remove centralized logging columns from audio_logs table...")

	// Drop indexes first
	log.Println("Dropping indexes for centralized logging columns...")
	
	// Drop composite indexes
	if err := db.Exec("DROP INDEX IF EXISTS idx_audio_logs_component_guild").Error; err != nil {
		log.Printf("Warning: Failed to drop idx_audio_logs_component_guild: %v", err)
	}

	if err := db.Exec("DROP INDEX IF EXISTS idx_audio_logs_user_guild").Error; err != nil {
		log.Printf("Warning: Failed to drop idx_audio_logs_user_guild: %v", err)
	}

	// Drop individual column indexes
	if err := db.Exec("DROP INDEX IF EXISTS idx_audio_logs_component").Error; err != nil {
		log.Printf("Warning: Failed to drop idx_audio_logs_component: %v", err)
	}

	if err := db.Exec("DROP INDEX IF EXISTS idx_audio_logs_user_id").Error; err != nil {
		log.Printf("Warning: Failed to drop idx_audio_logs_user_id: %v", err)
	}

	if err := db.Exec("DROP INDEX IF EXISTS idx_audio_logs_channel_id").Error; err != nil {
		log.Printf("Warning: Failed to drop idx_audio_logs_channel_id: %v", err)
	}

	// Drop columns
	log.Println("Dropping centralized logging columns...")

	// Check and drop component column
	if db.Migrator().HasColumn(&AudioLogMigration{}, "component") {
		log.Println("Dropping component column from audio_logs table...")
		if err := db.Exec("ALTER TABLE audio_logs DROP COLUMN component").Error; err != nil {
			return err
		}
	}

	// Check and drop user_id column
	if db.Migrator().HasColumn(&AudioLogMigration{}, "user_id") {
		log.Println("Dropping user_id column from audio_logs table...")
		if err := db.Exec("ALTER TABLE audio_logs DROP COLUMN user_id").Error; err != nil {
			return err
		}
	}

	// Check and drop channel_id column
	if db.Migrator().HasColumn(&AudioLogMigration{}, "channel_id") {
		log.Println("Dropping channel_id column from audio_logs table...")
		if err := db.Exec("ALTER TABLE audio_logs DROP COLUMN channel_id").Error; err != nil {
			return err
		}
	}

	log.Println("Centralized logging columns rollback completed successfully!")
	return nil
}