package audio

import (
	"fmt"
	"math"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/latoulicious/HKTM/pkg/database/models"
)

// BasicErrorHandler implements the ErrorHandler interface with retry logic and centralized logging
type BasicErrorHandler struct {
	retryConfig *RetryConfig
	logger      AudioLogger
	repository  AudioRepository
	guildID     string
}

// NewBasicErrorHandler creates a new BasicErrorHandler instance
func NewBasicErrorHandler(config *RetryConfig, logger AudioLogger, repo AudioRepository, guildID string) ErrorHandler {
	return &BasicErrorHandler{
		retryConfig: config,
		logger:      logger,
		repository:  repo,
		guildID:     guildID,
	}
}

// HandleError processes an error and determines if it should be retried and with what delay
func (beh *BasicErrorHandler) HandleError(err error, context string) (shouldRetry bool, delay time.Duration) {
	// Log the error first
	beh.LogError(err, context)

	// Check if the error is retryable
	if !beh.IsRetryableError(err) {
		beh.logger.Info("Error is not retryable, skipping retry logic", map[string]interface{}{
			"error":   err.Error(),
			"context": context,
		})
		return false, 0
	}

	// For retryable errors, calculate exponential backoff delay
	// We don't track retry attempts here since that's handled by the caller
	// This method just determines if an error type is retryable and calculates delay
	delay = beh.calculateExponentialBackoff(1) // Start with attempt 1 for base calculation

	beh.logger.Info("Error is retryable, will attempt retry", map[string]interface{}{
		"error":       err.Error(),
		"context":     context,
		"retry_delay": delay.String(),
	})

	return true, delay
}

// LogError logs an error to both console and database
func (beh *BasicErrorHandler) LogError(err error, context string) {
	// Classify the error type for database storage
	errorType := beh.classifyErrorType(err)

	// Log to console via AudioLogger
	beh.logger.Error("Audio pipeline error occurred", err, map[string]interface{}{
		"context":    context,
		"error_type": errorType,
		"guild_id":   beh.guildID,
	})

	// Save to database if repository is available
	if beh.repository != nil {
		audioError := &models.AudioError{
			ID:        uuid.New(),
			GuildID:   beh.guildID,
			ErrorType: errorType,
			ErrorMsg:  err.Error(),
			Context:   context,
			Timestamp: time.Now(),
			Resolved:  false,
		}

		if saveErr := beh.repository.SaveError(audioError); saveErr != nil {
			beh.logger.Warn("Failed to save error to database", map[string]interface{}{
				"save_error":     saveErr.Error(),
				"original_error": err.Error(),
				"context":        context,
			})
		}
	}
}

// IsRetryableError determines if an error should be retried based on its type and characteristics
func (beh *BasicErrorHandler) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := strings.ToLower(err.Error())

	// Network-related errors (retryable)
	if isNetworkError(err) {
		return true
	}

	// Process-related errors (retryable)
	if isProcessError(err) {
		return true
	}

	// yt-dlp specific retryable errors
	if isYtDlpRetryableError(errorStr) {
		return true
	}

	// FFmpeg specific retryable errors
	if isFFmpegRetryableError(errorStr) {
		return true
	}

	// Discord API retryable errors
	if isDiscordRetryableError(errorStr) {
		return true
	}

	// Temporary file system errors (retryable)
	if isTemporaryFileSystemError(errorStr) {
		return true
	}

	// Default to non-retryable for unknown errors
	beh.logger.Debug("Error classified as non-retryable", map[string]interface{}{
		"error":                 err.Error(),
		"classification_reason": "no matching retryable pattern",
	})

	return false
}

// calculateExponentialBackoff calculates the delay for a given retry attempt
func (beh *BasicErrorHandler) calculateExponentialBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return beh.retryConfig.BaseDelay
	}

	// Calculate exponential backoff: baseDelay * (multiplier ^ (attempt - 1))
	multiplier := math.Pow(beh.retryConfig.Multiplier, float64(attempt-1))
	delay := time.Duration(float64(beh.retryConfig.BaseDelay) * multiplier)

	// Cap at maximum delay
	if delay > beh.retryConfig.MaxDelay {
		delay = beh.retryConfig.MaxDelay
	}

	return delay
}

// classifyErrorType returns a string classification of the error type for database storage
func (beh *BasicErrorHandler) classifyErrorType(err error) string {
	if err == nil {
		return "unknown"
	}

	errorStr := strings.ToLower(err.Error())

	// Network errors
	if isNetworkError(err) {
		return "network"
	}

	// Process errors
	if isProcessError(err) {
		return "process"
	}

	// yt-dlp errors
	if strings.Contains(errorStr, "yt-dlp") || strings.Contains(errorStr, "youtube-dl") {
		return "yt-dlp"
	}

	// FFmpeg errors
	if strings.Contains(errorStr, "ffmpeg") {
		return "ffmpeg"
	}

	// Discord API errors
	if strings.Contains(errorStr, "discord") || strings.Contains(errorStr, "websocket") {
		return "discord"
	}

	// File system errors
	if strings.Contains(errorStr, "no such file") || strings.Contains(errorStr, "permission denied") ||
		strings.Contains(errorStr, "disk full") || strings.Contains(errorStr, "i/o error") {
		return "filesystem"
	}

	// Configuration errors
	if strings.Contains(errorStr, "config") || strings.Contains(errorStr, "invalid") {
		return "configuration"
	}

	// Encoding errors
	if strings.Contains(errorStr, "opus") || strings.Contains(errorStr, "encoding") {
		return "encoding"
	}

	return "unknown"
}

