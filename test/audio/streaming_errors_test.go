package audio_test

import (
	"errors"
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
