package audio

import (
	"fmt"
	"math"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// DiscordNotifier implements UserNotifier for Discord notifications
type DiscordNotifier struct {
	session *discordgo.Session
}

// NewDiscordNotifier creates a new Discord notifier
func NewDiscordNotifier(session *discordgo.Session) UserNotifier {
	return &DiscordNotifier{
		session: session,
	}
}

// NotifyError sends a general error notification to Discord
func (dn *DiscordNotifier) NotifyError(channelID string, errorType string, message string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🔧 Audio Pipeline Issue",
		Description: fmt.Sprintf("**Error Type:** %s\n**Details:** %s", errorType, message),
		Color:       0xFFA500, // Orange color for warnings
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err := dn.session.ChannelMessageSendEmbed(channelID, embed)
	return err
}

// NotifyRetry sends a retry notification to Discord
func (dn *DiscordNotifier) NotifyRetry(channelID string, attempt int, maxAttempts int, nextDelay time.Duration) error {
	embed := &discordgo.MessageEmbed{
		Title: "🔄 Retrying Audio Stream",
		Description: fmt.Sprintf("Attempting to recover from audio issue...\n**Attempt:** %d/%d\n**Next retry in:** %s",
			attempt, maxAttempts, nextDelay.Round(time.Second)),
		Color:     0x00BFFF, // Light blue for retry
		Timestamp: time.Now().Format(time.RFC3339),
	}

	_, err := dn.session.ChannelMessageSendEmbed(channelID, embed)
	return err
}

// NotifyFatalError sends a fatal error notification to Discord
func (dn *DiscordNotifier) NotifyFatalError(channelID string, errorType string, message string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Audio Stream Failed",
		Description: fmt.Sprintf("Unable to continue audio playback.\n**Error Type:** %s\n**Details:** %s", errorType, message),
		Color:       0xFF0000, // Red color for fatal errors
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "You may try playing the audio again or contact support if the issue persists.",
		},
	}

	_, err := dn.session.ChannelMessageSendEmbed(channelID, embed)
	return err
}

// BasicErrorHandler implements the ErrorHandler interface with retry logic and centralized logging
type BasicErrorHandler struct {
	retryConfig         *RetryConfig
	logger              AudioLogger
	pipelineLogger      logging.Logger // Centralized pipeline-specific logger
	repository          AudioRepository
	guildID             string
	notifier            UserNotifier
	channelID           string // Channel to send notifications to
	enableNotifications bool
}

// NewBasicErrorHandler creates a new BasicErrorHandler instance
func NewBasicErrorHandler(config *RetryConfig, logger AudioLogger, repo AudioRepository, guildID string) ErrorHandler {
	// Create pipeline-specific logger for error handling context
	loggerFactory := logging.GetGlobalLoggerFactory()
	pipelineLogger := loggerFactory.CreateAudioLogger(guildID).WithPipeline("error-handler")

	return &BasicErrorHandler{
		retryConfig:         config,
		logger:              logger,
		pipelineLogger:      pipelineLogger,
		repository:          repo,
		guildID:             guildID,
		enableNotifications: false, // Disabled by default, can be enabled via SetNotifier
	}
}

// NewBasicErrorHandlerWithNotifier creates a new BasicErrorHandler with user notification support
func NewBasicErrorHandlerWithNotifier(config *RetryConfig, logger AudioLogger, repo AudioRepository, guildID string, notifier UserNotifier, channelID string) ErrorHandler {
	// Create pipeline-specific logger for error handling context
	loggerFactory := logging.GetGlobalLoggerFactory()
	pipelineLogger := loggerFactory.CreateAudioLogger(guildID).WithPipeline("error-handler").WithContext(map[string]interface{}{
		"channel_id":            channelID,
		"notifications_enabled": true,
	})

	return &BasicErrorHandler{
		retryConfig:         config,
		logger:              logger,
		pipelineLogger:      pipelineLogger,
		repository:          repo,
		guildID:             guildID,
		notifier:            notifier,
		channelID:           channelID,
		enableNotifications: true,
	}
}

// SetNotifier enables user notifications by setting the notifier and channel
func (beh *BasicErrorHandler) SetNotifier(notifier UserNotifier, channelID string) {
	beh.notifier = notifier
	beh.channelID = channelID
	beh.enableNotifications = true

	// Update pipeline logger context with notification settings
	beh.pipelineLogger = beh.pipelineLogger.WithContext(map[string]interface{}{
		"channel_id":            channelID,
		"notifications_enabled": true,
	})
}

