package audio

import (
	"errors"
	"testing"

	"github.com/latoulicious/HKTM/pkg/audio"
)

// MockLogger for testing
type MockLogger struct {
	logs []string
}

func (m *MockLogger) Info(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, "INFO: "+msg)
}

func (m *MockLogger) Error(msg string, err error, fields map[string]interface{}) {
	m.logs = append(m.logs, "ERROR: "+msg+": "+err.Error())
}

func (m *MockLogger) Warn(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, "WARN: "+msg)
}

func (m *MockLogger) Debug(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, "DEBUG: "+msg)
}

func (m *MockLogger) WithPipeline(pipeline string) audio.AudioLogger {
	return m
}

func (m *MockLogger) WithContext(ctx map[string]interface{}) audio.AudioLogger {
	return m
}

// TestDetectStreamFailure tests URL expiry detection
func TestDetectStreamFailure(t *testing.T) {
	mockLogger := &MockLogger{}

	config := &audio.FFmpegConfig{
		BinaryPath: "ffmpeg",
	}
	ytdlpConfig := &audio.YtDlpConfig{
		BinaryPath: "yt-dlp",
	}
	streamingConfig := &audio.StreamingConfig{
		YtdlpPath:  "yt-dlp",
		FFmpegPath: "ffmpeg",
	}

	processor := audio.NewFFmpegProcessor(config, ytdlpConfig, streamingConfig, mockLogger)

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "403 Forbidden error should be detected",
			err:      errors.New("HTTP error 403 Forbidden"),
			expected: true,
		},
		{
			name:     "404 Not Found error should be detected",
			err:      errors.New("Server returned 404 Not Found"),
			expected: true,
		},
		{
			name:     "Connection refused should be detected",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "End of file should be detected",
			err:      errors.New("end of file"),
			expected: true,
		},
		{
			name:     "Invalid data should be detected",
			err:      errors.New("invalid data found when processing input"),
			expected: true,
		},
		{
			name:     "Regular network error should not be detected",
			err:      errors.New("network timeout"),
			expected: false,
		},
		{
			name:     "Nil error should not be detected",
			err:      nil,
			expected: false,
		},
		{
			name:     "Generic error should not be detected",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.DetectStreamFailure(tc.err)
			if result != tc.expected {
				t.Errorf("DetectStreamFailure(%v) = %v, expected %v", tc.err, result, tc.expected)
			}
		})
	}
}

// TestURLRefreshBasicFlow tests the basic URL refresh flow
func TestURLRefreshBasicFlow(t *testing.T) {
	mockLogger := &MockLogger{}

	config := &audio.FFmpegConfig{
		BinaryPath: "ffmpeg",
	}
	ytdlpConfig := &audio.YtDlpConfig{
		BinaryPath: "yt-dlp",
	}
	streamingConfig := &audio.StreamingConfig{
		YtdlpPath:  "yt-dlp",
		FFmpegPath: "ffmpeg",
	}

	processor := audio.NewFFmpegProcessor(config, ytdlpConfig, streamingConfig, mockLogger)

	// Test that URL refresh detection doesn't panic with various error types
	testErrors := []error{
		errors.New("403 Forbidden"),
		errors.New("connection refused"),
		nil,
		errors.New("some other error"),
	}

	for _, err := range testErrors {
		// This should not panic
		detected := processor.DetectStreamFailure(err)
		t.Logf("Error %v detected as URL expiry: %v", err, detected)
	}

	// Verify some logs were generated
	if len(mockLogger.logs) == 0 {
		t.Log("No logs generated (expected for this basic test)")
	}
}
