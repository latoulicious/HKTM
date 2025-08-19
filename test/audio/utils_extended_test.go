package audio_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/latoulicious/HKTM/pkg/audio"
)

// Test CreateAudioMetric utility function

func TestCreateAudioMetric(t *testing.T) {
	guildID := "test-guild-123"
	metricType := "startup_time"
	value := 2.5
	
	metric := audio.CreateAudioMetric(guildID, metricType, value)
	
	// Verify all fields are set correctly
	if metric.GuildID != guildID {
		t.Errorf("Expected guild ID %s, got %s", guildID, metric.GuildID)
	}
	
	if metric.MetricType != metricType {
		t.Errorf("Expected metric type %s, got %s", metricType, metric.MetricType)
	}
	
	if metric.Value != value {
		t.Errorf("Expected value %f, got %f", value, metric.Value)
	}
	
	// Verify ID is generated
	if metric.ID == uuid.Nil {
		t.Error("Expected non-nil UUID for metric ID")
	}
	
	// Verify timestamp is recent (within last second)
	if time.Since(metric.Timestamp) > time.Second {
		t.Error("Expected timestamp to be recent")
	}
}

func TestCreateAudioMetric_DifferentTypes(t *testing.T) {
	guildID := "test-guild-123"
	
	tests := []struct {
		name       string
		metricType string
		value      float64
	}{
		{
			name:       "startup time",
			metricType: "startup_time",
			value:      1.5,
		},
		{
			name:       "error count",
			metricType: "error_count",
			value:      1.0,
		},
		{
			name:       "playback duration",
			metricType: "playback_duration",
			value:      180.0,
		},
		{
			name:       "zero value",
			metricType: "test_metric",
			value:      0.0,
		},
		{
			name:       "negative value",
			metricType: "test_metric",
			value:      -1.0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric := audio.CreateAudioMetric(guildID, tt.metricType, tt.value)
			
			if metric.MetricType != tt.metricType {
				t.Errorf("Expected metric type %s, got %s", tt.metricType, metric.MetricType)
			}
			
			if metric.Value != tt.value {
				t.Errorf("Expected value %f, got %f", tt.value, metric.Value)
			}
		})
	}
}

// Test CreateAudioLog utility function

func TestCreateAudioLog(t *testing.T) {
	guildID := "test-guild-123"
	level := "INFO"
	message := "Test log message"
	errorMsg := "Test error message"
	fields := map[string]interface{}{
		"url":      "https://youtube.com/test",
		"duration": 180,
	}
	
	log := audio.CreateAudioLog(guildID, level, message, errorMsg, fields)
	
	// Verify all fields are set correctly
	if log.GuildID != guildID {
		t.Errorf("Expected guild ID %s, got %s", guildID, log.GuildID)
	}
	
	if log.Level != level {
		t.Errorf("Expected level %s, got %s", level, log.Level)
	}
	
	if log.Message != message {
		t.Errorf("Expected message %s, got %s", message, log.Message)
	}
	
	if log.Error != errorMsg {
		t.Errorf("Expected error %s, got %s", errorMsg, log.Error)
	}
	
	// Verify fields are preserved
	if log.Fields["url"] != "https://youtube.com/test" {
		t.Error("Expected fields to be preserved")
	}
	
	if log.Fields["duration"] != 180 {
		t.Error("Expected fields to be preserved")
	}
	
	// Verify ID is generated
	if log.ID == uuid.Nil {
		t.Error("Expected non-nil UUID for log ID")
	}
	
	// Verify timestamp is recent
	if time.Since(log.Timestamp) > time.Second {
		t.Error("Expected timestamp to be recent")
	}
}

func TestCreateAudioLog_EmptyFields(t *testing.T) {
	guildID := "test-guild-123"
	level := "ERROR"
	message := "Test message"
	errorMsg := ""
	
	// Test with nil fields
	log := audio.CreateAudioLog(guildID, level, message, errorMsg, nil)
	
	if log.Fields != nil {
		t.Error("Expected fields to be nil when nil is passed")
	}
	
	// Test with empty fields map
	emptyFields := make(map[string]interface{})
	log = audio.CreateAudioLog(guildID, level, message, errorMsg, emptyFields)
	
	if len(log.Fields) != 0 {
		t.Error("Expected fields to be empty when empty map is passed")
	}
}

// Test CreateAudioError utility function

