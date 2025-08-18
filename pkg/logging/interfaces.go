package logging

// Logger provides logging functionality with structured fields
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})
	WithPipeline(pipeline string) Logger
	WithContext(ctx map[string]interface{}) Logger
}

// LoggerFactory creates different types of loggers
type LoggerFactory interface {
	CreateLogger(component string) Logger
	CreateAudioLogger(guildID string) Logger
	CreateCommandLogger(commandName string) Logger
	CreateQueueLogger(guildID string) Logger
}

// LogRepository interface for persisting logs
type LogRepository interface {
	SaveLog(entry LogEntry) error
}

// LogEntry represents a log entry for persistence
type LogEntry struct {
	GuildID   string
	Component string
	Level     string
	Message   string
	Error     string
	Fields    map[string]interface{}
	UserID    string
	ChannelID string
}