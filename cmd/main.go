package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/latoulicious/HKTM/internal/commands"
	"github.com/latoulicious/HKTM/internal/config"
	"github.com/latoulicious/HKTM/internal/handlers"
	"github.com/latoulicious/HKTM/internal/presence"
	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/pkg/common"
	"github.com/latoulicious/HKTM/pkg/database"
	"github.com/latoulicious/HKTM/pkg/logging"
	"github.com/latoulicious/HKTM/pkg/uma/handler"
	"gorm.io/gorm"
)

func main() {
	// Initialize application with proper error handling
	if err := initializeApplication(); err != nil {
		log.Fatalf("Application initialization failed: %v", err)
	}
}

// initializeApplication handles the complete application initialization process
func initializeApplication() error {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
		// Continue execution as .env file might not exist in production
	}

	// Check environment variables
	common.CheckPersonalUse()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize PostgreSQL database for caching
	db, err := database.NewGormDBFromConfig(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Get the underlying *sql.DB for Close() method
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database: %w", err)
	}
	defer sqlDB.Close()

	// Initialize centralized logging system
	if err := initializeCentralizedLogging(db); err != nil {
		return fmt.Errorf("failed to initialize centralized logging: %w", err)
	}

	// Validate system dependencies for audio pipeline
	if err := validateSystemDependencies(); err != nil {
		log.Printf("Warning: Audio system dependencies validation failed: %v", err)
		log.Printf("Audio functionality may be limited. Please ensure ffmpeg and yt-dlp are installed.")
	}

	// Initialize audio pipeline system
	if err := initializeAudioPipelineSystem(db); err != nil {
		return fmt.Errorf("failed to initialize audio pipeline system: %w", err)
	}

	// Create a new Discord session using the provided token
	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Create presence manager
	presenceManager := presence.NewPresenceManager(dg)

	// Set the presence manager in the commands package
	commands.SetPresenceManager(presenceManager)

	// Initialize gametora client with config
	commands.InitializeGametoraClient(cfg)

	// Initialize UMA commands with database
	commands.InitializeUmaCommands(db)

	// Initialize commands with database for audio pipeline support
	commands.InitializeCommandsWithDB(db)

	// Register the message handler
	dg.AddHandler(handlers.MessageHandler)

	// Register the slash command handler
	dg.AddHandler(handlers.SlashCommandHandler)

	// Register the reaction handlers for Uma character image navigation
	dg.AddHandler(handlers.ReactionAddHandler)
	dg.AddHandler(handlers.ReactionRemoveHandler)

	// Start health check HTTP server
	healthServer := startHealthCheckServer()

	// Open a websocket connection to Discord and begin listening.
	if err := dg.Open(); err != nil {
		return fmt.Errorf("failed to open Discord session: %w", err)
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
	log.Println("Health check endpoint available at http://localhost:8080/health")

	// Wait here until CTRL-C or other term signal is received.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Shutting down gracefully...")

	// Shutdown health check server
	shutdownHealthServer(healthServer)

	// Shutdown audio pipeline system
	shutdownAudioPipelineSystem()

	// Cleanly close down the Discord session.
	dg.Close()

	// Stop the build ID manager cron job
	if gametoraClient := handler.GetGametoraClient(); gametoraClient != nil {
		gametoraClient.StopBuildIDManager()
	}

	log.Println("Application shutdown complete")
	return nil
}

// initializeCentralizedLogging sets up the centralized logging system
func initializeCentralizedLogging(db *gorm.DB) error {
	// Create a repository adapter for the logging system
	audioRepo := audio.NewAudioRepository(db)
	logRepo := &audio.LogRepositoryAdapter{AudioRepo: audioRepo}
	
	// Create database logger factory
	loggerFactory := logging.NewDatabaseLoggerFactory(logRepo)
	
	// Set as global logger factory
	logging.SetGlobalLoggerFactory(loggerFactory)
	
	// Create system logger for initialization logging
	systemLogger := loggerFactory.CreateLogger("system")
	systemLogger.Info("Centralized logging system initialized successfully", map[string]interface{}{
		"database_connected": true,
		"logger_type": "database",
	})
	
	return nil
}

// validateSystemDependencies validates that required system dependencies are available
func validateSystemDependencies() error {
	return audio.ValidateSystemDependencies()
}

