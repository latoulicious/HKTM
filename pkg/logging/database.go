package logging

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DatabaseLogger implements Logger interface with database persistence
type DatabaseLogger struct {
	zapLogger  *zap.Logger
	component  string
	context    map[string]interface{}
	repository LogRepository
}

// NewDatabaseLogger creates a new DatabaseLogger
func NewDatabaseLogger(component string, repository LogRepository) *DatabaseLogger {
	// Create a production logger configuration
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	
	logger, err := config.Build()
	if err != nil {
		// Fallback to a basic logger if configuration fails
		logger = zap.NewNop()
	}

	return &DatabaseLogger{
		zapLogger:  logger,
		component:  component,
		context:    make(map[string]interface{}),
		repository: repository,
	}
}

// Info logs an info message
func (d *DatabaseLogger) Info(msg string, fields map[string]interface{}) {
	// Log to zap first
	zapFields := d.buildZapFields(fields)
	d.zapLogger.Info(fmt.Sprintf("[%s] %s", d.component, msg), zapFields...)
	
	// Save to database if repository is available
	if d.repository != nil {
		entry := d.buildLogEntry("INFO", msg, "", fields)
		if err := d.repository.SaveLog(entry); err != nil {
			// Log the error but don't fail the original log operation
			d.zapLogger.Error("Failed to save log to database", zap.Error(err))
		}
	}
}

// Error logs an error message
func (d *DatabaseLogger) Error(msg string, err error, fields map[string]interface{}) {
	// Log to zap first
	zapFields := d.buildZapFields(fields)
	if err != nil {
		zapFields = append(zapFields, zap.Error(err))
	}
	d.zapLogger.Error(fmt.Sprintf("[%s] %s", d.component, msg), zapFields...)
	
	// Save to database if repository is available
	if d.repository != nil {
		errorStr := ""
		if err != nil {
			errorStr = err.Error()
		}
		entry := d.buildLogEntry("ERROR", msg, errorStr, fields)
		if saveErr := d.repository.SaveLog(entry); saveErr != nil {
			// Log the error but don't fail the original log operation
			d.zapLogger.Error("Failed to save log to database", zap.Error(saveErr))
		}
	}
}

// Warn logs a warning message
func (d *DatabaseLogger) Warn(msg string, fields map[string]interface{}) {
	// Log to zap first
	zapFields := d.buildZapFields(fields)
	d.zapLogger.Warn(fmt.Sprintf("[%s] %s", d.component, msg), zapFields...)
	
	// Save to database if repository is available
	if d.repository != nil {
		entry := d.buildLogEntry("WARN", msg, "", fields)
		if err := d.repository.SaveLog(entry); err != nil {
			// Log the error but don't fail the original log operation
			d.zapLogger.Error("Failed to save log to database", zap.Error(err))
		}
	}
}

// Debug logs a debug message
func (d *DatabaseLogger) Debug(msg string, fields map[string]interface{}) {
	// Log to zap first
	zapFields := d.buildZapFields(fields)
	d.zapLogger.Debug(fmt.Sprintf("[%s] %s", d.component, msg), zapFields...)
	
	// Save to database if repository is available
	if d.repository != nil {
		entry := d.buildLogEntry("DEBUG", msg, "", fields)
		if err := d.repository.SaveLog(entry); err != nil {
			// Log the error but don't fail the original log operation
			d.zapLogger.Error("Failed to save log to database", zap.Error(err))
		}
	}
}

// WithPipeline creates a new logger with pipeline context
func (d *DatabaseLogger) WithPipeline(pipeline string) Logger {
	newContext := make(map[string]interface{})
	for k, v := range d.context {
		newContext[k] = v
	}
	newContext["pipeline"] = pipeline

	return &DatabaseLogger{
		zapLogger:  d.zapLogger,
		component:  d.component,
		context:    newContext,
		repository: d.repository,
	}
}

// WithContext creates a new logger with additional context
func (d *DatabaseLogger) WithContext(ctx map[string]interface{}) Logger {
	newContext := make(map[string]interface{})
	for k, v := range d.context {
		newContext[k] = v
	}
	for k, v := range ctx {
		newContext[k] = v
	}

	return &DatabaseLogger{
		zapLogger:  d.zapLogger,
		component:  d.component,
		context:    newContext,
		repository: d.repository,
	}
}

// buildZapFields converts map fields to zap fields
func (d *DatabaseLogger) buildZapFields(fields map[string]interface{}) []zap.Field {
	var zapFields []zap.Field

	// Add context fields first
	for k, v := range d.context {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	// Add provided fields
	if fields != nil {
		for k, v := range fields {
			zapFields = append(zapFields, zap.Any(k, v))
		}
	}

	return zapFields
}

// buildLogEntry creates a LogEntry for database persistence
func (d *DatabaseLogger) buildLogEntry(level, message, errorStr string, fields map[string]interface{}) LogEntry {
	// Merge context and provided fields
	allFields := make(map[string]interface{})
	for k, v := range d.context {
		allFields[k] = v
	}
	if fields != nil {
		for k, v := range fields {
			allFields[k] = v
		}
	}
	
	// Extract special fields
	guildID := ""
	userID := ""
	channelID := ""
	
	if val, ok := allFields["guild_id"].(string); ok {
		guildID = val
	}
	if val, ok := allFields["user_id"].(string); ok {
		userID = val
	}
	if val, ok := allFields["channel_id"].(string); ok {
		channelID = val
	}
	
	// Add timestamp to fields
	allFields["timestamp"] = time.Now()
	
	return LogEntry{
		GuildID:   guildID,
		Component: d.component,
		Level:     level,
		Message:   message,
		Error:     errorStr,
		Fields:    allFields,
		UserID:    userID,
		ChannelID: channelID,
	}
}