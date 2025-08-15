package audio

import (
	"time"

	"github.com/google/uuid"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// AudioLoggerImpl implements the AudioLogger interface using zap logger
type AudioLoggerImpl struct {
	logger     *zap.Logger
	repository AudioRepository
	guildID    string
	config     *LoggerConfig
}

// NewAudioLogger creates a new AudioLogger implementation
func NewAudioLogger(repo AudioRepository, guildID string, config *LoggerConfig) AudioLogger {
	// Create zap logger configuration
	zapConfig := zap.NewProductionConfig()

	// Configure JSON formatting based on config
	if config.Format == "json" {
		zapConfig.Encoding = "json"
	} else {
		zapConfig.Encoding = "console"
	}

	// Set log level based on config
	switch config.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	// Build the logger
	logger, err := zapConfig.Build()
	if err != nil {
		// Fallback to a basic logger if configuration fails
		logger = zap.NewNop()
	}

	return &AudioLoggerImpl{
		logger:     logger,
		repository: repo,
		guildID:    guildID,
		config:     config,
	}
}

// Info logs an info message with optional fields
func (al *AudioLoggerImpl) Info(msg string, fields map[string]interface{}) {
	// Convert fields to zap fields
	zapFields := al.convertFieldsToZap(fields)

	// Log to console
	al.logger.Info(msg, zapFields...)

	// Save to database if enabled
	if al.config.SaveToDB && al.repository != nil {
		audioLog := &models.AudioLog{
			ID:        uuid.New(),
			GuildID:   al.guildID,
			Level:     "INFO",
			Message:   msg,
			Error:     "",
			Fields:    fields,
			Timestamp: time.Now(),
		}

		// Save to database (ignore errors to prevent logging loops)
		_ = al.repository.SaveLog(audioLog)
	}
}

// Error logs an error message with optional fields
func (al *AudioLoggerImpl) Error(msg string, err error, fields map[string]interface{}) {
	// Convert fields to zap fields and add error
	zapFields := al.convertFieldsToZap(fields)
	zapFields = append(zapFields, zap.Error(err))

	// Log to console
	al.logger.Error(msg, zapFields...)

	// Save to database if enabled
	if al.config.SaveToDB && al.repository != nil {
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}

		audioLog := &models.AudioLog{
			ID:        uuid.New(),
			GuildID:   al.guildID,
			Level:     "ERROR",
			Message:   msg,
			Error:     errorMsg,
			Fields:    fields,
			Timestamp: time.Now(),
		}

		// Save to database (ignore errors to prevent logging loops)
		_ = al.repository.SaveLog(audioLog)
	}
}

// Warn logs a warning message with optional fields
func (al *AudioLoggerImpl) Warn(msg string, fields map[string]interface{}) {
	// Convert fields to zap fields
	zapFields := al.convertFieldsToZap(fields)

	// Log to console
	al.logger.Warn(msg, zapFields...)

	// Save to database if enabled
	if al.config.SaveToDB && al.repository != nil {
		audioLog := &models.AudioLog{
			ID:        uuid.New(),
			GuildID:   al.guildID,
			Level:     "WARN",
			Message:   msg,
			Error:     "",
			Fields:    fields,
			Timestamp: time.Now(),
		}

		// Save to database (ignore errors to prevent logging loops)
		_ = al.repository.SaveLog(audioLog)
	}
}

// Debug logs a debug message with optional fields
func (al *AudioLoggerImpl) Debug(msg string, fields map[string]interface{}) {
	// Convert fields to zap fields
	zapFields := al.convertFieldsToZap(fields)

	// Log to console
	al.logger.Debug(msg, zapFields...)

	// Save to database if enabled
	if al.config.SaveToDB && al.repository != nil {
		audioLog := &models.AudioLog{
			ID:        uuid.New(),
			GuildID:   al.guildID,
			Level:     "DEBUG",
			Message:   msg,
			Error:     "",
			Fields:    fields,
			Timestamp: time.Now(),
		}

		// Save to database (ignore errors to prevent logging loops)
		_ = al.repository.SaveLog(audioLog)
	}
}

// convertFieldsToZap converts a map of fields to zap fields
func (al *AudioLoggerImpl) convertFieldsToZap(fields map[string]interface{}) []zap.Field {
	if fields == nil {
		return []zap.Field{}
	}

	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}

	return zapFields
}

// Close gracefully shuts down the logger
func (al *AudioLoggerImpl) Close() error {
	if al.logger != nil {
		return al.logger.Sync()
	}
	return nil
}
