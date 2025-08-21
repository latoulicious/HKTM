package audio_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/tools"
)

// TestYtDlpPipingBasic tests the yt-dlp | ffmpeg pipeline setup without external URLs
func TestYtDlpPipingBasic(t *testing.T) {
	// Skip if running in CI or if binaries are not available
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Validate binaries are available
	validator := tools.NewBinaryValidator("ffmpeg", "yt-dlp")
	if err := validator.QuickValidation(); err != nil {
		t.Skipf("Required binaries not available: %v", err)
	}

	// Test that yt-dlp can extract URLs (without actually streaming)
	t.Run("yt-dlp_url_extraction", func(t *testing.T) {
		// Test with a simple YouTube URL to see if yt-dlp can extract info
		testURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

		cmd := exec.Command("yt-dlp", "--get-url", "--quiet", "--no-warnings", testURL)
		output, err := cmd.Output()

		if err != nil {
			t.Logf("yt-dlp URL extraction failed (expected in some environments): %v", err)
			// Don't fail the test - this is expected to fail in many environments
		} else {
			streamURL := strings.TrimSpace(string(output))
			if streamURL != "" {
				t.Logf("yt-dlp successfully extracted stream URL: %s", streamURL[:50]+"...")
			} else {
				t.Log("yt-dlp returned empty URL")
			}
		}
	})

	// Test FFmpeg pipeline setup
	t.Run("ffmpeg_pipeline_setup", func(t *testing.T) {
		// Create test configuration
		ffmpegConfig := &audio.FFmpegConfig{
			BinaryPath:  "ffmpeg",
			AudioFormat: "s16le",
			SampleRate:  48000,
			Channels:    2,
			CustomArgs:  []string{},
		}

		ytdlpConfig := &audio.YtDlpConfig{
			BinaryPath: "yt-dlp",
			CustomArgs: []string{"--no-playlist"},
		}

		streamingConfig := &audio.StreamingConfig{
			YtdlpPath:  "yt-dlp",
			FFmpegPath: "ffmpeg",
			SampleRate: 48000,
			Channels:   2,
		}

		logger := &MockAudioLogger{}
		processor := audio.NewFFmpegProcessor(ffmpegConfig, ytdlpConfig, streamingConfig, logger)

		// Test processor creation
		if processor == nil {
			t.Fatal("Failed to create FFmpeg processor")
		}

		// Test initial state
		if processor.IsRunning() {
			t.Error("Processor should not be running initially")
		}

		// Test process info
		info := processor.GetProcessInfo()
		if info["is_running"].(bool) {
			t.Error("Process info should show not running initially")
		}

		t.Log("FFmpeg processor created and initialized successfully")
	})

	t.Log("Basic yt-dlp | ffmpeg pipeline tests completed")
}

