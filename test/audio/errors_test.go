package audio_test

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/pkg/database/models"
)

// MockAudioLogger implements the AudioLogger interface for testing
type MockAudioLogger struct {
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

func (m *MockAudioLogger) Info(msg string, fields map[string]interface{}) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockAudioLogger) Error(msg string, err error, fields map[string]interface{}) {
	m.ErrorCalls = append(m.ErrorCalls, ErrorCall{Message: msg, Error: err, Fields: fields})
}

func (m *MockAudioLogger) Warn(msg string, fields map[string]interface{}) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockAudioLogger) Debug(msg string, fields map[string]interface{}) {
	m.DebugCalls = append(m.DebugCalls, LogCall{Message: msg, Fields: fields})
}

// MockErrorRepository implements the AudioRepository interface for testing errors
type MockErrorRepository struct {
	SavedErrors     []*models.AudioError
	SavedMetrics    []*models.AudioMetric
	SavedLogs       []*models.AudioLog
	SaveErrorResult error // Error to return from Save methods
}

func (m *MockErrorRepository) SaveError(error *models.AudioError) error {
	if m.SaveErrorResult != nil {
		return m.SaveErrorResult
	}
	m.SavedErrors = append(m.SavedErrors, error)
	return nil
}

func (m *MockErrorRepository) SaveMetric(metric *models.AudioMetric) error {
	if m.SaveErrorResult != nil {
		return m.SaveErrorResult
	}
	m.SavedMetrics = append(m.SavedMetrics, metric)
	return nil
}

func (m *MockErrorRepository) SaveLog(log *models.AudioLog) error {
	if m.SaveErrorResult != nil {
		return m.SaveErrorResult
	}
	m.SavedLogs = append(m.SavedLogs, log)
	return nil
}

func (m *MockErrorRepository) GetErrorStats(guildID string) (*audio.ErrorStats, error) {
	return nil, nil
}

func (m *MockErrorRepository) GetMetricsStats(guildID string) (*audio.MetricsStats, error) {
	return nil, nil
}

// Test helper to create a basic retry config
func createTestRetryConfig() *audio.RetryConfig {
	return &audio.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
	}
}

func TestBasicErrorHandler_IsRetryableError(t *testing.T) {
	config := createTestRetryConfig()
	logger := &MockAudioLogger{}
	repo := &MockErrorRepository{}
	handler := audio.NewBasicErrorHandler(config, logger, repo, "test-guild")

	tests := []struct {
		name     string
		error    error
		expected bool
	}{
		{
			name:     "nil error",
			error:    nil,
			expected: false,
		},
		{
			name:     "network timeout error",
			error:    &net.OpError{Op: "dial", Err: errors.New("timeout")},
			expected: true,
		},
		{
			name:     "connection refused error",
			error:    errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "yt-dlp rate limit error",
			error:    errors.New("HTTP Error 429: Too Many Requests"),
			expected: true,
		},
		{
			name:     "ffmpeg connection reset error",
			error:    errors.New("ffmpeg: connection reset by peer"),
			expected: true,
		},
		{
			name:     "discord websocket error",
			error:    errors.New("websocket: close 1006 (abnormal closure)"),
			expected: true,
		},
		{
			name:     "process exit error",
			error:    &exec.ExitError{},
			expected: true,
		},
		{
			name:     "syscall EAGAIN error",
			error:    syscall.EAGAIN,
			expected: true,
		},
		{
			name:     "invalid URL error (non-retryable)",
			error:    errors.New("invalid YouTube URL"),
			expected: false,
		},
		{
			name:     "configuration error (non-retryable)",
			error:    errors.New("invalid config: missing ffmpeg binary"),
			expected: false,
		},
		{
			name:     "unknown error (non-retryable)",
			error:    errors.New("some unknown error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsRetryableError(tt.error)
			if result != tt.expected {
				t.Errorf("IsRetryableError() = %v, expected %v for error: %v", result, tt.expected, tt.error)
			}
		})
	}
}

func TestBasicErrorHandler_HandleError(t *testing.T) {
	config := createTestRetryConfig()
	logger := &MockAudioLogger{}
	repo := &MockErrorRepository{}
	handler := audio.NewBasicErrorHandler(config, logger, repo, "test-guild")

	tests := []struct {
		name          string
		error         error
		context       string
		expectedRetry bool
		expectedDelay time.Duration
		minDelay      time.Duration
		maxDelay      time.Duration
	}{
		{
			name:          "retryable network error",
			error:         errors.New("connection timeout"),
			context:       "ffmpeg_process",
			expectedRetry: true,
			minDelay:      1 * time.Second,
			maxDelay:      5 * time.Second,
		},
		{
			name:          "non-retryable configuration error",
			error:         errors.New("invalid config"),
			context:       "initialization",
			expectedRetry: false,
			expectedDelay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRetry, delay := handler.HandleError(tt.error, tt.context)

			if shouldRetry != tt.expectedRetry {
				t.Errorf("HandleError() shouldRetry = %v, expected %v", shouldRetry, tt.expectedRetry)
			}

			if tt.expectedRetry {
				if delay < tt.minDelay || delay > tt.maxDelay {
					t.Errorf("HandleError() delay = %v, expected between %v and %v", delay, tt.minDelay, tt.maxDelay)
				}
			} else {
				if delay != tt.expectedDelay {
					t.Errorf("HandleError() delay = %v, expected %v", delay, tt.expectedDelay)
				}
			}

			// Verify error was logged
			if len(logger.ErrorCalls) == 0 {
				t.Error("Expected error to be logged, but no error calls were made")
			}

			// Verify error was saved to repository
			if len(repo.SavedErrors) == 0 {
				t.Error("Expected error to be saved to repository, but no errors were saved")
			}
		})
	}
}