func TestCreateAudioError(t *testing.T) {
	guildID := "test-guild-123"
	errorType := "network"
	errorMsg := "Connection timeout"
	context := "ffmpeg_process"
	
	audioError := audio.CreateAudioError(guildID, errorType, errorMsg, context)
	
	// Verify all fields are set correctly
	if audioError.GuildID != guildID {
		t.Errorf("Expected guild ID %s, got %s", guildID, audioError.GuildID)
	}
	
	if audioError.ErrorType != errorType {
		t.Errorf("Expected error type %s, got %s", errorType, audioError.ErrorType)
	}
	
	if audioError.ErrorMsg != errorMsg {
		t.Errorf("Expected error message %s, got %s", errorMsg, audioError.ErrorMsg)
	}
	
	if audioError.Context != context {
		t.Errorf("Expected context %s, got %s", context, audioError.Context)
	}
	
	// Verify ID is generated
	if audioError.ID == uuid.Nil {
		t.Error("Expected non-nil UUID for error ID")
	}
	
	// Verify timestamp is recent
	if time.Since(audioError.Timestamp) > time.Second {
		t.Error("Expected timestamp to be recent")
	}
	
	// Verify resolved is false by default
	if audioError.Resolved {
		t.Error("Expected resolved to be false by default")
	}
}

func TestCreateAudioError_DifferentTypes(t *testing.T) {
	guildID := "test-guild-123"
	
	tests := []struct {
		name      string
		errorType string
		errorMsg  string
		context   string
	}{
		{
			name:      "network error",
			errorType: "network",
			errorMsg:  "Connection timeout",
			context:   "stream_start",
		},
		{
			name:      "ffmpeg error",
			errorType: "ffmpeg",
			errorMsg:  "Process crashed",
			context:   "encoding",
		},
		{
			name:      "discord error",
			errorType: "discord",
			errorMsg:  "Voice connection lost",
			context:   "voice_connection",
		},
		{
			name:      "empty context",
			errorType: "system",
			errorMsg:  "Unknown error",
			context:   "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audioError := audio.CreateAudioError(guildID, tt.errorType, tt.errorMsg, tt.context)
			
			if audioError.ErrorType != tt.errorType {
				t.Errorf("Expected error type %s, got %s", tt.errorType, audioError.ErrorType)
			}
			
			if audioError.ErrorMsg != tt.errorMsg {
				t.Errorf("Expected error message %s, got %s", tt.errorMsg, audioError.ErrorMsg)
			}
			
			if audioError.Context != tt.context {
				t.Errorf("Expected context %s, got %s", tt.context, audioError.Context)
			}
		})
	}
}

// Test ValidateBinaryDependency utility function

func TestValidateBinaryDependency(t *testing.T) {
	tests := []struct {
		name       string
		binaryName string
		binaryPath string
		wantErr    bool
	}{
		{
			name:       "empty binary path",
			binaryName: "ffmpeg",
			binaryPath: "",
			wantErr:    true,
		},
		{
			name:       "valid system binary",
			binaryName: "ls", // Should exist on most Unix systems
			binaryPath: "ls",
			wantErr:    false,
		},
		{
			name:       "invalid binary path",
			binaryName: "nonexistent",
			binaryPath: "nonexistent-binary-12345",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := audio.ValidateBinaryDependency(tt.binaryName, tt.binaryPath)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBinaryDependency() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err != nil && tt.wantErr {
				// Verify error message contains binary name
				if !strings.Contains(err.Error(), tt.binaryName) {
					t.Errorf("Error message should contain binary name %s: %v", tt.binaryName, err)
				}
			}
		})
	}
}

func TestValidateFFmpegBinary(t *testing.T) {
	tests := []struct {
		name       string
		binaryPath string
		wantErr    bool
	}{
		{
			name:       "empty path",
			binaryPath: "",
			wantErr:    true,
		},
		{
			name:       "nonexistent binary",
			binaryPath: "nonexistent-ffmpeg-12345",
			wantErr:    true,
		},
		// Note: We can't test with real ffmpeg as it may not be installed in test environment
		// In a real test environment, you might want to mock the exec.Command or skip if ffmpeg not available
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := audio.ValidateFFmpegBinary(tt.binaryPath)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFFmpegBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err != nil && tt.wantErr {
				// Verify error message mentions ffmpeg
				if !strings.Contains(strings.ToLower(err.Error()), "ffmpeg") {
					t.Errorf("Error message should mention ffmpeg: %v", err)
				}
			}
		})
	}
}

