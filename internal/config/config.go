package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken string
	OwnerID      string

	// Cron configuration
	CronEnabled  bool
	CronSchedule string

	// Database configuration
	DatabaseURL string
}

var (
	ErrDiscordTokenNotSet = os.ErrInvalid
	ErrOwnerIDNotSet      = os.ErrInvalid
	ErrDBPathNotSet       = os.ErrInvalid
)

func LoadConfig() (*Config, error) {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	discordToken := os.Getenv("DISCORD_TOKEN")
	if discordToken == "" {
		return nil, ErrDiscordTokenNotSet
	}

	ownerID := os.Getenv("BOT_OWNER_ID")
	if ownerID == "" {
		return nil, ErrOwnerIDNotSet
	}

	// Cron configuration with defaults
	cronEnabled := true // Default to enabled
	if enabled := os.Getenv("CRON_ENABLED"); enabled != "" {
		cronEnabled = enabled == "true" || enabled == "1"
	}

	cronSchedule := "0 0 */6 * * *" // Default: every 6 hours
	if schedule := os.Getenv("CRON_SCHEDULE"); schedule != "" {
		cronSchedule = schedule
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, ErrDBPathNotSet
	}

	return &Config{
		DiscordToken: discordToken,
		OwnerID:      ownerID,
		CronEnabled:  cronEnabled,
		CronSchedule: cronSchedule,
		DatabaseURL:  databaseURL,
	}, nil
}