// DisableNotifications disables user notifications
func (beh *BasicErrorHandler) DisableNotifications() {
	beh.enableNotifications = false

	// Update pipeline logger context
	beh.pipelineLogger = beh.pipelineLogger.WithContext(map[string]interface{}{
		"notifications_enabled": false,
	})
}

// HandleError processes an error and determines if it should be retried and with what delay
func (beh *BasicErrorHandler) HandleError(err error, context string) (shouldRetry bool, delay time.Duration) {
	// Create context-specific logger for this error handling session
	contextLogger := beh.pipelineLogger.WithContext(CreateContextFieldsWithComponent(beh.guildID, "", "", "error-handler"))

	// Log the error first with enhanced context
	beh.LogError(err, context)

	// Check if the error is retryable
	if !beh.IsRetryableError(err) {
		contextLogger.Info("Error is not retryable, skipping retry logic", map[string]interface{}{
			"error":      err.Error(),
			"context":    context,
			"error_type": beh.classifyErrorType(err),
		})

		// Notify user of fatal error if notifications are enabled
		if beh.enableNotifications && beh.notifier != nil && beh.channelID != "" {
			errorType := beh.classifyErrorType(err)
			userMessage := beh.createUserFriendlyErrorMessage(err, errorType)
			if notifyErr := beh.notifier.NotifyFatalError(beh.channelID, errorType, userMessage); notifyErr != nil {
				contextLogger.Warn("Failed to send fatal error notification to user", map[string]interface{}{
					"notification_error": notifyErr.Error(),
					"original_error":     err.Error(),
					"channel_id":         beh.channelID,
				})
			}
		}

		return false, 0
	}

	// For retryable errors, calculate appropriate backoff delay
	// Use simple backoff for streaming-related errors, exponential for others
	if beh.isStreamingRelatedError(err) {
		delay = beh.calculateStreamingBackoff(1) // Start with attempt 1 for base calculation
	} else {
		delay = beh.calculateExponentialBackoff(1) // Start with attempt 1 for base calculation
	}

	contextLogger.Info("Error is retryable, will attempt retry", map[string]interface{}{
		"error":       err.Error(),
		"context":     context,
		"error_type":  beh.classifyErrorType(err),
		"retry_delay": delay.String(),
		"max_retries": beh.retryConfig.MaxRetries,
	})

	// Notify user of retry attempt if notifications are enabled
	if beh.enableNotifications && beh.notifier != nil && beh.channelID != "" {
		if notifyErr := beh.notifier.NotifyRetry(beh.channelID, 1, beh.retryConfig.MaxRetries, delay); notifyErr != nil {
			contextLogger.Warn("Failed to send retry notification to user", map[string]interface{}{
				"notification_error": notifyErr.Error(),
				"original_error":     err.Error(),
				"channel_id":         beh.channelID,
			})
		}
	}

	return true, delay
}

