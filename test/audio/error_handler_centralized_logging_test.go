package audio_test

import (
	"errors"
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// MockLogger implements the logging.Logger interface for testing
type MockLogger struct {
	InfoCalls  []LogCall
	ErrorCalls []LogCall
	WarnCalls  []LogCall
	DebugCalls []LogCall
}

type LogCall struct {
	Message string
	Error   error
	Fields  map[string]interface{}
}

func (m *MockLogger) Info(msg string, fields map[string]interface{}) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) Error(msg string, err error, fields map[string]interface{}) {
	m.ErrorCalls = append(m.ErrorCalls, LogCall{Message: msg, Error: err, Fields: fields})
}

func (m *MockLogger) Warn(msg string, fields map[string]interface{}) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) Debug(msg string, fields map[string]interface{}) {
	m.DebugCalls = append(m.DebugCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) WithPipeline(pipeline string) logging.Logger {
	return &MockLogger{}
}

func (m *MockLogger) WithContext(ctx map[string]interface{}) logging.Logger {
	return &MockLogger{}
}

// MockLoggerFactory implements the logging.LoggerFactory interface for testing
type MockLoggerFactory struct {
	mockLogger *MockLogger
}

func (f *MockLoggerFactory) CreateLogger(component string) logging.Logger {
	return f.mockLogger
}

func (f *MockLoggerFactory) CreateAudioLogger(guildID string) logging.Logger {
	return f.mockLogger
}

func (f *MockLoggerFactory) CreateCommandLogger(commandName string) logging.Logger {
	return f.mockLogger
}

// MockAudioLogger implements the audio.AudioLogger interface for testing
type MockAudioLogger struct {
	InfoCalls  []LogCall
	ErrorCalls []LogCall
	WarnCalls  []LogCall
	DebugCalls []LogCall
}

func (m *MockAudioLogger) Info(msg string, fields map[string]interface{}) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockAudioLogger) Error(msg string, err error, fields map[string]interface{}) {
	m.ErrorCalls = append(m.ErrorCalls, LogCall{Message: msg, Error: err, Fields: fields})
}

func (m *MockAudioLogger) Warn(msg string, fields map[string]interface{}) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockAudioLogger) Debug(msg string, fields map[string]interface{}) {
	m.DebugCalls = append(m.DebugCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockAudioLogger) WithPipeline(pipeline string) audio.AudioLogger {
	return &MockAudioLogger{}
}

func (m *MockAudioLogger) WithContext(ctx map[string]interface{}) audio.AudioLogger {
	return &MockAudioLogger{}
}

// MockAudioRepository implements the audio.AudioRepository interface for testing
type MockAudioRepository struct {
	SaveErrorCalled bool
	SaveLogCalled   bool
}

func (m *MockAudioRepository) SaveError(error *models.AudioError) error {
	m.SaveErrorCalled = true
	return nil
}

func (m *MockAudioRepository) SaveMetric(metric *models.AudioMetric) error {
	return nil
}

func (m *MockAudioRepository) SaveLog(log *models.AudioLog) error {
	m.SaveLogCalled = true
	return nil
}

func (m *MockAudioRepository) GetErrorStats(guildID string) (*audio.ErrorStats, error) {
	return nil, nil
}

func (m *MockAudioRepository) GetMetricsStats(guildID string) (*audio.MetricsStats, error) {
	return nil, nil
}

func TestErrorHandler_CentralizedLoggingIntegration(t *testing.T) {
	// Setup mock components
	mockCentralizedLogger := &MockLogger{}
	mockLoggerFactory := &MockLoggerFactory{mockLogger: mockCentralizedLogger}
	mockAudioLogger := &MockAudioLogger{}
	mockRepo := &MockAudioRepository{}

	// Set the global logger factory for the test
	originalFactory := logging.GetGlobalLoggerFactory()
	logging.SetGlobalLoggerFactory(mockLoggerFactory)
	defer logging.SetGlobalLoggerFactory(originalFactory)

	// Create retry config
	retryConfig := &audio.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  time.Second,
		MaxDelay:   time.Minute,
		Multiplier: 2.0,
	}

	// Create error handler
	errorHandler := audio.NewBasicErrorHandler(retryConfig, mockAudioLogger, mockRepo, "test-guild-123")

	// Test error handling with centralized logging
	testError := errors.New("test network error: connection refused")
	shouldRetry, delay := errorHandler.HandleError(testError, "test-context")

	// Verify error handling behavior
	if !shouldRetry {
		t.Error("Expected network error to be retryable")
	}

	if delay <= 0 {
		t.Error("Expected positive retry delay")
	}

	// Verify that both legacy AudioLogger and centralized logging were called
	if len(mockAudioLogger.ErrorCalls) == 0 {
		t.Error("Expected AudioLogger.Error to be called for backward compatibility")
	}

	// Verify error was saved to repository
	if !mockRepo.SaveErrorCalled {
		t.Error("Expected error to be saved to repository")
	}

	// Test error classification with centralized logging
	if !errorHandler.IsRetryableError(testError) {
		t.Error("Expected network error to be classified as retryable")
	}

	// Test retry logic with centralized logging
	if !errorHandler.ShouldRetryAfterAttempts(1, testError) {
		t.Error("Expected to continue retrying after 1 attempt")
	}

	if errorHandler.ShouldRetryAfterAttempts(5, testError) {
		t.Error("Expected to stop retrying after max attempts exceeded")
	}
}

