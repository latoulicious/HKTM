package audio

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"github.com/latoulicious/HKTM/pkg/logging"
	"gorm.io/gorm"
)

// NewAudioPipelineWithDependencies creates a complete audio pipeline with all dependencies
// This factory function implements proper dependency injection using interfaces only
func NewAudioPipelineWithDependencies(db *gorm.DB, guildID string) (AudioPipeline, error) {
	// Create components in dependency order to prevent circular dependencies

	// Step 1: Create configuration provider (no dependencies)
	config, err := createConfigProvider()
	if err != nil {
		return nil, fmt.Errorf("config creation failed: %w", err)
	}

	// Step 2: Validate configuration and dependencies early
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	if err := config.ValidateDependencies(); err != nil {
		return nil, fmt.Errorf("dependency validation failed: %w", err)
	}

	// Step 3: Create repository (depends on database only)
	repo := createRepository(db)

	// Step 4: Create centralized logging factory and logger (depends on repository)
	loggerFactory := createLoggerFactory(repo)
	logger := createAudioLogger(loggerFactory, guildID)

	// Log successful dependency validation
	logger.Info("Binary dependencies validated successfully", CreateContextFieldsWithComponent(guildID, "", "", "factory"))

	// Step 5: Create individual components with interface injection only
	processor, err := createStreamProcessor(config, logger)
	if err != nil {
		return nil, fmt.Errorf("stream processor creation failed: %w", err)
	}

	encoder, err := createAudioEncoder(config, logger)
	if err != nil {
		return nil, fmt.Errorf("audio encoder creation failed: %w", err)
	}

	errorHandler := createErrorHandler(config, logger, repo, guildID)
	metrics := createMetricsCollector(repo, guildID)

	// Step 6: Wire controller with all interfaces
	controller := createPipelineController(processor, encoder, errorHandler, metrics, logger, config)

	// Step 7: Initialize the pipeline
	if err := controller.Initialize(); err != nil {
		return nil, fmt.Errorf("pipeline initialization failed: %w", err)
	}

	logger.Info("Audio pipeline created and initialized successfully", CreateContextFieldsWithComponent(guildID, "", "", "factory"))

	return controller, nil
}

// Separate factory functions for each component to prevent circular dependencies
// These functions create components with interface injection only

// createConfigProvider creates a ConfigProvider implementation
func createConfigProvider() (ConfigProvider, error) {
	return NewConfigManager()
}

// createRepository creates an AudioRepository implementation
func createRepository(db *gorm.DB) AudioRepository {
	return NewAudioRepository(db)
}

// createLoggerFactory creates a centralized logging factory with database persistence
func createLoggerFactory(repo AudioRepository) logging.LoggerFactory {
	// Create a repository adapter for the logging system
	logRepo := &LogRepositoryAdapter{AudioRepo: repo}
	return logging.NewDatabaseLoggerFactory(logRepo)
}

// createAudioLogger creates an AudioLogger using the centralized logging factory
func createAudioLogger(factory logging.LoggerFactory, guildID string) AudioLogger {
	// Get centralized logger for audio component
	baseLogger := factory.CreateAudioLogger(guildID)

	// Wrap with AudioLogger adapter to match the audio package interface
	return &AudioLoggerAdapter{
		logger:  baseLogger,
		guildID: guildID,
	}
}

// createStreamProcessor creates a StreamProcessor implementation
func createStreamProcessor(config ConfigProvider, logger AudioLogger) (StreamProcessor, error) {
	ffmpegConfig := config.GetFFmpegConfig()
	ytdlpConfig := config.GetYtDlpConfig()
	return NewFFmpegProcessor(ffmpegConfig, ytdlpConfig, logger), nil
}

// createAudioEncoder creates an AudioEncoder implementation
func createAudioEncoder(config ConfigProvider, logger AudioLogger) (AudioEncoder, error) {
	opusConfig := config.GetOpusConfig()
	return NewOpusProcessor(opusConfig, logger), nil
}

// createErrorHandler creates an ErrorHandler implementation
func createErrorHandler(config ConfigProvider, logger AudioLogger, repo AudioRepository, guildID string) ErrorHandler {
	retryConfig := config.GetRetryConfig()
	return NewBasicErrorHandler(retryConfig, logger, repo, guildID)
}

// createMetricsCollector creates a MetricsCollector implementation
func createMetricsCollector(repo AudioRepository, guildID string) MetricsCollector {
	return NewBasicMetrics(repo, guildID)
}

// createPipelineController creates the main AudioPipelineController with all dependencies
func createPipelineController(
	processor StreamProcessor,
	encoder AudioEncoder,
	errorHandler ErrorHandler,
	metrics MetricsCollector,
	logger AudioLogger,
	config ConfigProvider,
) AudioPipeline {
	return NewAudioPipelineController(processor, encoder, errorHandler, metrics, logger, config)
}

// LogRepositoryAdapter adapts AudioRepository to logging.LogRepository interface
type LogRepositoryAdapter struct {
	AudioRepo AudioRepository
}