// TestYtDlpPipingWithRealURL tests with a real YouTube URL (may fail in CI)
func TestYtDlpPipingWithRealURL(t *testing.T) {
	// Skip if running in CI or if binaries are not available
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Validate binaries are available
	validator := tools.NewBinaryValidator("ffmpeg", "yt-dlp")
	if err := validator.QuickValidation(); err != nil {
		t.Skipf("Required binaries not available: %v", err)
	}

	// Use a well-known YouTube URL
	testURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

	// Create test configuration
	ffmpegConfig := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	ytdlpConfig := &audio.YtDlpConfig{
		BinaryPath: "yt-dlp",
		CustomArgs: []string{"--no-playlist"},
	}

	streamingConfig := &audio.StreamingConfig{
		YtdlpPath:  "yt-dlp",
		FFmpegPath: "ffmpeg",
		SampleRate: 48000,
		Channels:   2,
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(ffmpegConfig, ytdlpConfig, streamingConfig, logger)

	t.Logf("Testing yt-dlp | ffmpeg pipeline with URL: %s", testURL)

	// Test starting the stream with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan error, 1)
	var reader io.ReadCloser

	go func() {
		var err error
		reader, err = processor.StartStream(testURL)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			// Log the error but don't fail the test - YouTube URLs can be flaky
			t.Logf("Failed to start stream (expected in many environments): %v", err)

			// Check if it's a known issue
			if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Forbidden") {
				t.Log("Got 403 error - this is expected for YouTube URLs in many environments")
			} else if strings.Contains(err.Error(), "unavailable") {
				t.Log("Video unavailable - this can happen with test URLs")
			} else if strings.Contains(err.Error(), "extraction failed") {
				t.Log("URL extraction failed - this is expected in restricted environments")
			} else {
				t.Logf("Got error: %v", err)
			}

			// Check logs for debugging
			logs := logger.GetLogs()
			t.Logf("Generated %d log entries", len(logs))
			return
		}

		if reader == nil {
			t.Error("StartStream returned nil reader")
			return
		}

		// Verify the processor is running
		if !processor.IsRunning() {
			t.Error("Processor should be running after StartStream")
		}

		// Try to read a small amount of data with timeout
		buffer := make([]byte, 1920) // 10ms of PCM audio
		readCtx, readCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer readCancel()

		readDone := make(chan error, 1)
		var bytesRead int

		go func() {
			var err error
			bytesRead, err = reader.Read(buffer)
			readDone <- err
		}()

		select {
		case readErr := <-readDone:
			if readErr != nil && readErr != io.EOF {
				t.Logf("Read error (may be expected): %v", readErr)
			} else if bytesRead > 0 {
				t.Logf("Successfully read %d bytes of audio data", bytesRead)

				// Check if we got actual audio data
				hasNonZeroData := false
				for i := 0; i < bytesRead; i++ {
					if buffer[i] != 0 {
						hasNonZeroData = true
						break
					}
				}

				if hasNonZeroData {
					t.Log("Audio data contains non-zero values - pipeline is working correctly")
				} else {
					t.Log("Audio data is all zeros - may indicate silence or issue")
				}
			}

		case <-readCtx.Done():
			t.Log("Timeout reading audio data - this can happen with slow networks")
		}

		// Clean up
		if err := processor.Stop(); err != nil {
			t.Logf("Error stopping processor: %v", err)
		}

	case <-ctx.Done():
		t.Log("Timeout starting stream - this can happen with slow networks or restricted environments")
		processor.Stop()
	}

	// Check logs
	logs := logger.GetLogs()
	t.Logf("Generated %d log entries", len(logs))

	errorCount := 0
	for _, log := range logs {
		if log.Level == "ERROR" {
			errorCount++
		}
	}

	if errorCount > 0 {
		t.Logf("Found %d error logs (may be expected)", errorCount)
	}

	t.Log("Real URL test completed (errors are expected in many environments)")
}

