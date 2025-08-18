package logging

// ZapLoggerFactory implements LoggerFactory using zap
type ZapLoggerFactory struct{}

// NewZapLoggerFactory creates a new ZapLoggerFactory
func NewZapLoggerFactory() LoggerFactory {
	return &ZapLoggerFactory{}
}

// CreateLogger creates a generic logger for a component
func (f *ZapLoggerFactory) CreateLogger(component string) Logger {
	return NewZapLogger(component)
}

// CreateAudioLogger creates a logger for audio operations
func (f *ZapLoggerFactory) CreateAudioLogger(guildID string) Logger {
	logger := NewZapLogger("audio")
	return logger.WithContext(map[string]interface{}{
		"guild_id": guildID,
	})
}

// CreateCommandLogger creates a logger for command operations
func (f *ZapLoggerFactory) CreateCommandLogger(commandName string) Logger {
	logger := NewZapLogger("commands")
	return logger.WithContext(map[string]interface{}{
		"command": commandName,
	})
}

// CreateQueueLogger creates a logger for queue operations
func (f *ZapLoggerFactory) CreateQueueLogger(guildID string) Logger {
	logger := NewZapLogger("queue")
	return logger.WithContext(map[string]interface{}{
		"guild_id": guildID,
	})
}

// Global logger factory instance
var globalLoggerFactory LoggerFactory

func init() {
	globalLoggerFactory = NewZapLoggerFactory()
}

// GetGlobalLoggerFactory returns the global LoggerFactory instance
func GetGlobalLoggerFactory() LoggerFactory {
	return globalLoggerFactory
}

// DatabaseLoggerFactory implements LoggerFactory with database persistence
type DatabaseLoggerFactory struct {
	repository LogRepository
}

// NewDatabaseLoggerFactory creates a new DatabaseLoggerFactory
func NewDatabaseLoggerFactory(repository LogRepository) LoggerFactory {
	return &DatabaseLoggerFactory{
		repository: repository,
	}
}

// CreateLogger creates a generic logger for a component
func (f *DatabaseLoggerFactory) CreateLogger(component string) Logger {
	return NewDatabaseLogger(component, f.repository)
}

// CreateAudioLogger creates a logger for audio operations
func (f *DatabaseLoggerFactory) CreateAudioLogger(guildID string) Logger {
	logger := NewDatabaseLogger("audio", f.repository)
	return logger.WithContext(map[string]interface{}{
		"guild_id": guildID,
	})
}

// CreateCommandLogger creates a logger for command operations
func (f *DatabaseLoggerFactory) CreateCommandLogger(commandName string) Logger {
	logger := NewDatabaseLogger("commands", f.repository)
	return logger.WithContext(map[string]interface{}{
		"command": commandName,
	})
}

// CreateQueueLogger creates a logger for queue operations
func (f *DatabaseLoggerFactory) CreateQueueLogger(guildID string) Logger {
	logger := NewDatabaseLogger("queue", f.repository)
	return logger.WithContext(map[string]interface{}{
		"guild_id": guildID,
	})
}