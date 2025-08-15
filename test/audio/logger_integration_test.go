package audio

import (
	"errors"
	"testing"

	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAudioRepository is a mock implementation of AudioRepository for testing
type MockAudioRepository struct {
	mock.Mock
}

func (m *MockAudioRepository) SaveError(audioError *models.AudioError) error {
	args := m.Called(audioError)
	return args.Error(0)
}

func (m *MockAudioRepository) SaveMetric(metric *models.AudioMetric) error {
	args := m.Called(metric)
	return args.Error(0)
}

func (m *MockAudioRepository) SaveLog(log *models.AudioLog) error {
	args := m.Called(log)
	return args.Error(0)
}

func (m *MockAudioRepository) GetErrorStats(guildID string) (*audio.ErrorStats, error) {
	args := m.Called(guildID)
	return args.Get(0).(*audio.ErrorStats), args.Error(1)
}

func (m *MockAudioRepository) GetMetricsStats(guildID string) (*audio.MetricsStats, error) {
	args := m.Called(guildID)
	return args.Get(0).(*audio.MetricsStats), args.Error(1)
}

func TestAudioLogger_DatabaseIntegration_Success(t *testing.T) {
	// Create mock repository
	mockRepo := &MockAudioRepository{}

	// Configure mock to succeed
	mockRepo.On("SaveLog", mock.AnythingOfType("*models.AudioLog")).Return(nil)

	// Create logger config
	config := &audio.LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: true,
	}

	// Create logger
	logger := audio.NewAudioLogger(mockRepo, "test-guild-123", config)

	// Test Info logging
	logger.Info("Test info message", map[string]interface{}{
		"test_field": "test_value",
	})

	// Test Error logging
	testErr := errors.New("test error")
	logger.Error("Test error message", testErr, map[string]interface{}{
		"error_context": "test_context",
	})

	// Verify SaveLog was called twice
	mockRepo.AssertNumberOfCalls(t, "SaveLog", 2)

	// Verify the calls had correct parameters
	calls := mockRepo.Calls

	// Check first call (Info)
	infoLog := calls[0].Arguments[0].(*models.AudioLog)
	assert.Equal(t, "test-guild-123", infoLog.GuildID)
	assert.Equal(t, "INFO", infoLog.Level)
	assert.Equal(t, "Test info message", infoLog.Message)
	assert.Equal(t, "", infoLog.Error)
	assert.Equal(t, "test_value", infoLog.Fields["test_field"])

	// Check second call (Error)
	errorLog := calls[1].Arguments[0].(*models.AudioLog)
	assert.Equal(t, "test-guild-123", errorLog.GuildID)
	assert.Equal(t, "ERROR", errorLog.Level)
	assert.Equal(t, "Test error message", errorLog.Message)
	assert.Equal(t, "test error", errorLog.Error)
	assert.Equal(t, "test_context", errorLog.Fields["error_context"])
}

func TestAudioLogger_DatabaseIntegration_Failure(t *testing.T) {
	// Create mock repository
	mockRepo := &MockAudioRepository{}

	// Configure mock to fail
	dbError := errors.New("database connection failed")
	mockRepo.On("SaveLog", mock.AnythingOfType("*models.AudioLog")).Return(dbError)

	// Create logger config
	config := &audio.LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: true,
	}

	// Create logger
	logger := audio.NewAudioLogger(mockRepo, "test-guild-123", config)

	// Cast to implementation to access failure counter
	loggerImpl := logger.(*audio.AudioLoggerImpl)

	// Test multiple failed attempts
	for i := 0; i < 7; i++ {
		logger.Info("Test message", nil)
	}

	// Should have attempted 5 times, then stopped due to max failures
	mockRepo.AssertNumberOfCalls(t, "SaveLog", 5)

	// Check failure counter
	assert.Equal(t, 5, loggerImpl.GetDatabaseFailures())

	// Reset failures and try again
	loggerImpl.ResetDatabaseFailures()
	assert.Equal(t, 0, loggerImpl.GetDatabaseFailures())

	// Should attempt again after reset
	logger.Info("Test after reset", nil)
	mockRepo.AssertNumberOfCalls(t, "SaveLog", 6)
}

func TestAudioLogger_DatabaseIntegration_Disabled(t *testing.T) {
	// Create mock repository
	mockRepo := &MockAudioRepository{}

	// Create logger config with database saving disabled
	config := &audio.LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: false, // Disabled
	}

	// Create logger
	logger := audio.NewAudioLogger(mockRepo, "test-guild-123", config)

	// Test logging
	logger.Info("Test info message", map[string]interface{}{
		"test_field": "test_value",
	})

	logger.Error("Test error message", errors.New("test error"), nil)

	// Verify SaveLog was never called
	mockRepo.AssertNumberOfCalls(t, "SaveLog", 0)
}

func TestAudioLogger_DatabaseIntegration_NilRepository(t *testing.T) {
	// Create logger config
	config := &audio.LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: true,
	}

	// Create logger with nil repository
	logger := audio.NewAudioLogger(nil, "test-guild-123", config)

	// Test logging - should not panic
	logger.Info("Test info message", map[string]interface{}{
		"test_field": "test_value",
	})

	logger.Error("Test error message", errors.New("test error"), nil)

	// Should complete without errors
}