func TestBasicErrorHandler_ExponentialBackoff(t *testing.T) {
	config := createTestRetryConfig()
	logger := &MockAudioLogger{}
	repo := &MockErrorRepository{}
	handler := audio.NewBasicErrorHandler(config, logger, repo, "test-guild")

	tests := []struct {
		attempt       int
		expectedDelay time.Duration
	}{
		{attempt: 1, expectedDelay: 2 * time.Second},  // base delay
		{attempt: 2, expectedDelay: 4 * time.Second},  // base * 2^1
		{attempt: 3, expectedDelay: 8 * time.Second},  // base * 2^2
		{attempt: 4, expectedDelay: 16 * time.Second}, // base * 2^3
		{attempt: 5, expectedDelay: 30 * time.Second}, // capped at max delay
		{attempt: 6, expectedDelay: 30 * time.Second}, // still capped at max delay
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := handler.GetRetryDelay(tt.attempt)
			if delay != tt.expectedDelay {
				t.Errorf("GetRetryDelay(%d) = %v, expected %v", tt.attempt, delay, tt.expectedDelay)
			}
		})
	}
}

func TestBasicErrorHandler_ShouldRetryAfterAttempts(t *testing.T) {
	config := createTestRetryConfig()
	logger := &MockAudioLogger{}
	repo := &MockErrorRepository{}
	handler := audio.NewBasicErrorHandler(config, logger, repo, "test-guild")

	retryableError := errors.New("connection timeout")
	nonRetryableError := errors.New("invalid config")

	tests := []struct {
		name     string
		attempts int
		error    error
		expected bool
	}{
		{
			name:     "retryable error, attempts below max",
			attempts: 2,
			error:    retryableError,
			expected: true,
		},
		{
			name:     "retryable error, attempts at max",
			attempts: 3,
			error:    retryableError,
			expected: false,
		},
		{
			name:     "retryable error, attempts above max",
			attempts: 4,
			error:    retryableError,
			expected: false,
		},
		{
			name:     "non-retryable error, attempts below max",
			attempts: 1,
			error:    nonRetryableError,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.ShouldRetryAfterAttempts(tt.attempts, tt.error)
			if result != tt.expected {
				t.Errorf("ShouldRetryAfterAttempts(%d, %v) = %v, expected %v", tt.attempts, tt.error, result, tt.expected)
			}
		})
	}
}

func TestBasicErrorHandler_LogError(t *testing.T) {
	config := createTestRetryConfig()
	logger := &MockAudioLogger{}
	repo := &MockErrorRepository{}
	handler := audio.NewBasicErrorHandler(config, logger, repo, "test-guild")

	testError := errors.New("test error")
	context := "test_context"

	handler.LogError(testError, context)

	// Verify logger was called
	if len(logger.ErrorCalls) != 1 {
		t.Errorf("Expected 1 error log call, got %d", len(logger.ErrorCalls))
	}

	errorCall := logger.ErrorCalls[0]
	if errorCall.Error != testError {
		t.Errorf("Expected logged error to be %v, got %v", testError, errorCall.Error)
	}

	// Verify repository was called
	if len(repo.SavedErrors) != 1 {
		t.Errorf("Expected 1 saved error, got %d", len(repo.SavedErrors))
	}

	savedError := repo.SavedErrors[0]
	if savedError.ErrorMsg != testError.Error() {
		t.Errorf("Expected saved error message to be %s, got %s", testError.Error(), savedError.ErrorMsg)
	}
	if savedError.Context != context {
		t.Errorf("Expected saved error context to be %s, got %s", context, savedError.Context)
	}
	if savedError.GuildID != "test-guild" {
		t.Errorf("Expected saved error guild ID to be test-guild, got %s", savedError.GuildID)
	}
}

func TestBasicErrorHandler_LogError_RepositoryFailure(t *testing.T) {
	config := createTestRetryConfig()
	logger := &MockAudioLogger{}
	repo := &MockErrorRepository{
		SaveErrorResult: errors.New("database connection failed"),
	}
	handler := audio.NewBasicErrorHandler(config, logger, repo, "test-guild")

	testError := errors.New("test error")
	context := "test_context"

	handler.LogError(testError, context)

	// Verify logger was called for the original error
	if len(logger.ErrorCalls) != 1 {
		t.Errorf("Expected 1 error log call, got %d", len(logger.ErrorCalls))
	}

	// Verify warning was logged for repository failure
	if len(logger.WarnCalls) != 1 {
		t.Errorf("Expected 1 warning log call for repository failure, got %d", len(logger.WarnCalls))
	}

	warnCall := logger.WarnCalls[0]
	if warnCall.Message != "Failed to save error to database" {
		t.Errorf("Expected warning message about database failure, got %s", warnCall.Message)
	}
}
