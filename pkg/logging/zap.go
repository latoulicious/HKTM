package logging

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger implements the Logger interface using zap
type ZapLogger struct {
	logger    *zap.Logger
	fields    map[string]interface{}
	component string
}

// NewZapLogger creates a new zap-based logger
func NewZapLogger(component string) (*ZapLogger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"

	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create zap logger: %w", err)
	}

	return &ZapLogger{
		logger:    logger,
		fields:    make(map[string]interface{}),
		component: component,
	}, nil
}

// Info logs informational messages
func (z *ZapLogger) Info(msg string, fields map[string]interface{}) {
	zapFields := z.buildZapFields(fields)
	z.logger.Info(msg, zapFields...)
}

// Error logs error messages with optional error object
func (z *ZapLogger) Error(msg string, err error, fields map[string]interface{}) {
	zapFields := z.buildZapFields(fields)
	if err != nil {
		zapFields = append(zapFields, zap.Error(err))
	}
	z.logger.Error(msg, zapFields...)
}

// Warn logs warning messages
func (z *ZapLogger) Warn(msg string, fields map[string]interface{}) {
	zapFields := z.buildZapFields(fields)
	z.logger.Warn(msg, zapFields...)
}

// Debug logs debug messages
func (z *ZapLogger) Debug(msg string, fields map[string]interface{}) {
	zapFields := z.buildZapFields(fields)
	z.logger.Debug(msg, zapFields...)
}

// WithPipeline creates a new logger with pipeline context
func (z *ZapLogger) WithPipeline(pipeline string) Logger {
	newFields := make(map[string]interface{})
	for k, v := range z.fields {
		newFields[k] = v
	}
	newFields["pipeline"] = pipeline

	return &ZapLogger{
		logger:    z.logger,
		fields:    newFields,
		component: z.component,
	}
}

// WithContext creates a new logger with additional context fields
func (z *ZapLogger) WithContext(ctx map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range z.fields {
		newFields[k] = v
	}
	for k, v := range ctx {
		newFields[k] = v
	}

	return &ZapLogger{
		logger:    z.logger,
		fields:    newFields,
		component: z.component,
	}
}

// buildZapFields converts map[string]interface{} to zap fields
func (z *ZapLogger) buildZapFields(fields map[string]interface{}) []zap.Field {
	zapFields := make([]zap.Field, 0, len(z.fields)+len(fields)+2)

	// Add component field
	zapFields = append(zapFields, zap.String("component", z.component))

	// Add timestamp
	zapFields = append(zapFields, zap.Time("timestamp", time.Now()))

	// Add persistent fields
	for k, v := range z.fields {
		zapFields = append(zapFields, z.convertToZapField(k, v))
	}

	// Add provided fields
	for k, v := range fields {
		zapFields = append(zapFields, z.convertToZapField(k, v))
	}

	return zapFields
}

// convertToZapField converts interface{} to appropriate zap field
func (z *ZapLogger) convertToZapField(key string, value interface{}) zap.Field {
	switch v := value.(type) {
	case string:
		return zap.String(key, v)
	case int:
		return zap.Int(key, v)
	case int64:
		return zap.Int64(key, v)
	case float64:
		return zap.Float64(key, v)
	case bool:
		return zap.Bool(key, v)
	case time.Duration:
		return zap.Duration(key, v)
	case time.Time:
		return zap.Time(key, v)
	case error:
		return zap.Error(v)
	default:
		return zap.Any(key, v)
	}
}

// Close closes the logger and flushes any buffered log entries
func (z *ZapLogger) Close() error {
	return z.logger.Sync()
}