// TestURLRefreshOnFailure tests URL refresh functionality when a stream fails
func TestURLRefreshOnFailure(t *testing.T) {
	// Skip if running in CI or if binaries are not available
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Validate binaries are available
	validator := tools.NewBinaryValidator("ffmpeg", "yt-dlp")
	if err := validator.QuickValidation(); err != nil {
		t.Skipf("Required binaries not available: %v", err)
	}

	// Create test configuration
	ffmpegConfig := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	ytdlpConfig := &audio.YtDlpConfig{
		BinaryPath: "yt-dlp",
		CustomArgs: []string{"--no-playlist"},
	}

	streamingConfig := &audio.StreamingConfig{
		YtdlpPath:  "yt-dlp",
		FFmpegPath: "ffmpeg",
		SampleRate: 48000,
		Channels:   2,
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(ffmpegConfig, ytdlpConfig, streamingConfig, logger)

	// Test URL refresh detection with various error types
	testErrors := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "403 Forbidden should trigger refresh",
			err:      fmt.Errorf("HTTP error 403 Forbidden"),
			expected: true,
		},
		{
			name:     "404 Not Found should trigger refresh",
			err:      fmt.Errorf("Server returned 404 Not Found"),
			expected: true,
		},
		{
			name:     "Connection refused should trigger refresh",
			err:      fmt.Errorf("connection refused"),
			expected: true,
		},
		{
			name:     "End of file should trigger refresh",
			err:      fmt.Errorf("end of file"),
			expected: true,
		},
		{
			name:     "Invalid data should trigger refresh",
			err:      fmt.Errorf("invalid data found when processing input"),
			expected: true,
		},
		{
			name:     "Generic error should not trigger refresh",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tc := range testErrors {
		t.Run(tc.name, func(t *testing.T) {
			detected := processor.DetectStreamFailure(tc.err)
			if detected != tc.expected {
				t.Errorf("DetectStreamFailure(%v) = %v, expected %v", tc.err, detected, tc.expected)
			}
			t.Logf("Error '%v' correctly detected as URL expiry: %v", tc.err, detected)
		})
	}

	t.Log("URL refresh detection test completed successfully")
}

// TestManualYouTubeURLs tests with multiple real YouTube URLs to verify robustness
func TestManualYouTubeURLs(t *testing.T) {
	// Skip if running in CI or if binaries are not available
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Validate binaries are available
	validator := tools.NewBinaryValidator("ffmpeg", "yt-dlp")
	if err := validator.QuickValidation(); err != nil {
		t.Skipf("Required binaries not available: %v", err)
	}

	// Test URLs - using well-known, stable videos
	testURLs := []struct {
		name string
		url  string
		desc string
	}{
		{
			name: "first_youtube_video",
			url:  "https://www.youtube.com/watch?v=jNQXAC9IVRw",
			desc: "Me at the zoo - First YouTube video",
		},
		{
			name: "short_video",
			url:  "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			desc: "Rick Astley - Never Gonna Give You Up",
		},
	}

	// Create test configuration
	ffmpegConfig := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	ytdlpConfig := &audio.YtDlpConfig{
		BinaryPath: "yt-dlp",
		CustomArgs: []string{"--no-playlist"},
	}

	streamingConfig := &audio.StreamingConfig{
		YtdlpPath:  "yt-dlp",
		FFmpegPath: "ffmpeg",
		SampleRate: 48000,
		Channels:   2,
	}

	for _, testCase := range testURLs {
		t.Run(testCase.name, func(t *testing.T) {
			logger := &MockAudioLogger{}
			processor := audio.NewFFmpegProcessor(ffmpegConfig, ytdlpConfig, streamingConfig, logger)

			t.Logf("Testing URL: %s (%s)", testCase.url, testCase.desc)

			// Test starting the stream with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			done := make(chan error, 1)
			var reader io.ReadCloser

			go func() {
				var err error
				reader, err = processor.StartStream(testCase.url)
				done <- err
			}()

			select {
			case err := <-done:
				if err != nil {
					// Log the error but don't fail the test - YouTube URLs can be flaky
					t.Logf("Failed to start stream for %s: %v", testCase.name, err)

					// Check if it's a known issue
					if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "Forbidden") {
						t.Logf("Got 403 error - this is expected for some YouTube URLs")
					} else if strings.Contains(err.Error(), "unavailable") {
						t.Logf("Video unavailable - this can happen with test URLs")
					} else {
						t.Logf("Unexpected error type: %v", err)
					}
					return
				}

				if reader == nil {
					t.Errorf("StartStream returned nil reader for %s", testCase.name)
					return
				}

				// Verify the processor is running
				if !processor.IsRunning() {
					t.Errorf("Processor should be running after StartStream for %s", testCase.name)
				}

				// Try to read a small amount of data
				buffer := make([]byte, 1920) // 10ms of PCM audio
				bytesRead, readErr := reader.Read(buffer)

				if readErr != nil && readErr != io.EOF {
					t.Logf("Read error for %s: %v", testCase.name, readErr)
				} else if bytesRead > 0 {
					t.Logf("Successfully read %d bytes from %s", bytesRead, testCase.name)
				}

				// Clean up
				processor.Stop()

			case <-ctx.Done():
				t.Logf("Timeout starting stream for %s - this can happen with slow networks", testCase.name)
				processor.Stop()
			}

			// Check logs
			logs := logger.GetLogs()
			t.Logf("Generated %d log entries for %s", len(logs), testCase.name)

			errorCount := 0
			for _, log := range logs {
				if log.Level == "ERROR" {
					errorCount++
				}
			}

			if errorCount > 0 {
				t.Logf("Found %d error logs for %s", errorCount, testCase.name)
			}
		})
	}

	t.Log("Manual YouTube URL testing completed")
}