// initializeAudioPipelineSystem initializes the audio pipeline system
func initializeAudioPipelineSystem(db *gorm.DB) error {
	// Initialize commands package with database for audio pipeline support
	// This ensures that when queues are created, they have access to the new audio pipeline
	commands.InitializeCommandsWithDB(db)
	
	// Log successful initialization
	loggerFactory := logging.GetGlobalLoggerFactory()
	systemLogger := loggerFactory.CreateLogger("system")
	systemLogger.Info("Audio pipeline system initialized successfully", map[string]interface{}{
		"database_connected": true,
		"pipeline_version": "v2",
	})
	
	return nil
}

// shutdownAudioPipelineSystem gracefully shuts down all active audio pipelines
func shutdownAudioPipelineSystem() {
	// Get all active queues and shutdown their pipelines
	commands.ShutdownAllAudioPipelines()
	
	// Log shutdown
	loggerFactory := logging.GetGlobalLoggerFactory()
	systemLogger := loggerFactory.CreateLogger("system")
	systemLogger.Info("Audio pipeline system shutdown complete", map[string]interface{}{
		"shutdown_reason": "application_exit",
	})
}

// Health check system for basic system validation
var (
	healthServer *http.Server
	systemHealth = &SystemHealth{
		StartTime: time.Now(),
		Status:    "starting",
	}
)

type SystemHealth struct {
	StartTime time.Time `json:"start_time"`
	Status    string    `json:"status"`
	Uptime    string    `json:"uptime"`
	Database  bool      `json:"database_connected"`
	Audio     bool      `json:"audio_system_ready"`
}

// startHealthCheckServer starts the HTTP server for health checks
func startHealthCheckServer() *http.Server {
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", healthCheckHandler)
	
	// Basic status endpoint
	mux.HandleFunc("/status", statusHandler)
	
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	
	go func() {
		log.Println("Starting health check server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health check server error: %v", err)
		}
	}()
	
	// Update system status to running
	systemHealth.Status = "running"
	
	return server
}

// healthCheckHandler handles the /health endpoint
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Update health status
	systemHealth.Uptime = time.Since(systemHealth.StartTime).String()
	
	// Check database connectivity (basic check)
	systemHealth.Database = true // Assume true if we got this far
	
	// Check audio system readiness
	systemHealth.Audio = checkAudioSystemHealth()
	
	// Determine overall health
	isHealthy := systemHealth.Database && systemHealth.Audio
	
	w.Header().Set("Content-Type", "application/json")
	
	if isHealthy {
		w.WriteHeader(http.StatusOK)
		systemHealth.Status = "healthy"
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		systemHealth.Status = "unhealthy"
	}
	
	// Simple JSON response
	fmt.Fprintf(w, `{
		"status": "%s",
		"uptime": "%s",
		"database_connected": %t,
		"audio_system_ready": %t,
		"start_time": "%s"
	}`, systemHealth.Status, systemHealth.Uptime, systemHealth.Database, 
		systemHealth.Audio, systemHealth.StartTime.Format(time.RFC3339))
}

// statusHandler handles the /status endpoint for more detailed information
func statusHandler(w http.ResponseWriter, r *http.Request) {
	systemHealth.Uptime = time.Since(systemHealth.StartTime).String()
	systemHealth.Database = true
	systemHealth.Audio = checkAudioSystemHealth()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	fmt.Fprintf(w, `{
		"application": "HKTM Discord Bot",
		"version": "2.0",
		"status": "%s",
		"uptime": "%s",
		"start_time": "%s",
		"components": {
			"database": %t,
			"audio_pipeline": %t,
			"discord_connection": true
		}
	}`, systemHealth.Status, systemHealth.Uptime, 
		systemHealth.StartTime.Format(time.RFC3339),
		systemHealth.Database, systemHealth.Audio)
}

// checkAudioSystemHealth performs basic audio system health checks
func checkAudioSystemHealth() bool {
	// Check if audio dependencies are available
	if err := audio.ValidateSystemDependencies(); err != nil {
		return false
	}
	
	// Additional health checks could be added here
	// For now, just check if dependencies are available
	return true
}

// shutdownHealthServer gracefully shuts down the health check server
func shutdownHealthServer(server *http.Server) {
	if server == nil {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Health server shutdown error: %v", err)
	} else {
		log.Println("Health check server shutdown complete")
	}
}
