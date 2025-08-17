package logging

import (
	"fmt"
	"sync"
)

// DefaultLoggerFactory implements LoggerFactory using zap loggers
type DefaultLoggerFactory struct {
	loggers map[string]Logger
	mu      sync.RWMutex
}

// NewLoggerFactory creates a new logger factory
func NewLoggerFactory() LoggerFactory {
	return &DefaultLoggerFactory{
		loggers: make(map[string]Logger),
	}
}

// CreateLogger creates a basic logger for the specified component
func (f *DefaultLoggerFactory) CreateLogger(component string) Logger {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if logger already exists
	if logger, exists := f.loggers[component]; exists {
		return logger
	}

	// Create new zap logger
	zapLogger, err := NewZapLogger(component)
	if err != nil {
		// Fallback to a basic logger if zap fails
		// In a real implementation, you might want to handle this differently
		panic(fmt.Sprintf("Failed to create logger for component %s: %v", component, err))
	}

	f.loggers[component] = zapLogger
	return zapLogger
}

// CreateAudioLogger creates a logger specifically for audio pipeline operations
func (f *DefaultLoggerFactory) CreateAudioLogger(guildID string) Logger {
	baseLogger := f.CreateLogger("audio")
	return NewAudioPipelineLogger(baseLogger, guildID)
}

// CreateCommandLogger creates a logger for Discord command operations
func (f *DefaultLoggerFactory) CreateCommandLogger(commandName string) Logger {
	baseLogger := f.CreateLogger("commands")
	return NewCommandLogger(baseLogger, commandName)
}

// DatabaseLoggerFactory extends the default factory with database persistence
type DatabaseLoggerFactory struct {
	*DefaultLoggerFactory
	repository LogRepository
}

// LogRepository defines the interface for persisting logs to database
type LogRepository interface {
	SaveLog(entry LogEntry) error
}

// LogEntry represents a log entry for database storage
type LogEntry struct {
	Component string                 `json:"component"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Error     string                 `json:"error,omitempty"`
	Fields    map[string]interface{} `json:"fields"`
	GuildID   string                 `json:"guild_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	ChannelID string                 `json:"channel_id,omitempty"`
}

// NewDatabaseLoggerFactory creates a logger factory with database persistence
func NewDatabaseLoggerFactory(repository LogRepository) LoggerFactory {
	return &DatabaseLoggerFactory{
		DefaultLoggerFactory: &DefaultLoggerFactory{
			loggers: make(map[string]Logger),
		},
		repository: repository,
	}
}

// CreateLogger creates a database-backed logger for the specified component
func (f *DatabaseLoggerFactory) CreateLogger(component string) Logger {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if logger already exists
	if logger, exists := f.loggers[component]; exists {
		return logger
	}

	// Create new database-backed logger
	baseLogger, err := NewZapLogger(component)
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger for component %s: %v", component, err))
	}

	// Wrap with database persistence
	dbLogger := NewDatabaseLogger(baseLogger, f.repository)
	f.loggers[component] = dbLogger
	return dbLogger
}

// DatabaseLogger wraps a base logger with database persistence
type DatabaseLogger struct {
	base       Logger
	repository LogRepository
}

// NewDatabaseLogger creates a new database-backed logger
func NewDatabaseLogger(base Logger, repository LogRepository) Logger {
	return &DatabaseLogger{
		base:       base,
		repository: repository,
	}
}

// Info logs informational messages and persists to database
func (d *DatabaseLogger) Info(msg string, fields map[string]interface{}) {
	d.base.Info(msg, fields)
	d.persistLog("INFO", msg, nil, fields)
}

// Error logs error messages and persists to database
func (d *DatabaseLogger) Error(msg string, err error, fields map[string]interface{}) {
	d.base.Error(msg, err, fields)
	d.persistLog("ERROR", msg, err, fields)
}

// Warn logs warning messages and persists to database
func (d *DatabaseLogger) Warn(msg string, fields map[string]interface{}) {
	d.base.Warn(msg, fields)
	d.persistLog("WARN", msg, nil, fields)
}

// Debug logs debug messages and persists to database
func (d *DatabaseLogger) Debug(msg string, fields map[string]interface{}) {
	d.base.Debug(msg, fields)
	d.persistLog("DEBUG", msg, nil, fields)
}

// WithPipeline creates a new logger with pipeline context
func (d *DatabaseLogger) WithPipeline(pipeline string) Logger {
	return &DatabaseLogger{
		base:       d.base.WithPipeline(pipeline),
		repository: d.repository,
	}
}

// WithContext creates a new logger with additional context fields
func (d *DatabaseLogger) WithContext(ctx map[string]interface{}) Logger {
	return &DatabaseLogger{
		base:       d.base.WithContext(ctx),
		repository: d.repository,
	}
}

// persistLog saves the log entry to the database
func (d *DatabaseLogger) persistLog(level, message string, err error, fields map[string]interface{}) {
	entry := LogEntry{
		Level:   level,
		Message: message,
		Fields:  fields,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Extract common fields
	if component, ok := fields["component"].(string); ok {
		entry.Component = component
	}
	if guildID, ok := fields["guild_id"].(string); ok {
		entry.GuildID = guildID
	}
	if userID, ok := fields["user_id"].(string); ok {
		entry.UserID = userID
	}
	if channelID, ok := fields["channel_id"].(string); ok {
		entry.ChannelID = channelID
	}

	// Save to database (non-blocking to avoid impacting performance)
	go func() {
		if saveErr := d.repository.SaveLog(entry); saveErr != nil {
			// Log the error to the base logger (without database persistence to avoid recursion)
			d.base.Error("Failed to persist log to database", saveErr, map[string]interface{}{
				"original_message": message,
				"original_level":   level,
			})
		}
	}()
}

// GlobalLoggerFactory provides a singleton logger factory instance
var (
	globalFactory LoggerFactory
	factoryOnce   sync.Once
)

// GetGlobalLoggerFactory returns the global logger factory instance
func GetGlobalLoggerFactory() LoggerFactory {
	factoryOnce.Do(func() {
		globalFactory = NewLoggerFactory()
	})
	return globalFactory
}

// SetGlobalLoggerFactory sets the global logger factory (useful for dependency injection)
func SetGlobalLoggerFactory(factory LoggerFactory) {
	globalFactory = factory
}
