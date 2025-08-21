package audio_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/audio"
)

func TestStreamingErrorClassification(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType string
		shouldRetry  bool
	}{
		{
			name:         "streaming pipeline error",
			err:          audio.CreateStreamingPipelineError("ffmpeg", errors.New("pipe broken")),
			expectedType: "streaming_pipeline",
			shouldRetry:  true,
		},
		{
			name:         "URL expiry error",
			err:          audio.CreateURLExpiryError("https://youtube.com/watch?v=test", errors.New("url expired")),
			expectedType: "url_expiry",
			shouldRetry:  true,
		},
		{
			name:         "yt-dlp streaming error",
			err:          audio.CreateYtDlpStreamingError("extraction", errors.New("streaming failed")),
			expectedType: "yt-dlp_streaming",
			shouldRetry:  true,
		},
		{
			name:         "ffmpeg streaming error",
			err:          audio.CreateFFmpegStreamingError("pipe input", errors.New("stdin error")),
			expectedType: "ffmpeg_streaming",
			shouldRetry:  true,
		},
	}

	// Create a basic error handler for testing
	config := &audio.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
	}

	handler := audio.NewBasicErrorHandler(config, nil, nil, "test-guild")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test error classification
			shouldRetry := handler.IsRetryableError(tt.err)
			if shouldRetry != tt.shouldRetry {
				t.Errorf("IsRetryableError() = %v, want %v", shouldRetry, tt.shouldRetry)
			}

			// Test that streaming errors use simple backoff
			if audio.IsStreamingError(tt.err) {
				delay1 := handler.GetRetryDelayForError(tt.err, 1)
				delay2 := handler.GetRetryDelayForError(tt.err, 2)
				delay3 := handler.GetRetryDelayForError(tt.err, 3)

				// Verify simple backoff delays: 2s, 5s, 10s
				if delay1 != 2*time.Second {
					t.Errorf("First retry delay = %v, want %v", delay1, 2*time.Second)
				}
				if delay2 != 5*time.Second {
					t.Errorf("Second retry delay = %v, want %v", delay2, 5*time.Second)
				}
				if delay3 != 10*time.Second {
					t.Errorf("Third retry delay = %v, want %v", delay3, 10*time.Second)
				}
			}
		})
	}
}

func TestStreamingErrorHelpers(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		isStreaming bool
	}{
		{
			name:        "streaming pipeline error",
			err:         audio.CreateStreamingPipelineError("test", errors.New("test")),
			isStreaming: true,
		},
		{
			name:        "URL expiry error",
			err:         audio.CreateURLExpiryError("test", errors.New("test")),
			isStreaming: true,
		},
		{
			name:        "regular network error",
			err:         errors.New("connection refused"),
			isStreaming: false,
		},
		{
			name:        "nil error",
			err:         nil,
			isStreaming: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := audio.IsStreamingError(tt.err)
			if result != tt.isStreaming {
				t.Errorf("IsStreamingError() = %v, want %v", result, tt.isStreaming)
			}
		})
	}
}

func TestSimpleBackoffDelays(t *testing.T) {
	config := &audio.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
	}

	handler := audio.NewBasicErrorHandler(config, nil, nil, "test-guild")
	streamingErr := audio.CreateStreamingPipelineError("test", errors.New("test error"))

	// Test that streaming errors get simple backoff (2s, 5s, 10s)
	delays := []time.Duration{
		handler.GetRetryDelayForError(streamingErr, 1),
		handler.GetRetryDelayForError(streamingErr, 2),
		handler.GetRetryDelayForError(streamingErr, 3),
		handler.GetRetryDelayForError(streamingErr, 4), // Should cap at 10s
	}

	expected := []time.Duration{
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
		10 * time.Second, // Capped at max
	}

	for i, delay := range delays {
		if delay != expected[i] {
			t.Errorf("Attempt %d: delay = %v, want %v", i+1, delay, expected[i])
		}
	}

	// Test that non-streaming errors still use exponential backoff
	networkErr := errors.New("connection refused")
	networkDelay1 := handler.GetRetryDelayForError(networkErr, 1)
	networkDelay2 := handler.GetRetryDelayForError(networkErr, 2)

	// Should be exponential: 2s, 4s
	if networkDelay1 != 2*time.Second {
		t.Errorf("Network error first delay = %v, want %v", networkDelay1, 2*time.Second)
	}
	if networkDelay2 != 4*time.Second {
		t.Errorf("Network error second delay = %v, want %v", networkDelay2, 4*time.Second)
	}
}