// Helper functions for error classification

// isNetworkError checks if an error is network-related
func isNetworkError(err error) bool {
	// Check for net.Error interface
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	// Check for common network error patterns
	errorStr := strings.ToLower(err.Error())
	networkPatterns := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"network unreachable",
		"host unreachable",
		"no route to host",
		"temporary failure",
		"timeout",
		"dial tcp",
		"i/o timeout",
		"connection aborted",
		"broken pipe",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// isProcessError checks if an error is process-related
func isProcessError(err error) bool {
	// Check for exec.ExitError
	if _, ok := err.(*exec.ExitError); ok {
		return true
	}

	// Check for syscall errors that might be retryable
	if errno, ok := err.(syscall.Errno); ok {
		switch errno {
		case syscall.EAGAIN, syscall.EINTR:
			return true
		}
		// EWOULDBLOCK might be the same as EAGAIN on some systems
		if errno == syscall.EWOULDBLOCK && errno != syscall.EAGAIN {
			return true
		}
	}

	errorStr := strings.ToLower(err.Error())
	processPatterns := []string{
		"process killed",
		"process terminated",
		"signal: killed",
		"signal: terminated",
		"exit status",
	}

	for _, pattern := range processPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// isYtDlpRetryableError checks if a yt-dlp error is retryable
func isYtDlpRetryableError(errorStr string) bool {
	retryablePatterns := []string{
		"http error 429", // Rate limiting
		"http error 503", // Service unavailable
		"http error 502", // Bad gateway
		"http error 504", // Gateway timeout
		"connection timed out",
		"temporary failure",
		"unable to download webpage",
		"download error",
		"network error",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// isFFmpegRetryableError checks if an FFmpeg error is retryable
func isFFmpegRetryableError(errorStr string) bool {
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"i/o error",
		"resource temporarily unavailable",
		"interrupted system call",
		"broken pipe",
		"protocol error",
		"server returned 5", // 5xx HTTP errors
		"timeout",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// isDiscordRetryableError checks if a Discord API error is retryable
func isDiscordRetryableError(errorStr string) bool {
	retryablePatterns := []string{
		"websocket: close 1006", // Abnormal closure
		"websocket: close 4000", // Unknown error
		"rate limit",
		"internal server error",
		"bad gateway",
		"service unavailable",
		"gateway timeout",
		"connection reset",
		"timeout",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// isTemporaryFileSystemError checks if a file system error is temporary/retryable
func isTemporaryFileSystemError(errorStr string) bool {
	temporaryPatterns := []string{
		"resource temporarily unavailable",
		"device busy",
		"interrupted system call",
		"i/o error", // Sometimes temporary
	}

	for _, pattern := range temporaryPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// GetRetryDelay calculates the delay for a specific retry attempt number
// This is a utility method that can be used by callers to get consistent delay calculations
func (beh *BasicErrorHandler) GetRetryDelay(attempt int) time.Duration {
	return beh.calculateExponentialBackoff(attempt)
}

// GetMaxRetries returns the maximum number of retries configured
func (beh *BasicErrorHandler) GetMaxRetries() int {
	return beh.retryConfig.MaxRetries
}

// ShouldRetryAfterAttempts determines if retrying should continue after a given number of attempts
func (beh *BasicErrorHandler) ShouldRetryAfterAttempts(attempts int, err error) bool {
	if attempts >= beh.retryConfig.MaxRetries {
		beh.logger.Info("Maximum retry attempts reached", map[string]interface{}{
			"attempts":    attempts,
			"max_retries": beh.retryConfig.MaxRetries,
			"final_error": err.Error(),
		})
		return false
	}

	return beh.IsRetryableError(err)
}

// CreateRetryableError wraps an error with retry context information
func CreateRetryableError(originalErr error, context string, attempt int) error {
	return fmt.Errorf("retry attempt %d failed in %s: %w", attempt, context, originalErr)
}

// CreateFatalError wraps an error to indicate it should not be retried
func CreateFatalError(originalErr error, reason string) error {
	return fmt.Errorf("fatal error (%s): %w", reason, originalErr)
}

// IsMaxRetriesError checks if an error indicates max retries were exceeded
func IsMaxRetriesError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "max retries exceeded")
}

// CreateMaxRetriesError creates an error indicating max retries were exceeded
func CreateMaxRetriesError(lastErr error, attempts int) error {
	return fmt.Errorf("max retries exceeded after %d attempts, last error: %w", attempts, lastErr)
}