// LogError logs an error to both console and database with enhanced context and debugging information
func (beh *BasicErrorHandler) LogError(err error, context string) {
	// Classify the error type for database storage
	errorType := beh.classifyErrorType(err)

	// Create enhanced debugging context using shared utilities
	debugContext := beh.createDebugContext(err, context, errorType)

	// Log to centralized logging system with pipeline context
	contextLogger := beh.pipelineLogger.WithContext(debugContext)
	contextLogger.Error("Audio pipeline error occurred", err, map[string]interface{}{
		"error_type": errorType,
		"context":    context,
		"retryable":  beh.IsRetryableError(err),
	})

	// Also log to legacy AudioLogger for backward compatibility
	beh.logger.Error("Audio pipeline error occurred", err, debugContext)

	// Save to database if repository is available
	if beh.repository != nil {
		// Use shared utility to create consistent error record
		audioError := CreateAudioError(beh.guildID, errorType, err.Error(), beh.formatContextForDatabase(context, debugContext))

		if saveErr := beh.repository.SaveError(audioError); saveErr != nil {
			contextLogger.Warn("Failed to save error to database", map[string]interface{}{
				"save_error":     saveErr.Error(),
				"original_error": err.Error(),
				"context":        context,
				"error_type":     errorType,
			})
		} else {
			contextLogger.Debug("Error successfully saved to database", map[string]interface{}{
				"error_id":   audioError.ID.String(),
				"error_type": errorType,
				"context":    context,
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
	errorType := beh.classifyErrorType(err)

	// Network-related errors (retryable)
	if isNetworkError(err) {
		beh.pipelineLogger.Debug("Error classified as retryable network error", map[string]interface{}{
			"error":                 err.Error(),
			"error_type":            errorType,
			"classification_reason": "network error pattern matched",
		})
		return true
	}

	// Process-related errors (retryable)
	if isProcessError(err) {
		beh.pipelineLogger.Debug("Error classified as retryable process error", map[string]interface{}{
			"error":                 err.Error(),
			"error_type":            errorType,
			"classification_reason": "process error pattern matched",
		})
		return true
	}

	// Streaming pipeline errors (retryable)
	if isStreamingPipelineError(err) {
		beh.pipelineLogger.Debug("Error classified as retryable streaming pipeline error", map[string]interface{}{
			"error":                 err.Error(),
			"error_type":            errorType,
			"classification_reason": "streaming pipeline error pattern matched",
		})
		return true
	}

	// URL expiry errors (retryable)
	if isURLExpiryError(err) {
		beh.pipelineLogger.Debug("Error classified as retryable URL expiry error", map[string]interface{}{
			"error":                 err.Error(),
			"error_type":            errorType,
			"classification_reason": "URL expiry error pattern matched",
		})
		return true
	}

	// yt-dlp streaming errors (retryable)
	if isYtDlpStreamingError(err) {
		beh.pipelineLogger.Debug("Error classified as retryable yt-dlp streaming error", map[string]interface{}{
			"error":                 err.Error(),
			"error_type":            errorType,
			"classification_reason": "yt-dlp streaming error pattern matched",
		})
		return true
	}

	// FFmpeg streaming errors (retryable)
	if isFFmpegStreamingError(err) {
		beh.pipelineLogger.Debug("Error classified as retryable FFmpeg streaming error", map[string]interface{}{
			"error":                 err.Error(),
			"error_type":            errorType,
			"classification_reason": "FFmpeg streaming error pattern matched",
		})
		return true
	}

	// yt-dlp specific retryable errors (general)
	if isYtDlpRetryableError(errorStr) {
		beh.pipelineLogger.Debug("Error classified as retryable yt-dlp error", map[string]interface{}{
			"error":                 err.Error(),
			"error_type":            errorType,
			"classification_reason": "yt-dlp retryable pattern matched",
		})
		return true
	}

	// FFmpeg specific retryable errors (general)
	if isFFmpegRetryableError(errorStr) {
		beh.pipelineLogger.Debug("Error classified as retryable FFmpeg error", map[string]interface{}{
			"error":                 err.Error(),
			"error_type":            errorType,
			"classification_reason": "FFmpeg retryable pattern matched",
		})
		return true
	}

	// Discord API retryable errors
	if isDiscordRetryableError(errorStr) {
		beh.pipelineLogger.Debug("Error classified as retryable Discord error", map[string]interface{}{
			"error":                 err.Error(),
			"error_type":            errorType,
			"classification_reason": "Discord retryable pattern matched",
		})
		return true
	}

	// Temporary file system errors (retryable)
	if isTemporaryFileSystemError(errorStr) {
		beh.pipelineLogger.Debug("Error classified as retryable filesystem error", map[string]interface{}{
			"error":                 err.Error(),
			"error_type":            errorType,
			"classification_reason": "temporary filesystem error pattern matched",
		})
		return true
	}

	// Default to non-retryable for unknown errors
	beh.pipelineLogger.Debug("Error classified as non-retryable", map[string]interface{}{
		"error":                 err.Error(),
		"error_type":            errorType,
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

// calculateStreamingBackoff calculates simple backoff delays for streaming failures (2s, 5s, 10s)
func (beh *BasicErrorHandler) calculateStreamingBackoff(attempt int) time.Duration {
	// Simple backoff for streaming failures: 2s, 5s, 10s
	switch attempt {
	case 1:
		return 2 * time.Second
	case 2:
		return 5 * time.Second
	case 3:
		return 10 * time.Second
	default:
		// For attempts beyond 3, use the maximum delay
		return 10 * time.Second
	}
}

// isStreamingRelatedError checks if an error is streaming-related and should use simple backoff
func (beh *BasicErrorHandler) isStreamingRelatedError(err error) bool {
	return isStreamingPipelineError(err) ||
		isURLExpiryError(err) ||
		isYtDlpStreamingError(err) ||
		isFFmpegStreamingError(err)
}

// classifyErrorType returns a string classification of the error type for database storage
func (beh *BasicErrorHandler) classifyErrorType(err error) string {
	if err == nil {
		return "unknown"
	}

	errorStr := strings.ToLower(err.Error())

	// Streaming pipeline errors (new for streaming improvements)
	if isStreamingPipelineError(err) {
		return "streaming_pipeline"
	}

	// URL expiry/refresh errors (new for streaming improvements)
	if isURLExpiryError(err) {
		return "url_expiry"
	}

	// yt-dlp specific streaming errors (enhanced classification)
	if isYtDlpStreamingError(err) {
		return "yt-dlp_streaming"
	}

	// FFmpeg streaming errors (enhanced classification)
	if isFFmpegStreamingError(err) {
		return "ffmpeg_streaming"
	}

	// Network errors
	if isNetworkError(err) {
		return "network"
	}

	// Process errors
	if isProcessError(err) {
		return "process"
	}

	// yt-dlp errors (general)
	if strings.Contains(errorStr, "yt-dlp") || strings.Contains(errorStr, "youtube-dl") {
		return "yt-dlp"
	}

	// FFmpeg errors (general)
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
		return netErr.Timeout()
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
		"ffmpeg failed", // General FFmpeg failures can be retryable
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

// isStreamingPipelineError checks if an error is related to the streaming pipeline
func isStreamingPipelineError(err error) bool {
	errorStr := strings.ToLower(err.Error())

	streamingPatterns := []string{
		"streaming pipeline",
		"pipeline failed",
		"pipe broken",
		"pipe closed",
		"process chain",
		"pipeline error",
		"stream interrupted",
		"pipeline timeout",
		"process synchronization",
		"pipeline coordination",
	}

	for _, pattern := range streamingPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// isURLExpiryError checks if an error is related to URL expiration or refresh
func isURLExpiryError(err error) bool {
	errorStr := strings.ToLower(err.Error())

	urlExpiryPatterns := []string{
		"url expired",
		"url expiry",
		"url refresh",
		"stream url",
		"url invalid",
		"url not found",
		"expired stream",
		"stream expired",
		"url ttl",
		"refresh failed",
		"url lifecycle",
	}

	for _, pattern := range urlExpiryPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// isYtDlpStreamingError checks if an error is specifically related to yt-dlp streaming operations
func isYtDlpStreamingError(err error) bool {
	errorStr := strings.ToLower(err.Error())

	// Must contain yt-dlp reference AND streaming-related patterns
	hasYtDlp := strings.Contains(errorStr, "yt-dlp") || strings.Contains(errorStr, "youtube-dl")
	if !hasYtDlp {
		return false
	}

	streamingPatterns := []string{
		"streaming",
		"pipe",
		"stdout",
		"output",
		"format extraction",
		"stream extraction",
		"download interrupted",
		"stream failed",
		"extraction failed",
		"format unavailable",
	}

	for _, pattern := range streamingPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// isFFmpegStreamingError checks if an error is specifically related to FFmpeg streaming operations
func isFFmpegStreamingError(err error) bool {
	errorStr := strings.ToLower(err.Error())

	// Must contain ffmpeg reference AND streaming-related patterns
	hasFFmpeg := strings.Contains(errorStr, "ffmpeg")
	if !hasFFmpeg {
		return false
	}

	streamingPatterns := []string{
		"pipe",
		"stdin",
		"stdout",
		"streaming",
		"input/output error",
		"broken pipe",
		"end of file",
		"invalid data found",
		"stream mapping",
		"codec parameters",
		"format detection",
	}

	for _, pattern := range streamingPatterns {
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

// GetRetryDelayForError calculates the delay for a specific retry attempt based on error type
// Uses simple backoff (2s, 5s, 10s) for streaming errors, exponential backoff for others
func (beh *BasicErrorHandler) GetRetryDelayForError(err error, attempt int) time.Duration {
	if beh.isStreamingRelatedError(err) {
		return beh.calculateStreamingBackoff(attempt)
	}
	return beh.calculateExponentialBackoff(attempt)
}

// GetMaxRetries returns the maximum number of retries configured
func (beh *BasicErrorHandler) GetMaxRetries() int {
	return beh.retryConfig.MaxRetries
}

// ShouldRetryAfterAttempts determines if retrying should continue after a given number of attempts
func (beh *BasicErrorHandler) ShouldRetryAfterAttempts(attempts int, err error) bool {
	if attempts >= beh.retryConfig.MaxRetries {
		contextLogger := beh.pipelineLogger.WithContext(CreateContextFieldsWithComponent(beh.guildID, "", "", "retry-logic"))
		contextLogger.Info("Maximum retry attempts reached", map[string]interface{}{
			"attempts":    attempts,
			"max_retries": beh.retryConfig.MaxRetries,
			"final_error": err.Error(),
			"error_type":  beh.classifyErrorType(err),
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

// Streaming-specific error creation functions

// CreateStreamingPipelineError creates an error for streaming pipeline failures
func CreateStreamingPipelineError(component string, originalErr error) error {
	return fmt.Errorf("streaming pipeline error in %s: %w", component, originalErr)
}

// CreateURLExpiryError creates an error for URL expiry/refresh failures
func CreateURLExpiryError(url string, originalErr error) error {
	return fmt.Errorf("url expiry error for %s: %w", url, originalErr)
}

// CreateYtDlpStreamingError creates an error for yt-dlp streaming failures
func CreateYtDlpStreamingError(operation string, originalErr error) error {
	return fmt.Errorf("yt-dlp streaming error during %s: %w", operation, originalErr)
}

// CreateFFmpegStreamingError creates an error for FFmpeg streaming failures
func CreateFFmpegStreamingError(operation string, originalErr error) error {
	return fmt.Errorf("ffmpeg streaming error during %s: %w", operation, originalErr)
}

// IsStreamingError checks if an error is any type of streaming-related error
func IsStreamingError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := strings.ToLower(err.Error())
	return strings.Contains(errorStr, "streaming pipeline") ||
		strings.Contains(errorStr, "url expiry") ||
		strings.Contains(errorStr, "yt-dlp streaming") ||
		strings.Contains(errorStr, "ffmpeg streaming")
}

// createDebugContext creates enhanced debugging context for error logging
func (beh *BasicErrorHandler) createDebugContext(err error, context string, errorType string) map[string]interface{} {
	// Start with shared context fields using utility function
	debugContext := CreateContextFieldsWithComponent(beh.guildID, "", "", "error-handler")

	// Add error-specific context
	debugContext["context"] = context
	debugContext["error_type"] = errorType
	debugContext["retryable"] = beh.IsRetryableError(err)
	debugContext["max_retries"] = beh.retryConfig.MaxRetries
	debugContext["base_delay"] = beh.retryConfig.BaseDelay.String()
	debugContext["max_delay"] = beh.retryConfig.MaxDelay.String()

	// Add error-specific debugging information
	switch errorType {
	case "streaming_pipeline":
		debugContext["streaming_pipeline_details"] = beh.extractStreamingPipelineErrorDetails(err)
	case "url_expiry":
		debugContext["url_expiry_details"] = beh.extractURLExpiryErrorDetails(err)
	case "yt-dlp_streaming":
		debugContext["ytdlp_streaming_details"] = beh.extractYtDlpStreamingErrorDetails(err)
	case "ffmpeg_streaming":
		debugContext["ffmpeg_streaming_details"] = beh.extractFFmpegStreamingErrorDetails(err)
	case "network":
		debugContext["network_error_details"] = beh.extractNetworkErrorDetails(err)
	case "process":
		debugContext["process_error_details"] = beh.extractProcessErrorDetails(err)
	case "yt-dlp":
		debugContext["ytdlp_error_details"] = beh.extractYtDlpErrorDetails(err)
	case "ffmpeg":
		debugContext["ffmpeg_error_details"] = beh.extractFFmpegErrorDetails(err)
	case "discord":
		debugContext["discord_error_details"] = beh.extractDiscordErrorDetails(err)
	case "filesystem":
		debugContext["filesystem_error_details"] = beh.extractFilesystemErrorDetails(err)
	case "encoding":
		debugContext["encoding_error_details"] = beh.extractEncodingErrorDetails(err)
	}

	// Add system context if available
	if beh.channelID != "" {
		debugContext["notification_channel"] = beh.channelID
	}
	debugContext["notifications_enabled"] = beh.enableNotifications

	return debugContext
}

// formatContextForDatabase formats the context and debug information for database storage
func (beh *BasicErrorHandler) formatContextForDatabase(originalContext string, debugContext map[string]interface{}) string {
	contextParts := []string{originalContext}

	// Add key debugging information to the context string
	if errorType, ok := debugContext["error_type"].(string); ok {
		contextParts = append(contextParts, fmt.Sprintf("type=%s", errorType))
	}

	if retryable, ok := debugContext["retryable"].(bool); ok {
		contextParts = append(contextParts, fmt.Sprintf("retryable=%t", retryable))
	}

	if timestamp, ok := debugContext["timestamp"].(string); ok {
		contextParts = append(contextParts, fmt.Sprintf("timestamp=%s", timestamp))
	}

	return strings.Join(contextParts, "; ")
}

// createUserFriendlyErrorMessage creates a user-friendly error message for Discord notifications
func (beh *BasicErrorHandler) createUserFriendlyErrorMessage(err error, errorType string) string {
	switch errorType {
	case "streaming_pipeline":
		return "Audio streaming pipeline encountered an issue. The system will automatically retry to restore playback."
	case "url_expiry":
		return "The audio stream URL has expired. The system is refreshing the connection to continue playback."
	case "yt-dlp_streaming":
		return "Issue with audio stream extraction. This is usually temporary and the system will retry automatically."
	case "ffmpeg_streaming":
		return "Audio stream processing encountered an issue. The system will attempt to restart the audio pipeline."
	case "network":
		return "Network connection issue. This might be temporary - please try again in a few moments."
	case "yt-dlp":
		return "Unable to download audio from the provided URL. The video might be unavailable or restricted."
	case "ffmpeg":
		return "Audio processing failed. This might be due to an unsupported audio format or temporary issue."
	case "discord":
		return "Discord connection issue. The bot might have lost connection to the voice channel."
	case "process":
		return "Audio processing system encountered an issue. This is usually temporary."
	case "filesystem":
		return "File system issue encountered. This might be due to insufficient disk space or permissions."
	case "encoding":
		return "Audio encoding failed. This might be due to corrupted audio data or system resources."
	case "configuration":
		return "Configuration issue detected. Please contact the bot administrator."
	default:
		// For unknown errors, provide a generic but helpful message
		errorStr := strings.ToLower(err.Error())
		if strings.Contains(errorStr, "timeout") {
			return "Request timed out. This might be due to slow network or server issues."
		}
		if strings.Contains(errorStr, "not found") || strings.Contains(errorStr, "404") {
			return "The requested audio content was not found or is no longer available."
		}
		if strings.Contains(errorStr, "forbidden") || strings.Contains(errorStr, "403") {
			return "Access to the audio content is restricted or forbidden."
		}
		if strings.Contains(errorStr, "rate limit") {
			return "Too many requests. Please wait a moment before trying again."
		}
		return "An unexpected issue occurred while processing the audio. Please try again."
	}
}

// Error detail extraction methods for enhanced debugging

func (beh *BasicErrorHandler) extractNetworkErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})

	if netErr, ok := err.(net.Error); ok {
		details["timeout"] = netErr.Timeout()
		details["temporary"] = isTemporaryFileSystemError(err.Error())
	}

	errorStr := strings.ToLower(err.Error())
	details["connection_refused"] = strings.Contains(errorStr, "connection refused")
	details["connection_reset"] = strings.Contains(errorStr, "connection reset")
	details["timeout_detected"] = strings.Contains(errorStr, "timeout")
	details["dns_issue"] = strings.Contains(errorStr, "no such host") || strings.Contains(errorStr, "name resolution")

	return details
}

func (beh *BasicErrorHandler) extractProcessErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})

	if exitErr, ok := err.(*exec.ExitError); ok {
		details["exit_code"] = exitErr.ExitCode()
		details["stderr"] = string(exitErr.Stderr)
	}

	if errno, ok := err.(syscall.Errno); ok {
		details["syscall_errno"] = int(errno)
		details["errno_name"] = errno.Error()
	}

	errorStr := strings.ToLower(err.Error())
	details["killed"] = strings.Contains(errorStr, "killed")
	details["terminated"] = strings.Contains(errorStr, "terminated")

	return details
}

func (beh *BasicErrorHandler) extractYtDlpErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})
	errorStr := strings.ToLower(err.Error())

	details["http_error"] = strings.Contains(errorStr, "http error")
	details["rate_limited"] = strings.Contains(errorStr, "429") || strings.Contains(errorStr, "rate limit")
	details["unavailable"] = strings.Contains(errorStr, "unavailable") || strings.Contains(errorStr, "not available")
	details["private_video"] = strings.Contains(errorStr, "private") || strings.Contains(errorStr, "forbidden")
	details["geo_blocked"] = strings.Contains(errorStr, "geo") || strings.Contains(errorStr, "region")

	return details
}

func (beh *BasicErrorHandler) extractFFmpegErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})
	errorStr := strings.ToLower(err.Error())

	details["invalid_data"] = strings.Contains(errorStr, "invalid data")
	details["protocol_error"] = strings.Contains(errorStr, "protocol")
	details["format_error"] = strings.Contains(errorStr, "format")
	details["codec_error"] = strings.Contains(errorStr, "codec")
	details["stream_error"] = strings.Contains(errorStr, "stream")

	return details
}

func (beh *BasicErrorHandler) extractDiscordErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})
	errorStr := strings.ToLower(err.Error())

	details["websocket_error"] = strings.Contains(errorStr, "websocket")
	details["rate_limited"] = strings.Contains(errorStr, "rate limit")
	details["voice_error"] = strings.Contains(errorStr, "voice")
	details["connection_closed"] = strings.Contains(errorStr, "close")
	details["api_error"] = strings.Contains(errorStr, "api")

	return details
}

func (beh *BasicErrorHandler) extractFilesystemErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})
	errorStr := strings.ToLower(err.Error())

	details["permission_denied"] = strings.Contains(errorStr, "permission denied")
	details["no_space"] = strings.Contains(errorStr, "no space") || strings.Contains(errorStr, "disk full")
	details["file_not_found"] = strings.Contains(errorStr, "no such file")
	details["io_error"] = strings.Contains(errorStr, "i/o error")

	return details
}