// SaveLog implements logging.LogRepository interface
func (l *LogRepositoryAdapter) SaveLog(entry logging.LogEntry) error {
	// Convert logging.LogEntry to models.AudioLog
	audioLog := &models.AudioLog{
		ID:        uuid.New(), // Generate unique UUID for each log entry
		GuildID:   entry.GuildID,
		Level:     entry.Level,
		Message:   entry.Message,
		Error:     entry.Error,
		Fields:    entry.Fields,
		Timestamp: time.Now(),
	}

	// Ensure Fields map exists
	if audioLog.Fields == nil {
		audioLog.Fields = make(map[string]interface{})
	}

	// Store UserID and ChannelID in Fields since AudioLog model doesn't have them directly
	if entry.UserID != "" {
		audioLog.Fields["user_id"] = entry.UserID
	}
	if entry.ChannelID != "" {
		audioLog.Fields["channel_id"] = entry.ChannelID
	}
	if entry.Component != "" {
		audioLog.Fields["component"] = entry.Component
	}

	return l.AudioRepo.SaveLog(audioLog)
}

// AudioLoggerAdapter adapts logging.Logger to AudioLogger interface
type AudioLoggerAdapter struct {
	logger  logging.Logger
	guildID string
}

// WithPipeline creates a new AudioLogger with pipeline-specific context
func (a *AudioLoggerAdapter) WithPipeline(pipeline string) AudioLogger {
	pipelineLogger := a.logger.WithPipeline(pipeline)
	return &AudioLoggerAdapter{
		logger:  pipelineLogger,
		guildID: a.guildID,
	}
}

// WithContext creates a new AudioLogger with additional context
func (a *AudioLoggerAdapter) WithContext(ctx map[string]interface{}) AudioLogger {
	contextLogger := a.logger.WithContext(ctx)
	return &AudioLoggerAdapter{
		logger:  contextLogger,
		guildID: a.guildID,
	}
}

// Info implements AudioLogger interface
func (a *AudioLoggerAdapter) Info(msg string, fields map[string]interface{}) {
	// Ensure guild_id is always present in fields for audio logging
	enrichedFields := a.enrichWithGuildID(fields)
	a.logger.Info(msg, enrichedFields)
}

// Error implements AudioLogger interface
func (a *AudioLoggerAdapter) Error(msg string, err error, fields map[string]interface{}) {
	// Ensure guild_id is always present in fields for audio logging
	enrichedFields := a.enrichWithGuildID(fields)
	a.logger.Error(msg, err, enrichedFields)
}

// Warn implements AudioLogger interface
func (a *AudioLoggerAdapter) Warn(msg string, fields map[string]interface{}) {
	// Ensure guild_id is always present in fields for audio logging
	enrichedFields := a.enrichWithGuildID(fields)
	a.logger.Warn(msg, enrichedFields)
}

// Debug implements AudioLogger interface
func (a *AudioLoggerAdapter) Debug(msg string, fields map[string]interface{}) {
	// Ensure guild_id is always present in fields for audio logging
	enrichedFields := a.enrichWithGuildID(fields)
	a.logger.Debug(msg, enrichedFields)
}

// enrichWithGuildID ensures guild_id is present in the fields
func (a *AudioLoggerAdapter) enrichWithGuildID(fields map[string]interface{}) map[string]interface{} {
	if fields == nil {
		fields = make(map[string]interface{})
	}

	// Add guild_id if not already present
	if _, exists := fields["guild_id"]; !exists && a.guildID != "" {
		fields["guild_id"] = a.guildID
	}

	return fields
}

// Default configuration values
var (
	DefaultPipelineConfig = &PipelineConfig{
		RetryCount:     3,
		TimeoutSeconds: 30,
		LogLevel:       "info",
		FFmpegOptions:  []string{"-reconnect", "1", "-reconnect_delay_max", "5"},
	}

	DefaultFFmpegConfig = &FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{"-reconnect", "1", "-reconnect_delay_max", "5"},
	}

	DefaultYtDlpConfig = &YtDlpConfig{
		BinaryPath: "yt-dlp",
		CustomArgs: []string{"--no-playlist", "--extract-flat"},
	}

	DefaultOpusConfig = &OpusConfig{
		Bitrate:   128000,
		FrameSize: 960,
	}

	DefaultRetryConfig = &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  2000000000,  // 2 seconds in nanoseconds
		MaxDelay:   30000000000, // 30 seconds in nanoseconds
		Multiplier: 2.0,
	}

	DefaultLoggerConfig = &LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: true,
	}
)

// ShutdownAudioPipeline gracefully shuts down an audio pipeline
// This function should be called during application shutdown to ensure proper cleanup
func ShutdownAudioPipeline(pipeline AudioPipeline) error {
	if pipeline == nil {
		return nil
	}

	if !pipeline.IsInitialized() {
		return nil // Not initialized, nothing to shutdown
	}

	return pipeline.Shutdown()
}

// ValidateSystemDependencies validates that all required system dependencies are available
// This function can be called during application startup to ensure the system is ready
func ValidateSystemDependencies() error {
	// Use default binary paths for validation
	return ValidateAllBinaryDependencies("ffmpeg", "yt-dlp")
}
