package audio

import (
	"fmt"
	"net/url"
	"strings"
	"time"
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