func (beh *BasicErrorHandler) extractEncodingErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})
	errorStr := strings.ToLower(err.Error())

	details["opus_error"] = strings.Contains(errorStr, "opus")
	details["pcm_error"] = strings.Contains(errorStr, "pcm")
	details["frame_size_error"] = strings.Contains(errorStr, "frame size")
	details["sample_rate_error"] = strings.Contains(errorStr, "sample rate")

	return details
}

// extractStreamingPipelineErrorDetails extracts details for streaming pipeline errors
func (beh *BasicErrorHandler) extractStreamingPipelineErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})
	errorStr := strings.ToLower(err.Error())

	details["pipeline_failed"] = strings.Contains(errorStr, "pipeline failed")
	details["pipe_broken"] = strings.Contains(errorStr, "pipe broken") || strings.Contains(errorStr, "broken pipe")
	details["pipe_closed"] = strings.Contains(errorStr, "pipe closed")
	details["process_chain_error"] = strings.Contains(errorStr, "process chain")
	details["stream_interrupted"] = strings.Contains(errorStr, "stream interrupted")
	details["pipeline_timeout"] = strings.Contains(errorStr, "pipeline timeout")
	details["coordination_error"] = strings.Contains(errorStr, "coordination") || strings.Contains(errorStr, "synchronization")

	return details
}

