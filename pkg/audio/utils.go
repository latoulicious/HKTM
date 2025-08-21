package audio

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/latoulicious/HKTM/pkg/database/models"
)

// ValidateURL validates that a URL is not empty and has a basic valid format
// Used by Controller, ErrorHandler, Metrics
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Trim whitespace
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Check for basic URL patterns
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") && !strings.HasPrefix(urlStr, "www.") {
		// Check if it's a YouTube URL without protocol
		if !strings.Contains(urlStr, "youtube.com") && !strings.Contains(urlStr, "youtu.be") {
			return fmt.Errorf("URL must be a valid HTTP/HTTPS URL or YouTube URL")
		}
	}

	// Try to parse as URL if it has a protocol
	if strings.HasPrefix(urlStr, "http://") || strings.HasPrefix(urlStr, "https://") {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return fmt.Errorf("invalid URL format: %w", err)
		}
		// Check if the URL has a valid host
		if parsedURL.Host == "" {
			return fmt.Errorf("URL must have a valid host")
		}
	}

	return nil
}

// FormatDuration formats a duration into a human-readable string
// Used by Controller, Metrics, Logger
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	hours := minutes / 60
	minutes = minutes % 60

	return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
}

// CreateContextFields creates a standardized context map for logging and metrics
// Used by Logger, ErrorHandler, Metrics
func CreateContextFields(guildID, userID, url string) map[string]interface{} {
	fields := map[string]interface{}{
		"timestamp": time.Now(),
	}

	if guildID != "" {
		fields["guild_id"] = guildID
	}

	if userID != "" {
		fields["user_id"] = userID
	}

	if url != "" {
		fields["url"] = url
	}

	return fields
}

// CreateContextFieldsWithComponent creates context fields with an additional component field
// Used by Logger, ErrorHandler, Metrics when component context is needed
func CreateContextFieldsWithComponent(guildID, userID, url, component string) map[string]interface{} {
	fields := CreateContextFields(guildID, userID, url)

	if component != "" {
		fields["component"] = component
	}

	return fields
}

// IsYouTubeURL checks if a URL appears to be from YouTube
// Used by Controller, ErrorHandler for YouTube-specific handling
func IsYouTubeURL(urlStr string) bool {
	return strings.Contains(urlStr, "youtube.com") || strings.Contains(urlStr, "youtu.be")
}

// SanitizeURL removes sensitive information from URLs for logging
// Used by Logger, ErrorHandler, Metrics for safe logging
func SanitizeURL(urlStr string) string {
	if urlStr == "" {
		return ""
	}

	// Parse the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		// If parsing fails, just return the original URL truncated
		if len(urlStr) > 100 {
			return urlStr[:100] + "..."
		}
		return urlStr
	}

	// For YouTube URLs, preserve the video ID but remove other parameters
	if IsYouTubeURL(urlStr) {
		if videoID := parsedURL.Query().Get("v"); videoID != "" {
			// Keep only the video ID parameter
			parsedURL.RawQuery = "v=" + videoID
		} else {
			// No video ID found, remove all query params
			parsedURL.RawQuery = ""
		}
		parsedURL.Fragment = ""
		return parsedURL.String()
	}

	// For other URLs, remove all query parameters and fragments
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	// Truncate if too long
	sanitized := parsedURL.String()
	if len(sanitized) > 100 {
		return sanitized[:100] + "..."
	}

	return sanitized
}

// CreateAudioMetric creates a new AudioMetric with common fields populated
// Used by Metrics, Logger, ErrorHandler for consistent metric creation
func CreateAudioMetric(guildID, metricType string, value float64) *models.AudioMetric {
	return &models.AudioMetric{
		ID:         uuid.New(),
		GuildID:    guildID,
		MetricType: metricType,
		Value:      value,
		Timestamp:  time.Now(),
	}
}

// CreateAudioLog creates a new AudioLog with common fields populated
// Used by Logger, ErrorHandler for consistent log creation
func CreateAudioLog(guildID, level, message, errorMsg string, fields map[string]interface{}) *models.AudioLog {
	return &models.AudioLog{
		ID:        uuid.New(),
		GuildID:   guildID,
		Level:     level,
		Message:   message,
		Error:     errorMsg,
		Fields:    fields,
		Timestamp: time.Now(),
	}
}

