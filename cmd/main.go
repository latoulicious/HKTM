package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/latoulicious/HKTM/internal/commands"
	"github.com/latoulicious/HKTM/internal/config"
	"github.com/latoulicious/HKTM/internal/handlers"
	"github.com/latoulicious/HKTM/internal/presence"
	"github.com/latoulicious/HKTM/pkg/common"
	"github.com/latoulicious/HKTM/pkg/database"
	"github.com/latoulicious/HKTM/pkg/uma/handler"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Check environment variables
	common.CheckPersonalUse()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create a new Discord session using the provided token
	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}

	// Create presence manager
	presenceManager := presence.NewPresenceManager(dg)

	// Set the presence manager in the commands package
	commands.SetPresenceManager(presenceManager)

	// Initialize PostgreSQL database for caching
	db, err := database.NewGormDBFromConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Get the underlying *sql.DB for Close() method
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying database: %v", err)
	}
	defer sqlDB.Close()

	// Initialize gametora client with config
	commands.InitializeGametoraClient(cfg)

	// Initialize UMA commands with database
	commands.InitializeUmaCommands(db)

	// Register the message handler
	dg.AddHandler(handlers.MessageHandler)

	// Register the slash command handler
	dg.AddHandler(handlers.SlashCommandHandler)

	// Register the reaction handlers for Uma character image navigation
	dg.AddHandler(handlers.ReactionAddHandler)
	dg.AddHandler(handlers.ReactionRemoveHandler)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Fatalf("Failed to open Discord session: %v", err)
	}

	// Set initial presence
	presenceManager.UpdateDefaultPresence()

	// Start periodic presence updates
	presenceManager.StartPeriodicUpdates()

	// Start idle monitor
	idleMonitor := commands.GetIdleMonitor()
	idleMonitor(dg)

	common.EnforceGuildAndDev(cfg.OwnerID)

	log.Println("Bot is running. Press CTRL-C to exit.")
	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()

	// Stop the build ID manager cron job
	if gametoraClient := handler.GetGametoraClient(); gametoraClient != nil {
		gametoraClient.StopBuildIDManager()
	}
}