// extractURLExpiryErrorDetails extracts details for URL expiry/refresh errors
func (beh *BasicErrorHandler) extractURLExpiryErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})
	errorStr := strings.ToLower(err.Error())

	details["url_expired"] = strings.Contains(errorStr, "url expired") || strings.Contains(errorStr, "expired")
	details["url_invalid"] = strings.Contains(errorStr, "url invalid") || strings.Contains(errorStr, "invalid")
	details["refresh_failed"] = strings.Contains(errorStr, "refresh failed") || strings.Contains(errorStr, "refresh")
	details["stream_url_error"] = strings.Contains(errorStr, "stream url")
	details["url_not_found"] = strings.Contains(errorStr, "url not found") || strings.Contains(errorStr, "not found")
	details["ttl_exceeded"] = strings.Contains(errorStr, "ttl") || strings.Contains(errorStr, "time to live")

	return details
}

// extractYtDlpStreamingErrorDetails extracts details for yt-dlp streaming-specific errors
func (beh *BasicErrorHandler) extractYtDlpStreamingErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})
	errorStr := strings.ToLower(err.Error())

	details["extraction_failed"] = strings.Contains(errorStr, "extraction failed")
	details["format_unavailable"] = strings.Contains(errorStr, "format unavailable")
	details["stream_extraction_error"] = strings.Contains(errorStr, "stream extraction")
	details["download_interrupted"] = strings.Contains(errorStr, "download interrupted")
	details["pipe_output_error"] = strings.Contains(errorStr, "pipe") || strings.Contains(errorStr, "stdout")
	details["streaming_mode_error"] = strings.Contains(errorStr, "streaming")

	return details
}

