package audio_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/pkg/database/models"
)

// MockAudioRepository implements AudioRepository for testing
type MockAudioRepository struct {
	savedLogs []models.AudioLog
}

func (m *MockAudioRepository) SaveError(error *models.AudioError) error {
	return nil
}

func (m *MockAudioRepository) SaveMetric(metric *models.AudioMetric) error {
	return nil
}

func (m *MockAudioRepository) SaveLog(log *models.AudioLog) error {
	m.savedLogs = append(m.savedLogs, *log)
	return nil
}

func (m *MockAudioRepository) GetErrorStats(guildID string) (*audio.ErrorStats, error) {
	return nil, nil
}

func (m *MockAudioRepository) GetMetricsStats(guildID string) (*audio.MetricsStats, error) {
	return nil, nil
}

func TestAudioLogger_Info(t *testing.T) {
	// Setup
	mockRepo := &MockAudioRepository{}
	config := &audio.LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: true,
	}
	guildID := uuid.New().String()

	logger := audio.NewAudioLogger(mockRepo, guildID, config)

	// Test Info logging
	fields := map[string]interface{}{
		"url":      "https://youtube.com/watch?v=test",
		"duration": 180,
	}

	logger.Info("Starting audio playback", fields)

	// Verify log was saved to database
	if len(mockRepo.savedLogs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(mockRepo.savedLogs))
	}

	savedLog := mockRepo.savedLogs[0]
	if savedLog.Level != "INFO" {
		t.Errorf("Expected log level INFO, got %s", savedLog.Level)
	}
	if savedLog.Message != "Starting audio playback" {
		t.Errorf("Expected message 'Starting audio playback', got %s", savedLog.Message)
	}
	if savedLog.GuildID != guildID {
		t.Errorf("Expected guild ID %s, got %s", guildID, savedLog.GuildID)
	}
	if savedLog.Fields["url"] != "https://youtube.com/watch?v=test" {
		t.Errorf("Expected URL field to be preserved")
	}
}

func TestAudioLogger_Error(t *testing.T) {
	// Setup
	mockRepo := &MockAudioRepository{}
	config := &audio.LoggerConfig{
		Level:    "error",
		Format:   "json",
		SaveToDB: true,
	}
	guildID := uuid.New().String()

	logger := audio.NewAudioLogger(mockRepo, guildID, config)

	// Test Error logging
	testError := errors.New("FFmpeg process crashed")
	fields := map[string]interface{}{
		"process_id": 12345,
		"exit_code":  1,
	}

	logger.Error("Audio processing failed", testError, fields)

	// Verify log was saved to database
	if len(mockRepo.savedLogs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(mockRepo.savedLogs))
	}

	savedLog := mockRepo.savedLogs[0]
	if savedLog.Level != "ERROR" {
		t.Errorf("Expected log level ERROR, got %s", savedLog.Level)
	}
	if savedLog.Error != "FFmpeg process crashed" {
		t.Errorf("Expected error message 'FFmpeg process crashed', got %s", savedLog.Error)
	}
	if savedLog.Fields["process_id"] != 12345 {
		t.Errorf("Expected process_id field to be preserved")
	}
}

func TestAudioLogger_DatabaseDisabled(t *testing.T) {
	// Setup with database saving disabled
	mockRepo := &MockAudioRepository{}
	config := &audio.LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: false, // Database saving disabled
	}
	guildID := uuid.New().String()

	logger := audio.NewAudioLogger(mockRepo, guildID, config)

	// Test logging
	logger.Info("Test message", nil)

	// Verify no logs were saved to database
	if len(mockRepo.savedLogs) != 0 {
		t.Errorf("Expected 0 log entries when SaveToDB is false, got %d", len(mockRepo.savedLogs))
	}
}

func TestAudioLogger_NilRepository(t *testing.T) {
	// Setup with nil repository
	config := &audio.LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: true,
	}
	guildID := uuid.New().String()

	// This should not panic even with nil repository
	logger := audio.NewAudioLogger(nil, guildID, config)

	// Test logging - should not panic
	logger.Info("Test message", nil)
	logger.Error("Test error", errors.New("test"), nil)
	logger.Warn("Test warning", nil)
	logger.Debug("Test debug", nil)
}

func TestAudioLogger_AllLogLevels(t *testing.T) {
	// Setup
	mockRepo := &MockAudioRepository{}
	config := &audio.LoggerConfig{
		Level:    "debug", // Enable all log levels
		Format:   "json",
		SaveToDB: true,
	}
	guildID := uuid.New().String()

	logger := audio.NewAudioLogger(mockRepo, guildID, config)

	// Test all log levels
	logger.Debug("Debug message", nil)
	logger.Info("Info message", nil)
	logger.Warn("Warn message", nil)
	logger.Error("Error message", errors.New("test error"), nil)

	// Verify all logs were saved
	if len(mockRepo.savedLogs) != 4 {
		t.Errorf("Expected 4 log entries, got %d", len(mockRepo.savedLogs))
	}

	// Verify log levels
	expectedLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for i, expectedLevel := range expectedLevels {
		if mockRepo.savedLogs[i].Level != expectedLevel {
			t.Errorf("Expected log level %s at index %d, got %s", expectedLevel, i, mockRepo.savedLogs[i].Level)
		}
	}
}
