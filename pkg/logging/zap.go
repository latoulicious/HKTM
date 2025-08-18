package logging

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger implements Logger interface using zap
type ZapLogger struct {
	logger    *zap.Logger
	component string
	context   map[string]interface{}
}

// NewZapLogger creates a new ZapLogger
func NewZapLogger(component string) *ZapLogger {
	// Create a production logger configuration
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	
	logger, err := config.Build()
	if err != nil {
		// Fallback to a basic logger if configuration fails
		logger = zap.NewNop()
	}

	return &ZapLogger{
		logger:    logger,
		component: component,
		context:   make(map[string]interface{}),
	}
}

// Info logs an info message
func (z *ZapLogger) Info(msg string, fields map[string]interface{}) {
	zapFields := z.buildZapFields(fields)
	z.logger.Info(fmt.Sprintf("[%s] %s", z.component, msg), zapFields...)
}

// Error logs an error message
func (z *ZapLogger) Error(msg string, err error, fields map[string]interface{}) {
	zapFields := z.buildZapFields(fields)
	if err != nil {
		zapFields = append(zapFields, zap.Error(err))
	}
	z.logger.Error(fmt.Sprintf("[%s] %s", z.component, msg), zapFields...)
}

// Warn logs a warning message
func (z *ZapLogger) Warn(msg string, fields map[string]interface{}) {
	zapFields := z.buildZapFields(fields)
	z.logger.Warn(fmt.Sprintf("[%s] %s", z.component, msg), zapFields...)
}

// Debug logs a debug message
func (z *ZapLogger) Debug(msg string, fields map[string]interface{}) {
	zapFields := z.buildZapFields(fields)
	z.logger.Debug(fmt.Sprintf("[%s] %s", z.component, msg), zapFields...)
}

// WithPipeline creates a new logger with pipeline context
func (z *ZapLogger) WithPipeline(pipeline string) Logger {
	newContext := make(map[string]interface{})
	for k, v := range z.context {
		newContext[k] = v
	}
	newContext["pipeline"] = pipeline

	return &ZapLogger{
		logger:    z.logger,
		component: z.component,
		context:   newContext,
	}
}

// WithContext creates a new logger with additional context
func (z *ZapLogger) WithContext(ctx map[string]interface{}) Logger {
	newContext := make(map[string]interface{})
	for k, v := range z.context {
		newContext[k] = v
	}
	for k, v := range ctx {
		newContext[k] = v
	}

	return &ZapLogger{
		logger:    z.logger,
		component: z.component,
		context:   newContext,
	}
}

// buildZapFields converts map fields to zap fields
func (z *ZapLogger) buildZapFields(fields map[string]interface{}) []zap.Field {
	var zapFields []zap.Field

	// Add context fields first
	for k, v := range z.context {
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