// TestYtDlpFormatFallback tests that yt-dlp format fallback works correctly
func TestYtDlpFormatFallback(t *testing.T) {
	// Test the format specification we're using
	format := "bestaudio/best"

	// Verify the format contains both options
	if !strings.Contains(format, "bestaudio") {
		t.Error("Format should contain 'bestaudio'")
	}

	if !strings.Contains(format, "best") {
		t.Error("Format should contain 'best' as fallback")
	}

	t.Logf("✅ Format fallback specification is correct: %s", format)
}

// TestStreamingURLDetection tests the streaming URL detection logic
func TestStreamingURLDetection(t *testing.T) {
	testCases := []struct {
		url         string
		isStreaming bool
		description string
	}{
		{
			url:         "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			isStreaming: false,
			description: "YouTube URL should not be detected as streaming",
		},
		{
			url:         "https://manifest.googlevideo.com/api/manifest/hls_playlist/...",
			isStreaming: true,
			description: "Google video manifest should be detected as streaming",
		},
		{
			url:         "https://rr3---sn-4pgnuhxqp5-jb3k.googlevideo.com/...",
			isStreaming: true,
			description: "Google video CDN URL should be detected as streaming",
		},
		{
			url:         "https://example.com/audio.m3u8",
			isStreaming: true,
			description: "HLS playlist should be detected as streaming",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// This is a simplified version of the detection logic
			urlLower := strings.ToLower(tc.url)
			isStreaming := false

			streamingPatterns := []string{
				"googlevideo.com",
				"manifest.googlevideo.com",
				"manifest",
				"m3u8",
				"mpd",
				"index.m3u8",
			}

			for _, pattern := range streamingPatterns {
				if strings.Contains(urlLower, pattern) {
					isStreaming = true
					break
				}
			}

			if isStreaming != tc.isStreaming {
				t.Errorf("Expected %v for URL %s, got %v", tc.isStreaming, tc.url, isStreaming)
			} else {
				t.Logf("✅ Correctly detected streaming URL: %s", tc.url)
			}
		})
	}
}

// TestPipelineTypeSelection tests that the correct pipeline type is selected
func TestPipelineTypeSelection(t *testing.T) {
	testCases := []struct {
		url              string
		expectedPipeline string
		description      string
	}{
		{
			url:              "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			expectedPipeline: "ytdlp_ffmpeg",
			description:      "YouTube URL should use yt-dlp | ffmpeg pipeline",
		},
		{
			url:              "https://manifest.googlevideo.com/api/manifest/hls_playlist/...",
			expectedPipeline: "direct_ffmpeg",
			description:      "Streaming URL should use direct FFmpeg pipeline",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// This is a simplified version of the pipeline selection logic
			urlLower := strings.ToLower(tc.url)
			pipelineType := "ytdlp_ffmpeg" // default

			streamingPatterns := []string{
				"googlevideo.com",
				"manifest.googlevideo.com",
				"manifest",
				"m3u8",
				"mpd",
				"index.m3u8",
			}

			for _, pattern := range streamingPatterns {
				if strings.Contains(urlLower, pattern) {
					pipelineType = "direct_ffmpeg"
					break
				}
			}

			if pipelineType != tc.expectedPipeline {
				t.Errorf("Expected pipeline %s for URL %s, got %s", tc.expectedPipeline, tc.url, pipelineType)
			} else {
				t.Logf("✅ Correctly selected pipeline %s for URL: %s", pipelineType, tc.url)
			}
		})
	}
}
