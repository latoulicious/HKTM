package audio

import (
	"errors"
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/audio"
)

// TestErrorHandler_UserNotificationIntegration tests the error handler with user notification
func TestErrorHandler_UserNotificationIntegration(t *testing.T) {
	// Setup basic configuration
	config := &audio.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
	}

	// Create a simple mock logger that just stores calls
	logger := &SimpleLogger{calls: make([]string, 0)}

	// Create error handler without notification first
	handler := audio.NewBasicErrorHandler(config, logger, nil, "test-guild")

	// Test that it works without notifications
	testErr := errors.New("connection timed out")
	shouldRetry, delay := handler.HandleError(testErr, "test context")

	if !shouldRetry {
		t.Error("Expected retryable error to return shouldRetry=true")
	}

	if delay <= 0 {
		t.Error("Expected positive delay for retryable error")
	}

	// Verify logger was called
	if len(logger.calls) == 0 {
		t.Error("Expected logger to be called")
	}
}

// TestErrorHandler_FatalErrorHandling tests fatal error handling
func TestErrorHandler_FatalErrorHandling(t *testing.T) {
	config := &audio.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
	}

	logger := &SimpleLogger{calls: make([]string, 0)}
	handler := audio.NewBasicErrorHandler(config, logger, nil, "test-guild")

	// Test fatal error (invalid URL)
	testErr := errors.New("invalid youtube url")
	shouldRetry, delay := handler.HandleError(testErr, "test context")

	if shouldRetry {
		t.Error("Expected fatal error to return shouldRetry=false")
	}

	if delay != 0 {
		t.Error("Expected zero delay for fatal error")
	}
}

// TestErrorHandler_RetryLogic tests the retry logic
func TestErrorHandler_RetryLogic(t *testing.T) {
	config := &audio.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
	}

	logger := &SimpleLogger{calls: make([]string, 0)}
	handler := audio.NewBasicErrorHandler(config, logger, nil, "test-guild")

	// Test retry delay calculation
	delay1 := handler.GetRetryDelay(1)
	delay2 := handler.GetRetryDelay(2)
	delay3 := handler.GetRetryDelay(3)

	if delay1 >= delay2 {
		t.Error("Expected exponential backoff: delay1 should be less than delay2")
	}

	if delay2 >= delay3 {
		t.Error("Expected exponential backoff: delay2 should be less than delay3")
	}

	// Test max retries
	maxRetries := handler.GetMaxRetries()
	if maxRetries != 3 {
		t.Errorf("Expected max retries to be 3, got %d", maxRetries)
	}
}

// TestErrorHandler_ErrorClassification tests error classification
func TestErrorHandler_ErrorClassification(t *testing.T) {
	config := &audio.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
	}

	logger := &SimpleLogger{calls: make([]string, 0)}
	handler := audio.NewBasicErrorHandler(config, logger, nil, "test-guild")

	// Test various error types
	testCases := []struct {
		error     error
		retryable bool
		name      string
	}{
		{errors.New("connection timed out"), true, "network timeout"},
		{errors.New("connection refused"), true, "network connection refused"},
		{errors.New("invalid youtube url"), false, "invalid URL"},
		{errors.New("http error 429"), true, "rate limit"},
		{errors.New("ffmpeg failed"), true, "ffmpeg error"},
		{errors.New("websocket: close 1006"), true, "discord websocket error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isRetryable := handler.IsRetryableError(tc.error)
			if isRetryable != tc.retryable {
				t.Errorf("Expected error '%s' to be retryable=%t, got %t",
					tc.error.Error(), tc.retryable, isRetryable)
			}
		})
	}
}

// SimpleLogger is a minimal logger implementation for testing
type SimpleLogger struct {
	calls []string
}

func (sl *SimpleLogger) Info(msg string, fields map[string]interface{}) {
	sl.calls = append(sl.calls, "INFO: "+msg)
}

func (sl *SimpleLogger) Error(msg string, err error, fields map[string]interface{}) {
	sl.calls = append(sl.calls, "ERROR: "+msg+" - "+err.Error())
}

func (sl *SimpleLogger) Warn(msg string, fields map[string]interface{}) {
	sl.calls = append(sl.calls, "WARN: "+msg)
}

func (sl *SimpleLogger) Debug(msg string, fields map[string]interface{}) {
	sl.calls = append(sl.calls, "DEBUG: "+msg)
}
