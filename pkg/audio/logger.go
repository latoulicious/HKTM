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
	dbFailures int // Track consecutive database failures
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
		dbFailures: 0,
	}
}

// Info logs an info message with optional fields
func (al *AudioLoggerImpl) Info(msg string, fields map[string]interface{}) {
	// Convert fields to zap fields
	zapFields := al.convertFieldsToZap(fields)

	// Log to console
	al.logger.Info(msg, zapFields...)

	// Save to database with proper error handling
	audioLog := &models.AudioLog{
		ID:        uuid.New(),
		GuildID:   al.guildID,
		Level:     "INFO",
		Message:   msg,
		Error:     "",
		Fields:    fields,
		Timestamp: time.Now(),
	}
	al.saveToDatabase(audioLog)
}

// Error logs an error message with optional fields
func (al *AudioLoggerImpl) Error(msg string, err error, fields map[string]interface{}) {
	// Convert fields to zap fields and add error
	zapFields := al.convertFieldsToZap(fields)
	zapFields = append(zapFields, zap.Error(err))

	// Log to console
	al.logger.Error(msg, zapFields...)

	// Save to database with proper error handling
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
	al.saveToDatabase(audioLog)
}

// Warn logs a warning message with optional fields
func (al *AudioLoggerImpl) Warn(msg string, fields map[string]interface{}) {
	// Convert fields to zap fields
	zapFields := al.convertFieldsToZap(fields)

	// Log to console
	al.logger.Warn(msg, zapFields...)

	// Save to database with proper error handling
	audioLog := &models.AudioLog{
		ID:        uuid.New(),
		GuildID:   al.guildID,
		Level:     "WARN",
		Message:   msg,
		Error:     "",
		Fields:    fields,
		Timestamp: time.Now(),
	}
	al.saveToDatabase(audioLog)
}

// Debug logs a debug message with optional fields
func (al *AudioLoggerImpl) Debug(msg string, fields map[string]interface{}) {
	// Convert fields to zap fields
	zapFields := al.convertFieldsToZap(fields)

	// Log to console
	al.logger.Debug(msg, zapFields...)

	// Save to database with proper error handling
	audioLog := &models.AudioLog{
		ID:        uuid.New(),
		GuildID:   al.guildID,
		Level:     "DEBUG",
		Message:   msg,
		Error:     "",
		Fields:    fields,
		Timestamp: time.Now(),
	}
	al.saveToDatabase(audioLog)
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

// saveToDatabase attempts to save a log entry to the database with proper error handling
func (al *AudioLoggerImpl) saveToDatabase(audioLog *models.AudioLog) {
	if !al.config.SaveToDB || al.repository == nil {
		return
	}

	// Skip database logging if we've had too many consecutive failures
	// This prevents infinite loops and reduces overhead when DB is down
	const maxConsecutiveFailures = 5
	if al.dbFailures >= maxConsecutiveFailures {
		return
	}

	err := al.repository.SaveLog(audioLog)
	if err != nil {
		al.dbFailures++
		// Log database failure to console only (avoid recursion)
		al.logger.Warn("Failed to save log to database",
			zap.Error(err),
			zap.Int("consecutive_failures", al.dbFailures),
			zap.String("log_level", audioLog.Level),
			zap.String("log_message", audioLog.Message),
		)

		// If we've reached max failures, log a warning about disabling DB logging
		if al.dbFailures >= maxConsecutiveFailures {
			al.logger.Error("Too many consecutive database logging failures, disabling database logging",
				zap.Int("max_failures", maxConsecutiveFailures),
			)
		}
	} else {
		// Reset failure counter on successful save
		al.dbFailures = 0
	}
}

// ResetDatabaseFailures resets the database failure counter
// This can be called when database connectivity is restored
func (al *AudioLoggerImpl) ResetDatabaseFailures() {
	if al.dbFailures > 0 {
		al.logger.Info("Resetting database logging failure counter",
			zap.Int("previous_failures", al.dbFailures),
		)
		al.dbFailures = 0
	}
}

// GetDatabaseFailures returns the current number of consecutive database failures
func (al *AudioLoggerImpl) GetDatabaseFailures() int {
	return al.dbFailures
}

// Close gracefully shuts down the logger
func (al *AudioLoggerImpl) Close() error {
	if al.logger != nil {
		return al.logger.Sync()
	}
	return nil
}