// extractFFmpegStreamingErrorDetails extracts details for FFmpeg streaming-specific errors
func (beh *BasicErrorHandler) extractFFmpegStreamingErrorDetails(err error) map[string]interface{} {
	details := make(map[string]interface{})
	errorStr := strings.ToLower(err.Error())

	details["pipe_input_error"] = strings.Contains(errorStr, "pipe") || strings.Contains(errorStr, "stdin")
	details["format_detection_error"] = strings.Contains(errorStr, "format detection")
	details["codec_parameters_error"] = strings.Contains(errorStr, "codec parameters")
	details["stream_mapping_error"] = strings.Contains(errorStr, "stream mapping")
	details["invalid_data_error"] = strings.Contains(errorStr, "invalid data found")
	details["end_of_file_error"] = strings.Contains(errorStr, "end of file")
	details["io_error"] = strings.Contains(errorStr, "input/output error")

	return details
}

// NotifyRetryAttempt sends a notification for a specific retry attempt
func (beh *BasicErrorHandler) NotifyRetryAttempt(attempt int, err error, delay time.Duration) {
	if !beh.enableNotifications || beh.notifier == nil || beh.channelID == "" {
		return
	}

	// Log retry attempt with centralized logging
	contextLogger := beh.pipelineLogger.WithContext(CreateContextFieldsWithComponent(beh.guildID, "", "", "notification"))
	contextLogger.Info("Sending retry notification to user", map[string]interface{}{
		"retry_attempt": attempt,
		"max_retries":   beh.retryConfig.MaxRetries,
		"retry_delay":   FormatDuration(delay),
		"error_type":    beh.classifyErrorType(err),
		"channel_id":    beh.channelID,
	})

	if notifyErr := beh.notifier.NotifyRetry(beh.channelID, attempt, beh.retryConfig.MaxRetries, delay); notifyErr != nil {
		contextLogger.Warn("Failed to send retry attempt notification to user", map[string]interface{}{
			"notification_error": notifyErr.Error(),
			"retry_attempt":      attempt,
			"original_error":     err.Error(),
			"channel_id":         beh.channelID,
		})
	}
}

