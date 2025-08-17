package logging

import (
	"fmt"
)

// PipelineLogger wraps a base logger with pipeline-specific context
type PipelineLogger struct {
	base     Logger
	pipeline string
	context  map[string]interface{}
}

// NewPipelineLogger creates a new pipeline-specific logger
func NewPipelineLogger(base Logger, pipeline string) *PipelineLogger {
	return &PipelineLogger{
		base:     base,
		pipeline: pipeline,
		context:  make(map[string]interface{}),
	}
}

// Info logs informational messages with pipeline context
func (p *PipelineLogger) Info(msg string, fields map[string]interface{}) {
	enrichedFields := p.enrichFields(fields)
	p.base.Info(fmt.Sprintf("[%s] %s", p.pipeline, msg), enrichedFields)
}

// Error logs error messages with pipeline context
func (p *PipelineLogger) Error(msg string, err error, fields map[string]interface{}) {
	enrichedFields := p.enrichFields(fields)
	p.base.Error(fmt.Sprintf("[%s] %s", p.pipeline, msg), err, enrichedFields)
}

// Warn logs warning messages with pipeline context
func (p *PipelineLogger) Warn(msg string, fields map[string]interface{}) {
	enrichedFields := p.enrichFields(fields)
	p.base.Warn(fmt.Sprintf("[%s] %s", p.pipeline, msg), enrichedFields)
}

// Debug logs debug messages with pipeline context
func (p *PipelineLogger) Debug(msg string, fields map[string]interface{}) {
	enrichedFields := p.enrichFields(fields)
	p.base.Debug(fmt.Sprintf("[%s] %s", p.pipeline, msg), enrichedFields)
}

// WithPipeline creates a new logger with updated pipeline context
func (p *PipelineLogger) WithPipeline(pipeline string) Logger {
	return &PipelineLogger{
		base:     p.base,
		pipeline: pipeline,
		context:  p.copyContext(),
	}
}

// WithContext creates a new logger with additional context fields
func (p *PipelineLogger) WithContext(ctx map[string]interface{}) Logger {
	newContext := p.copyContext()
	for k, v := range ctx {
		newContext[k] = v
	}

	return &PipelineLogger{
		base:     p.base,
		pipeline: p.pipeline,
		context:  newContext,
	}
}

// enrichFields combines pipeline context with provided fields
func (p *PipelineLogger) enrichFields(fields map[string]interface{}) map[string]interface{} {
	enriched := make(map[string]interface{})

	// Add pipeline context
	for k, v := range p.context {
		enriched[k] = v
	}

	// Add provided fields (these can override context)
	for k, v := range fields {
		enriched[k] = v
	}

	// Always add pipeline identifier
	enriched["pipeline"] = p.pipeline

	return enriched
}

// copyContext creates a copy of the current context
func (p *PipelineLogger) copyContext() map[string]interface{} {
	newContext := make(map[string]interface{})
	for k, v := range p.context {
		newContext[k] = v
	}
	return newContext
}

// AudioPipelineLogger creates a logger specifically for audio pipeline operations
type AudioPipelineLogger struct {
	*PipelineLogger
	guildID string
}

// NewAudioPipelineLogger creates a new audio pipeline logger
func NewAudioPipelineLogger(base Logger, guildID string) *AudioPipelineLogger {
	pipelineLogger := NewPipelineLogger(base, "audio")

	// Add audio-specific context
	audioContext := map[string]interface{}{
		"guild_id": guildID,
	}

	return &AudioPipelineLogger{
		PipelineLogger: pipelineLogger.WithContext(audioContext).(*PipelineLogger),
		guildID:        guildID,
	}
}

// WithURL adds URL context to the audio logger
func (a *AudioPipelineLogger) WithURL(url string) Logger {
	return a.WithContext(map[string]interface{}{
		"url": url,
	})
}

// WithUser adds user context to the audio logger
func (a *AudioPipelineLogger) WithUser(userID string) Logger {
	return a.WithContext(map[string]interface{}{
		"user_id": userID,
	})
}

// WithChannel adds channel context to the audio logger
func (a *AudioPipelineLogger) WithChannel(channelID string) Logger {
	return a.WithContext(map[string]interface{}{
		"channel_id": channelID,
	})
}

// CommandLogger creates a logger specifically for Discord command operations
type CommandLogger struct {
	*PipelineLogger
	commandName string
}

// NewCommandLogger creates a new command logger
func NewCommandLogger(base Logger, commandName string) *CommandLogger {
	pipelineLogger := NewPipelineLogger(base, "commands")

	// Add command-specific context
	commandContext := map[string]interface{}{
		"command": commandName,
	}

	return &CommandLogger{
		PipelineLogger: pipelineLogger.WithContext(commandContext).(*PipelineLogger),
		commandName:    commandName,
	}
}

// WithInteraction adds Discord interaction context to the command logger
func (c *CommandLogger) WithInteraction(guildID, userID, channelID string) Logger {
	return c.WithContext(map[string]interface{}{
		"guild_id":   guildID,
		"user_id":    userID,
		"channel_id": channelID,
	})
}