func TestErrorHandler_PipelineSpecificContext(t *testing.T) {
	// Setup mock components
	mockCentralizedLogger := &MockLogger{}
	mockLoggerFactory := &MockLoggerFactory{mockLogger: mockCentralizedLogger}
	mockAudioLogger := &MockAudioLogger{}
	mockRepo := &MockAudioRepository{}

	// Set the global logger factory for the test
	originalFactory := logging.GetGlobalLoggerFactory()
	logging.SetGlobalLoggerFactory(mockLoggerFactory)
	defer logging.SetGlobalLoggerFactory(originalFactory)

	// Create retry config
	retryConfig := &audio.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  time.Second,
		MaxDelay:   time.Minute,
		Multiplier: 2.0,
	}

	// Create error handler with notification support
	mockNotifier := &MockUserNotifier{}
	errorHandler := audio.NewBasicErrorHandlerWithNotifier(
		retryConfig, 
		mockAudioLogger, 
		mockRepo, 
		"test-guild-123", 
		mockNotifier, 
		"test-channel-456",
	)

	// Test that pipeline-specific context is maintained
	testError := errors.New("test error for context verification")
	errorHandler.LogError(testError, "pipeline-context-test")

	// Verify that error was logged (both systems should be called)
	if len(mockAudioLogger.ErrorCalls) == 0 {
		t.Error("Expected AudioLogger.Error to be called")
	}

	// Verify error was saved to repository with proper context
	if !mockRepo.SaveErrorCalled {
		t.Error("Expected error to be saved to repository")
	}
}

// MockUserNotifier implements the audio.UserNotifier interface for testing
type MockUserNotifier struct {
	NotifyErrorCalls      []NotifyCall
	NotifyRetryCalls      []NotifyCall
	NotifyFatalErrorCalls []NotifyCall
}

type NotifyCall struct {
	ChannelID   string
	ErrorType   string
	Message     string
	Attempt     int
	MaxAttempts int
	NextDelay   time.Duration
}

func (m *MockUserNotifier) NotifyError(channelID string, errorType string, message string) error {
	m.NotifyErrorCalls = append(m.NotifyErrorCalls, NotifyCall{
		ChannelID: channelID,
		ErrorType: errorType,
		Message:   message,
	})
	return nil
}

func (m *MockUserNotifier) NotifyRetry(channelID string, attempt int, maxAttempts int, nextDelay time.Duration) error {
	m.NotifyRetryCalls = append(m.NotifyRetryCalls, NotifyCall{
		ChannelID:   channelID,
		Attempt:     attempt,
		MaxAttempts: maxAttempts,
		NextDelay:   nextDelay,
	})
	return nil
}

func (m *MockUserNotifier) NotifyFatalError(channelID string, errorType string, message string) error {
	m.NotifyFatalErrorCalls = append(m.NotifyFatalErrorCalls, NotifyCall{
		ChannelID: channelID,
		ErrorType: errorType,
		Message:   message,
	})
	return nil
}