// NotifyMaxRetriesExceeded sends a notification when max retries are exceeded
func (beh *BasicErrorHandler) NotifyMaxRetriesExceeded(finalErr error, attempts int) {
	if !beh.enableNotifications || beh.notifier == nil || beh.channelID == "" {
		return
	}

	errorType := beh.classifyErrorType(finalErr)
	userMessage := fmt.Sprintf("%s\n\nTried %d times but couldn't recover. You may try playing the audio again.",
		beh.createUserFriendlyErrorMessage(finalErr, errorType), attempts)

	// Log max retries exceeded with centralized logging
	contextLogger := beh.pipelineLogger.WithContext(CreateContextFieldsWithComponent(beh.guildID, "", "", "notification"))
	contextLogger.Info("Sending max retries exceeded notification to user", map[string]interface{}{
		"final_attempts": attempts,
		"max_retries":    beh.retryConfig.MaxRetries,
		"error_type":     errorType,
		"channel_id":     beh.channelID,
	})

	if notifyErr := beh.notifier.NotifyFatalError(beh.channelID, errorType, userMessage); notifyErr != nil {
		contextLogger.Warn("Failed to send max retries exceeded notification to user", map[string]interface{}{
			"notification_error": notifyErr.Error(),
			"final_error":        finalErr.Error(),
			"attempts":           attempts,
			"channel_id":         beh.channelID,
		})
	}
}

// GetChannelID returns the current notification channel ID
func (beh *BasicErrorHandler) GetChannelID() string {
	return beh.channelID
}

// IsNotificationEnabled returns whether notifications are currently enabled
func (beh *BasicErrorHandler) IsNotificationEnabled() bool {
	return beh.enableNotifications && beh.notifier != nil && beh.channelID != ""
}
