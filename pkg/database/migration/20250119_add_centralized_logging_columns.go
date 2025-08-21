package migration

import (
	"log"

	"gorm.io/gorm"
)

// AddCentralizedLoggingColumns adds component, user_id, and channel_id columns to audio_logs table
// This migration supports the centralized logging system (Requirements 8.5, 8.6)
func AddCentralizedLoggingColumns(db *gorm.DB) error {
	log.Println("Running migration: Add centralized logging columns to audio_logs table...")

	// Check if component column exists, if not add it
	if !db.Migrator().HasColumn(&AudioLogMigration{}, "component") {
		log.Println("Adding component column to audio_logs table...")
		if err := db.Exec("ALTER TABLE audio_logs ADD COLUMN component VARCHAR(50) DEFAULT 'audio'").Error; err != nil {
			return err
		}
		
		// Update existing records to have 'audio' as component
		if err := db.Exec("UPDATE audio_logs SET component = 'audio' WHERE component IS NULL OR component = ''").Error; err != nil {
			return err
		}
		
		// Make component NOT NULL after setting default values
		if err := db.Exec("ALTER TABLE audio_logs ALTER COLUMN component SET NOT NULL").Error; err != nil {
			return err
		}
	}

	// Check if user_id column exists, if not add it
	if !db.Migrator().HasColumn(&AudioLogMigration{}, "user_id") {
		log.Println("Adding user_id column to audio_logs table...")
		if err := db.Exec("ALTER TABLE audio_logs ADD COLUMN user_id VARCHAR(50)").Error; err != nil {
			return err
		}
	}

	// Check if channel_id column exists, if not add it
	if !db.Migrator().HasColumn(&AudioLogMigration{}, "channel_id") {
		log.Println("Adding channel_id column to audio_logs table...")
		if err := db.Exec("ALTER TABLE audio_logs ADD COLUMN channel_id VARCHAR(50)").Error; err != nil {
			return err
		}
	}

	// Create indexes for new columns to support efficient querying
	log.Println("Creating indexes for new columns...")
	
	// Index for component column
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audio_logs_component ON audio_logs(component)").Error; err != nil {
		return err
	}

	// Index for user_id column
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audio_logs_user_id ON audio_logs(user_id)").Error; err != nil {
		return err
	}

	// Index for channel_id column
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audio_logs_channel_id ON audio_logs(channel_id)").Error; err != nil {
		return err
	}

	// Composite index for common query patterns (component + guild_id)
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audio_logs_component_guild ON audio_logs(component, guild_id)").Error; err != nil {
		return err
	}

	// Composite index for user context queries (user_id + guild_id)
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_audio_logs_user_guild ON audio_logs(user_id, guild_id) WHERE user_id IS NOT NULL").Error; err != nil {
		return err
	}

	log.Println("Centralized logging columns migration completed successfully!")
	return nil
}

// AudioLogMigration is a temporary struct for migration column checks
type AudioLogMigration struct {
	Component string `gorm:"column:component"`
	UserID    string `gorm:"column:user_id"`
	ChannelID string `gorm:"column:channel_id"`
}

// TableName returns the table name for migration checks
func (AudioLogMigration) TableName() string {
	return "audio_logs"
}