// CreateAudioError creates a new AudioError with common fields populated
// Used by ErrorHandler, Metrics for consistent error creation
func CreateAudioError(guildID, errorType, errorMsg, context string) *models.AudioError {
	return &models.AudioError{
		ID:        uuid.New(),
		GuildID:   guildID,
		ErrorType: errorType,
		ErrorMsg:  errorMsg,
		Context:   context,
		Timestamp: time.Now(),
		Resolved:  false,
	}
}

// ValidateBinaryDependency validates that a required binary is available and executable
// Used by Factory, ConfigProvider for dependency validation
func ValidateBinaryDependency(binaryName, binaryPath string) error {
	if binaryPath == "" {
		return fmt.Errorf("%s binary path cannot be empty", binaryName)
	}

	// Check if binary exists and is executable
	_, err := exec.LookPath(binaryPath)
	if err != nil {
		return fmt.Errorf("%s binary not found at path '%s': %w", binaryName, binaryPath, err)
	}

	return nil
}

// ValidateFFmpegBinary validates FFmpeg binary and basic functionality
// Used by Factory, StreamProcessor for FFmpeg validation
func ValidateFFmpegBinary(binaryPath string) error {
	if err := ValidateBinaryDependency("ffmpeg", binaryPath); err != nil {
		return err
	}

	// Test basic FFmpeg functionality with version check
	cmd := exec.Command(binaryPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ffmpeg binary validation failed - unable to get version: %w", err)
	}

	// Check if output contains expected FFmpeg version information
	outputStr := string(output)
	if !strings.Contains(outputStr, "ffmpeg version") {
		return fmt.Errorf("ffmpeg binary validation failed - unexpected version output")
	}

	return nil
}

// ValidateYtDlpBinary validates yt-dlp binary and basic functionality
// Used by Factory, StreamProcessor for yt-dlp validation
func ValidateYtDlpBinary(binaryPath string) error {
	if err := ValidateBinaryDependency("yt-dlp", binaryPath); err != nil {
		return err
	}

	// Test basic yt-dlp functionality with version check
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("yt-dlp binary validation failed - unable to get version: %w", err)
	}

	// Check if output contains version information (yt-dlp outputs just the version number)
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return fmt.Errorf("yt-dlp binary validation failed - empty version output")
	}

	return nil
}

// ValidateAllBinaryDependencies validates all required binary dependencies
// Used by Factory for comprehensive dependency validation
func ValidateAllBinaryDependencies(binaryPaths ...string) error {
	if len(binaryPaths) < 2 {
		return fmt.Errorf("at least ffmpeg and yt-dlp paths must be provided")
	}

	// Validate FFmpeg (first parameter)
	if err := ValidateFFmpegBinary(binaryPaths[0]); err != nil {
		return fmt.Errorf("FFmpeg validation failed: %w", err)
	}

	// Validate yt-dlp (second parameter)
	if err := ValidateYtDlpBinary(binaryPaths[1]); err != nil {
		return fmt.Errorf("yt-dlp validation failed: %w", err)
	}

	// Validate additional streaming binaries if provided
	if len(binaryPaths) > 2 {
		// Streaming yt-dlp path (third parameter)
		if binaryPaths[2] != "" && binaryPaths[2] != binaryPaths[1] {
			if err := ValidateYtDlpBinary(binaryPaths[2]); err != nil {
				return fmt.Errorf("streaming yt-dlp validation failed: %w", err)
			}
		}
	}

	if len(binaryPaths) > 3 {
		// Streaming ffmpeg path (fourth parameter)
		if binaryPaths[3] != "" && binaryPaths[3] != binaryPaths[0] {
			if err := ValidateFFmpegBinary(binaryPaths[3]); err != nil {
				return fmt.Errorf("streaming FFmpeg validation failed: %w", err)
			}
		}
	}

	return nil
}