func TestValidateYtDlpBinary(t *testing.T) {
	tests := []struct {
		name       string
		binaryPath string
		wantErr    bool
	}{
		{
			name:       "empty path",
			binaryPath: "",
			wantErr:    true,
		},
		{
			name:       "nonexistent binary",
			binaryPath: "nonexistent-yt-dlp-12345",
			wantErr:    true,
		},
		// Note: We can't test with real yt-dlp as it may not be installed in test environment
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := audio.ValidateYtDlpBinary(tt.binaryPath)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateYtDlpBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err != nil && tt.wantErr {
				// Verify error message mentions yt-dlp
				if !strings.Contains(strings.ToLower(err.Error()), "yt-dlp") {
					t.Errorf("Error message should mention yt-dlp: %v", err)
				}
			}
		})
	}
}

func TestValidateAllBinaryDependencies(t *testing.T) {
	tests := []struct {
		name       string
		ffmpegPath string
		ytDlpPath  string
		wantErr    bool
	}{
		{
			name:       "both empty",
			ffmpegPath: "",
			ytDlpPath:  "",
			wantErr:    true,
		},
		{
			name:       "ffmpeg empty",
			ffmpegPath: "",
			ytDlpPath:  "yt-dlp",
			wantErr:    true,
		},
		{
			name:       "yt-dlp empty",
			ffmpegPath: "ffmpeg",
			ytDlpPath:  "",
			wantErr:    true,
		},
		{
			name:       "both nonexistent",
			ffmpegPath: "nonexistent-ffmpeg",
			ytDlpPath:  "nonexistent-yt-dlp",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := audio.ValidateAllBinaryDependencies(tt.ffmpegPath, tt.ytDlpPath)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAllBinaryDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test edge cases and error conditions

func TestCreateContextFields_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		guildID string
		userID  string
		url     string
	}{
		{
			name:    "very long guild ID",
			guildID: strings.Repeat("a", 1000),
			userID:  "user123",
			url:     "https://youtube.com/test",
		},
		{
			name:    "special characters in URL",
			guildID: "guild123",
			userID:  "user123",
			url:     "https://youtube.com/watch?v=test&list=playlist&t=30s",
		},
		{
			name:    "unicode characters",
			guildID: "guild123",
			userID:  "user123",
			url:     "https://youtube.com/watch?v=测试",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic with any input
			fields := audio.CreateContextFields(tt.guildID, tt.userID, tt.url)
			
			// Verify timestamp is always present
			if _, exists := fields["timestamp"]; !exists {
				t.Error("Timestamp should always be present")
			}
			
			// Verify non-empty fields are included
			if tt.guildID != "" && fields["guild_id"] != tt.guildID {
				t.Error("Guild ID should be preserved")
			}
			
			if tt.userID != "" && fields["user_id"] != tt.userID {
				t.Error("User ID should be preserved")
			}
			
			if tt.url != "" && fields["url"] != tt.url {
				t.Error("URL should be preserved")
			}
		})
	}
}

func TestCreateContextFieldsWithComponent_EdgeCases(t *testing.T) {
	// Test with empty component
	fields := audio.CreateContextFieldsWithComponent("guild123", "user456", "url", "")
	
	// Component should not be included if empty
	if _, exists := fields["component"]; exists {
		t.Error("Component should not be included when empty")
	}
	
	// Other fields should still be present
	if fields["guild_id"] != "guild123" {
		t.Error("Guild ID should be present")
	}
	
	// Test with very long component name
	longComponent := strings.Repeat("component", 100)
	fields = audio.CreateContextFieldsWithComponent("guild123", "", "", longComponent)
	
	if fields["component"] != longComponent {
		t.Error("Long component name should be preserved")
	}
}

func TestFormatDuration_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "negative duration",
			duration: -5 * time.Second,
			expected: "-5.0s",
		},
		{
			name:     "very small duration",
			duration: 1 * time.Nanosecond,
			expected: "0.0s",
		},
		{
			name:     "very large duration",
			duration: 25*time.Hour + 30*time.Minute + 45*time.Second,
			expected: "25h 30m 45s",
		},
		{
			name:     "fractional seconds",
			duration: 1500 * time.Millisecond,
			expected: "1.5s",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := audio.FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration() = %v, expected %v", result, tt.expected)
			}
		})
	}
}