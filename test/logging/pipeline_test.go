package logging_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/latoulicious/HKTM/pkg/logging"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	InfoCalls  []LogCall
	ErrorCalls []ErrorCall
	WarnCalls  []LogCall
	DebugCalls []LogCall
}

type LogCall struct {
	Message string
	Fields  map[string]interface{}
}

type ErrorCall struct {
	Message string
	Error   error
	Fields  map[string]interface{}
}

func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

func (m *MockLogger) Info(msg string, fields map[string]interface{}) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) Error(msg string, err error, fields map[string]interface{}) {
	m.ErrorCalls = append(m.ErrorCalls, ErrorCall{Message: msg, Error: err, Fields: fields})
}

func (m *MockLogger) Warn(msg string, fields map[string]interface{}) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) Debug(msg string, fields map[string]interface{}) {
	m.DebugCalls = append(m.DebugCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) WithPipeline(pipeline string) logging.Logger {
	// For testing, return a new mock that tracks the pipeline
	newMock := NewMockLogger()
	// Copy existing calls to maintain history
	newMock.InfoCalls = append(newMock.InfoCalls, m.InfoCalls...)
	newMock.ErrorCalls = append(newMock.ErrorCalls, m.ErrorCalls...)
	newMock.WarnCalls = append(newMock.WarnCalls, m.WarnCalls...)
	newMock.DebugCalls = append(newMock.DebugCalls, m.DebugCalls...)
	return newMock
}

func (m *MockLogger) WithContext(ctx map[string]interface{}) logging.Logger {
	// For testing, return a new mock that tracks the context
	newMock := NewMockLogger()
	// Copy existing calls to maintain history
	newMock.InfoCalls = append(newMock.InfoCalls, m.InfoCalls...)
	newMock.ErrorCalls = append(newMock.ErrorCalls, m.ErrorCalls...)
	newMock.WarnCalls = append(newMock.WarnCalls, m.WarnCalls...)
	newMock.DebugCalls = append(newMock.DebugCalls, m.DebugCalls...)
	return newMock
}

// Test PipelineLogger functionality

func TestPipelineLogger_BasicLogging(t *testing.T) {
	baseLogger := NewMockLogger()
	pipelineLogger := logging.NewPipelineLogger(baseLogger, "audio")
	
	// Test Info logging
	fields := map[string]interface{}{
		"url":      "https://youtube.com/watch?v=test",
		"duration": 180,
	}
	
	pipelineLogger.Info("Starting audio playback", fields)
	
	// Verify base logger was called
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	infoCall := baseLogger.InfoCalls[0]
	
	// Verify message includes pipeline prefix
	if !strings.Contains(infoCall.Message, "[audio]") {
		t.Errorf("Expected message to contain pipeline prefix, got: %s", infoCall.Message)
	}
	
	if !strings.Contains(infoCall.Message, "Starting audio playback") {
		t.Errorf("Expected message to contain original text, got: %s", infoCall.Message)
	}
	
	// Verify fields are enriched with pipeline info
	if infoCall.Fields["pipeline"] != "audio" {
		t.Errorf("Expected pipeline field to be 'audio', got: %v", infoCall.Fields["pipeline"])
	}
	
	if infoCall.Fields["url"] != "https://youtube.com/watch?v=test" {
		t.Errorf("Expected original fields to be preserved")
	}
}

func TestPipelineLogger_ErrorLogging(t *testing.T) {
	baseLogger := NewMockLogger()
	pipelineLogger := logging.NewPipelineLogger(baseLogger, "commands")
	
	testError := errors.New("command execution failed")
	fields := map[string]interface{}{
		"command": "play",
		"user_id": "user123",
	}
	
	pipelineLogger.Error("Command failed", testError, fields)
	
	// Verify base logger was called
	if len(baseLogger.ErrorCalls) != 1 {
		t.Errorf("Expected 1 error call, got %d", len(baseLogger.ErrorCalls))
	}
	
	errorCall := baseLogger.ErrorCalls[0]
	
	// Verify message includes pipeline prefix
	if !strings.Contains(errorCall.Message, "[commands]") {
		t.Errorf("Expected message to contain pipeline prefix, got: %s", errorCall.Message)
	}
	
	// Verify error is passed through
	if errorCall.Error != testError {
		t.Errorf("Expected error to be passed through, got: %v", errorCall.Error)
	}
	
	// Verify fields are enriched
	if errorCall.Fields["pipeline"] != "commands" {
		t.Errorf("Expected pipeline field to be 'commands', got: %v", errorCall.Fields["pipeline"])
	}
}

func TestPipelineLogger_WithContext(t *testing.T) {
	baseLogger := NewMockLogger()
	pipelineLogger := logging.NewPipelineLogger(baseLogger, "audio")
	
	// Add context to the logger
	contextFields := map[string]interface{}{
		"guild_id": "guild123",
		"url":      "https://youtube.com/test",
	}
	
	contextLogger := pipelineLogger.WithContext(contextFields)
	
	// Log with the context logger
	contextLogger.Info("Playback started", map[string]interface{}{
		"duration": 180,
	})
	
	// Verify base logger was called
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	infoCall := baseLogger.InfoCalls[0]
	
	// Verify context fields are included
	if infoCall.Fields["guild_id"] != "guild123" {
		t.Errorf("Expected guild_id from context, got: %v", infoCall.Fields["guild_id"])
	}
	
	if infoCall.Fields["url"] != "https://youtube.com/test" {
		t.Errorf("Expected url from context, got: %v", infoCall.Fields["url"])
	}
	
	// Verify new fields are also included
	if infoCall.Fields["duration"] != 180 {
		t.Errorf("Expected duration from new fields, got: %v", infoCall.Fields["duration"])
	}
	
	// Verify pipeline field is still present
	if infoCall.Fields["pipeline"] != "audio" {
		t.Errorf("Expected pipeline field to be preserved, got: %v", infoCall.Fields["pipeline"])
	}
}

func TestPipelineLogger_WithPipeline(t *testing.T) {
	baseLogger := NewMockLogger()
	pipelineLogger := logging.NewPipelineLogger(baseLogger, "audio")
	
	// Create a new logger with different pipeline
	newPipelineLogger := pipelineLogger.WithPipeline("database")
	
	// Log with the new pipeline logger
	newPipelineLogger.Info("Database operation", nil)
	
	// Verify base logger was called
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	infoCall := baseLogger.InfoCalls[0]
	
	// Verify message uses new pipeline prefix
	if !strings.Contains(infoCall.Message, "[database]") {
		t.Errorf("Expected message to contain new pipeline prefix, got: %s", infoCall.Message)
	}
	
	// Verify pipeline field is updated
	if infoCall.Fields["pipeline"] != "database" {
		t.Errorf("Expected pipeline field to be 'database', got: %v", infoCall.Fields["pipeline"])
	}
}

func TestPipelineLogger_FieldOverrides(t *testing.T) {
	baseLogger := NewMockLogger()
	pipelineLogger := logging.NewPipelineLogger(baseLogger, "audio")
	
	// Add context with a field
	contextLogger := pipelineLogger.WithContext(map[string]interface{}{
		"url":      "https://youtube.com/original",
		"guild_id": "guild123",
	})
	
	// Log with fields that override context
	contextLogger.Info("Test message", map[string]interface{}{
		"url":      "https://youtube.com/override", // This should override context
		"duration": 180,                            // This is new
	})
	
	// Verify base logger was called
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	infoCall := baseLogger.InfoCalls[0]
	
	// Verify override worked
	if infoCall.Fields["url"] != "https://youtube.com/override" {
		t.Errorf("Expected url to be overridden, got: %v", infoCall.Fields["url"])
	}
	
	// Verify context field that wasn't overridden is preserved
	if infoCall.Fields["guild_id"] != "guild123" {
		t.Errorf("Expected guild_id from context to be preserved, got: %v", infoCall.Fields["guild_id"])
	}
	
	// Verify new field is included
	if infoCall.Fields["duration"] != 180 {
		t.Errorf("Expected duration from new fields, got: %v", infoCall.Fields["duration"])
	}
}

func TestPipelineLogger_AllLogLevels(t *testing.T) {
	baseLogger := NewMockLogger()
	pipelineLogger := logging.NewPipelineLogger(baseLogger, "test")
	
	// Test all log levels
	pipelineLogger.Debug("Debug message", nil)
	pipelineLogger.Info("Info message", nil)
	pipelineLogger.Warn("Warn message", nil)
	pipelineLogger.Error("Error message", errors.New("test error"), nil)
	
	// Verify all calls were made
	if len(baseLogger.DebugCalls) != 1 {
		t.Errorf("Expected 1 debug call, got %d", len(baseLogger.DebugCalls))
	}
	
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	if len(baseLogger.WarnCalls) != 1 {
		t.Errorf("Expected 1 warn call, got %d", len(baseLogger.WarnCalls))
	}
	
	if len(baseLogger.ErrorCalls) != 1 {
		t.Errorf("Expected 1 error call, got %d", len(baseLogger.ErrorCalls))
	}
	
	// Verify all messages have pipeline prefix
	if !strings.Contains(baseLogger.DebugCalls[0].Message, "[test]") {
		t.Error("Debug message should have pipeline prefix")
	}
	
	if !strings.Contains(baseLogger.InfoCalls[0].Message, "[test]") {
		t.Error("Info message should have pipeline prefix")
	}
	
	if !strings.Contains(baseLogger.WarnCalls[0].Message, "[test]") {
		t.Error("Warn message should have pipeline prefix")
	}
	
	if !strings.Contains(baseLogger.ErrorCalls[0].Message, "[test]") {
		t.Error("Error message should have pipeline prefix")
	}
}

// Test AudioPipelineLogger functionality

func TestAudioPipelineLogger_Creation(t *testing.T) {
	baseLogger := NewMockLogger()
	audioLogger := logging.NewAudioPipelineLogger(baseLogger, "guild123")
	
	// Log a message
	audioLogger.Info("Audio pipeline started", nil)
	
	// Verify base logger was called
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	infoCall := baseLogger.InfoCalls[0]
	
	// Verify message has audio pipeline prefix
	if !strings.Contains(infoCall.Message, "[audio]") {
		t.Errorf("Expected message to contain audio pipeline prefix, got: %s", infoCall.Message)
	}
	
	// Verify guild_id is automatically included
	if infoCall.Fields["guild_id"] != "guild123" {
		t.Errorf("Expected guild_id to be automatically included, got: %v", infoCall.Fields["guild_id"])
	}
	
	// Verify pipeline field is set
	if infoCall.Fields["pipeline"] != "audio" {
		t.Errorf("Expected pipeline field to be 'audio', got: %v", infoCall.Fields["pipeline"])
	}
}

func TestAudioPipelineLogger_WithURL(t *testing.T) {
	baseLogger := NewMockLogger()
	audioLogger := logging.NewAudioPipelineLogger(baseLogger, "guild123")
	
	// Add URL context
	urlLogger := audioLogger.WithURL("https://youtube.com/watch?v=test")
	
	// Log a message
	urlLogger.Info("Starting playback", nil)
	
	// Verify base logger was called
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	infoCall := baseLogger.InfoCalls[0]
	
	// Verify URL is included
	if infoCall.Fields["url"] != "https://youtube.com/watch?v=test" {
		t.Errorf("Expected url to be included, got: %v", infoCall.Fields["url"])
	}
	
	// Verify guild_id is still included
	if infoCall.Fields["guild_id"] != "guild123" {
		t.Errorf("Expected guild_id to be preserved, got: %v", infoCall.Fields["guild_id"])
	}
}

func TestAudioPipelineLogger_WithUser(t *testing.T) {
	baseLogger := NewMockLogger()
	audioLogger := logging.NewAudioPipelineLogger(baseLogger, "guild123")
	
	// Add user context
	userLogger := audioLogger.WithUser("user456")
	
	// Log a message
	userLogger.Info("User initiated playback", nil)
	
	// Verify base logger was called
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	infoCall := baseLogger.InfoCalls[0]
	
	// Verify user_id is included
	if infoCall.Fields["user_id"] != "user456" {
		t.Errorf("Expected user_id to be included, got: %v", infoCall.Fields["user_id"])
	}
	
	// Verify guild_id is still included
	if infoCall.Fields["guild_id"] != "guild123" {
		t.Errorf("Expected guild_id to be preserved, got: %v", infoCall.Fields["guild_id"])
	}
}

func TestAudioPipelineLogger_WithChannel(t *testing.T) {
	baseLogger := NewMockLogger()
	audioLogger := logging.NewAudioPipelineLogger(baseLogger, "guild123")
	
	// Add channel context
	channelLogger := audioLogger.WithChannel("channel789")
	
	// Log a message
	channelLogger.Info("Playback in channel", nil)
	
	// Verify base logger was called
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	infoCall := baseLogger.InfoCalls[0]
	
	// Verify channel_id is included
	if infoCall.Fields["channel_id"] != "channel789" {
		t.Errorf("Expected channel_id to be included, got: %v", infoCall.Fields["channel_id"])
	}
	
	// Verify guild_id is still included
	if infoCall.Fields["guild_id"] != "guild123" {
		t.Errorf("Expected guild_id to be preserved, got: %v", infoCall.Fields["guild_id"])
	}
}

// Test CommandLogger functionality

func TestCommandLogger_Creation(t *testing.T) {
	baseLogger := NewMockLogger()
	commandLogger := logging.NewCommandLogger(baseLogger, "play")
	
	// Log a message
	commandLogger.Info("Command executed", nil)
	
	// Verify base logger was called
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	infoCall := baseLogger.InfoCalls[0]
	
	// Verify message has commands pipeline prefix
	if !strings.Contains(infoCall.Message, "[commands]") {
		t.Errorf("Expected message to contain commands pipeline prefix, got: %s", infoCall.Message)
	}
	
	// Verify command is automatically included
	if infoCall.Fields["command"] != "play" {
		t.Errorf("Expected command to be automatically included, got: %v", infoCall.Fields["command"])
	}
	
	// Verify pipeline field is set
	if infoCall.Fields["pipeline"] != "commands" {
		t.Errorf("Expected pipeline field to be 'commands', got: %v", infoCall.Fields["pipeline"])
	}
}

func TestCommandLogger_WithInteraction(t *testing.T) {
	baseLogger := NewMockLogger()
	commandLogger := logging.NewCommandLogger(baseLogger, "play")
	
	// Add interaction context
	interactionLogger := commandLogger.WithInteraction("guild123", "user456", "channel789")
	
	// Log a message
	interactionLogger.Info("Command executed with interaction", nil)
	
	// Verify base logger was called
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	infoCall := baseLogger.InfoCalls[0]
	
	// Verify all interaction fields are included
	if infoCall.Fields["guild_id"] != "guild123" {
		t.Errorf("Expected guild_id to be included, got: %v", infoCall.Fields["guild_id"])
	}
	
	if infoCall.Fields["user_id"] != "user456" {
		t.Errorf("Expected user_id to be included, got: %v", infoCall.Fields["user_id"])
	}
	
	if infoCall.Fields["channel_id"] != "channel789" {
		t.Errorf("Expected channel_id to be included, got: %v", infoCall.Fields["channel_id"])
	}
	
	// Verify command is still included
	if infoCall.Fields["command"] != "play" {
		t.Errorf("Expected command to be preserved, got: %v", infoCall.Fields["command"])
	}
}

func TestPipelineLogger_NilFields(t *testing.T) {
	baseLogger := NewMockLogger()
	pipelineLogger := logging.NewPipelineLogger(baseLogger, "test")
	
	// Log with nil fields (should not panic)
	pipelineLogger.Info("Test message", nil)
	pipelineLogger.Error("Test error", errors.New("test"), nil)
	pipelineLogger.Warn("Test warning", nil)
	pipelineLogger.Debug("Test debug", nil)
	
	// Verify all calls were made without panicking
	if len(baseLogger.InfoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(baseLogger.InfoCalls))
	}
	
	if len(baseLogger.ErrorCalls) != 1 {
		t.Errorf("Expected 1 error call, got %d", len(baseLogger.ErrorCalls))
	}
	
	if len(baseLogger.WarnCalls) != 1 {
		t.Errorf("Expected 1 warn call, got %d", len(baseLogger.WarnCalls))
	}
	
	if len(baseLogger.DebugCalls) != 1 {
		t.Errorf("Expected 1 debug call, got %d", len(baseLogger.DebugCalls))
	}
	
	// Verify pipeline field is still added even with nil input fields
	if baseLogger.InfoCalls[0].Fields["pipeline"] != "test" {
		t.Error("Pipeline field should be added even with nil input fields")
	}
}