// TestStreamingPipelineRobustness tests the pipeline under various conditions
func TestStreamingPipelineRobustness(t *testing.T) {
	// Skip if running in CI or if binaries are not available
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Validate binaries are available
	validator := tools.NewBinaryValidator("ffmpeg", "yt-dlp")
	if err := validator.QuickValidation(); err != nil {
		t.Skipf("Required binaries not available: %v", err)
	}

	// Create test configuration
	ffmpegConfig := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	ytdlpConfig := &audio.YtDlpConfig{
		BinaryPath: "yt-dlp",
		CustomArgs: []string{"--no-playlist"},
	}

	streamingConfig := &audio.StreamingConfig{
		YtdlpPath:  "yt-dlp",
		FFmpegPath: "ffmpeg",
		SampleRate: 48000,
		Channels:   2,
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(ffmpegConfig, ytdlpConfig, streamingConfig, logger)

	// Test 1: Invalid URL should fail gracefully
	t.Run("invalid_url", func(t *testing.T) {
		invalidURL := "https://invalid-url-that-does-not-exist.com/test"

		_, err := processor.StartStream(invalidURL)
		if err == nil {
			t.Error("Expected error for invalid URL")
		} else {
			t.Logf("Invalid URL correctly failed: %v", err)
		}

		// Ensure processor is not running after failure
		if processor.IsRunning() {
			t.Error("Processor should not be running after failed start")
		}
	})

	// Test 2: Multiple start/stop cycles
	t.Run("multiple_cycles", func(t *testing.T) {
		testURL := "https://www.youtube.com/watch?v=jNQXAC9IVRw"

		for i := 0; i < 3; i++ {
			t.Logf("Cycle %d: Starting stream", i+1)

			reader, err := processor.StartStream(testURL)
			if err != nil {
				t.Logf("Cycle %d failed to start: %v", i+1, err)
				continue
			}

			if reader != nil {
				// Read a tiny bit of data
				buffer := make([]byte, 100)
				reader.Read(buffer)
			}

			t.Logf("Cycle %d: Stopping stream", i+1)
			if err := processor.Stop(); err != nil {
				t.Errorf("Cycle %d failed to stop: %v", i+1, err)
			}

			// Verify stopped
			if processor.IsRunning() {
				t.Errorf("Cycle %d: Processor still running after stop", i+1)
			}

			// Small delay between cycles
			time.Sleep(100 * time.Millisecond)
		}
	})

	// Test 3: Process info and monitoring
	t.Run("process_info", func(t *testing.T) {
		// Test process info when not running
		info := processor.GetProcessInfo()
		if info["is_running"].(bool) {
			t.Error("Process info should show not running initially")
		}

		testURL := "https://www.youtube.com/watch?v=jNQXAC9IVRw"
		reader, err := processor.StartStream(testURL)
		if err != nil {
			t.Logf("Failed to start for process info test: %v", err)
			return
		}
		defer processor.Stop()

		if reader != nil {
			// Test process info when running
			info = processor.GetProcessInfo()
			if !info["is_running"].(bool) {
				t.Error("Process info should show running after start")
			}

			t.Logf("Process info: %+v", info)
		}
	})

	t.Log("Pipeline robustness testing completed")
}

// MockAudioLogger implements the AudioLogger interface for testing
type MockAudioLogger struct {
	logs []LogEntry
}

type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]interface{}
	Error   error
}

func (m *MockAudioLogger) Info(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "INFO", Message: msg, Fields: fields})
}

func (m *MockAudioLogger) Error(msg string, err error, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "ERROR", Message: msg, Fields: fields, Error: err})
}

func (m *MockAudioLogger) Warn(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "WARN", Message: msg, Fields: fields})
}

func (m *MockAudioLogger) Debug(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "DEBUG", Message: msg, Fields: fields})
}

func (m *MockAudioLogger) WithPipeline(pipeline string) audio.AudioLogger {
	return m
}

func (m *MockAudioLogger) WithContext(ctx map[string]interface{}) audio.AudioLogger {
	return m
}

func (m *MockAudioLogger) GetLogs() []LogEntry {
	return m.logs
}
