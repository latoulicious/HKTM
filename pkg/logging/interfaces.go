package logging

// Logger defines the core logging interface used throughout the system
type Logger interface {
	// Info logs informational messages
	Info(msg string, fields map[string]interface{})

	// Error logs error messages with optional error object
	Error(msg string, err error, fields map[string]interface{})

	// Warn logs warning messages
	Warn(msg string, fields map[string]interface{})

	// Debug logs debug messages
	Debug(msg string, fields map[string]interface{})

	// WithPipeline creates a new logger with pipeline context
	WithPipeline(pipeline string) Logger

	// WithContext creates a new logger with additional context fields
	WithContext(ctx map[string]interface{}) Logger
}

// LoggerFactory creates different types of loggers for various components
type LoggerFactory interface {
	// CreateLogger creates a basic logger for the specified component
	CreateLogger(component string) Logger

	// CreateAudioLogger creates a logger specifically for audio pipeline operations
	CreateAudioLogger(guildID string) Logger

	// CreateCommandLogger creates a logger for Discord command operations
	CreateCommandLogger(commandName string) Logger
}
