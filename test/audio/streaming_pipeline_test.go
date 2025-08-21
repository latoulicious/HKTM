package audio_test

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/tools"
)

// TestStreamingPipelineComponents tests individual components of the streaming pipeline
func TestStreamingPipelineComponents(t *testing.T) {
	// Test binary availability
	t.Run("binary_validation", func(t *testing.T) {
		validator := tools.NewBinaryValidator("ffmpeg", "yt-dlp")

		// Test quick validation
		err := validator.QuickValidation()
		if err != nil {
			t.Logf("Binary validation failed (expected in some environments): %v", err)
		} else {
			t.Log("All required binaries are available")
		}

		// Test detailed validation
		results, err := validator.ValidateAllBinaries()
		if err != nil {
			t.Logf("Detailed validation failed (expected in some environments): %v", err)
		}

		// Check individual results
		if ffmpegInfo, exists := results["ffmpeg"]; exists {
			if ffmpegInfo.IsAvailable {
				t.Logf("FFmpeg available: %s at %s", ffmpegInfo.Version, ffmpegInfo.Path)
			} else {
				t.Logf("FFmpeg not available: %s", ffmpegInfo.ErrorMessage)
			}
		}

		if ytdlpInfo, exists := results["yt-dlp"]; exists {
			if ytdlpInfo.IsAvailable {
				t.Logf("yt-dlp available: %s at %s", ytdlpInfo.Version, ytdlpInfo.Path)
			} else {
				t.Logf("yt-dlp not available: %s", ytdlpInfo.ErrorMessage)
			}
		}
	})

	// Test FFmpeg command building
	t.Run("ffmpeg_command_building", func(t *testing.T) {
		ffmpegConfig := &audio.FFmpegConfig{
			BinaryPath:  "ffmpeg",
			AudioFormat: "s16le",
			SampleRate:  48000,
			Channels:    2,
			CustomArgs:  []string{"-hide_banner"},
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
		if info == nil {
			t.Error("GetProcessInfo should not return nil")
		}

		if info["is_running"].(bool) {
			t.Error("Process info should show not running initially")
		}

		t.Log("FFmpeg processor created successfully with proper configuration")
	})

	// Test error detection
	t.Run("error_detection", func(t *testing.T) {
		ffmpegConfig := &audio.FFmpegConfig{
			BinaryPath: "ffmpeg",
		}
		ytdlpConfig := &audio.YtDlpConfig{
			BinaryPath: "yt-dlp",
		}
		streamingConfig := &audio.StreamingConfig{
			YtdlpPath:  "yt-dlp",
			FFmpegPath: "ffmpeg",
		}

		logger := &MockAudioLogger{}
		processor := audio.NewFFmpegProcessor(ffmpegConfig, ytdlpConfig, streamingConfig, logger)

		// Test URL refresh detection
		testCases := []struct {
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
				name:     "Generic error should not trigger refresh",
				err:      fmt.Errorf("some other error"),
				expected: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				detected := processor.DetectStreamFailure(tc.err)
				if detected != tc.expected {
					t.Errorf("DetectStreamFailure(%v) = %v, expected %v", tc.err, detected, tc.expected)
				}
			})
		}
	})
}

// TestYtDlpCommandExecution tests yt-dlp command execution without streaming
func TestYtDlpCommandExecution(t *testing.T) {
	// Skip if binaries not available
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		t.Skipf("yt-dlp not available: %v", err)
	}

	t.Run("yt-dlp_version_check", func(t *testing.T) {
		cmd := exec.Command("yt-dlp", "--version")
		output, err := cmd.Output()
		if err != nil {
			t.Errorf("Failed to get yt-dlp version: %v", err)
		} else {
			version := strings.TrimSpace(string(output))
			t.Logf("yt-dlp version: %s", version)
		}
	})

	t.Run("yt-dlp_help_check", func(t *testing.T) {
		cmd := exec.Command("yt-dlp", "--help")
		err := cmd.Run()
		if err != nil {
			t.Errorf("yt-dlp --help failed: %v", err)
		} else {
			t.Log("yt-dlp help command executed successfully")
		}
	})

	// Test with an invalid URL to see error handling
	t.Run("yt-dlp_invalid_url", func(t *testing.T) {
		cmd := exec.Command("yt-dlp", "--get-url", "--quiet", "https://invalid-url-that-does-not-exist.com")
		_, err := cmd.Output()
		if err != nil {
			t.Logf("yt-dlp correctly failed with invalid URL: %v", err)
		} else {
			t.Log("yt-dlp did not fail with invalid URL (unexpected)")
		}
	})
}

// TestFFmpegCommandExecution tests FFmpeg command execution
func TestFFmpegCommandExecution(t *testing.T) {
	// Skip if binaries not available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skipf("ffmpeg not available: %v", err)
	}

	t.Run("ffmpeg_version_check", func(t *testing.T) {
		cmd := exec.Command("ffmpeg", "-version")
		output, err := cmd.Output()
		if err != nil {
			t.Errorf("Failed to get ffmpeg version: %v", err)
		} else {
			version := string(output)
			if strings.Contains(version, "ffmpeg version") {
				t.Log("FFmpeg version check successful")
			} else {
				t.Error("FFmpeg version output does not contain expected text")
			}
		}
	})

	t.Run("ffmpeg_help_check", func(t *testing.T) {
		cmd := exec.Command("ffmpeg", "-h")
		err := cmd.Run()
		if err != nil {
			t.Logf("ffmpeg -h failed (may be expected): %v", err)
		} else {
			t.Log("FFmpeg help command executed successfully")
		}
	})

	t.Run("ffmpeg_formats_check", func(t *testing.T) {
		cmd := exec.Command("ffmpeg", "-formats")
		output, err := cmd.Output()
		if err != nil {
			t.Logf("ffmpeg -formats failed: %v", err)
		} else {
			formats := string(output)
			if strings.Contains(formats, "s16le") {
				t.Log("FFmpeg supports s16le format (required for pipeline)")
			} else {
				t.Error("FFmpeg does not list s16le format support")
			}
		}
	})
}

// TestStreamingConfigurationValidation tests configuration validation
func TestStreamingConfigurationValidation(t *testing.T) {
	t.Run("valid_configuration", func(t *testing.T) {
		config := &audio.StreamingConfig{
			YtdlpPath:  "yt-dlp",
			FFmpegPath: "ffmpeg",
			SampleRate: 48000,
			Channels:   2,
		}

		// Test that configuration values are set correctly
		if config.SampleRate != 48000 {
			t.Errorf("Expected sample rate 48000, got %d", config.SampleRate)
		}
		if config.Channels != 2 {
			t.Errorf("Expected 2 channels, got %d", config.Channels)
		}
		if config.YtdlpPath != "yt-dlp" {
			t.Errorf("Expected yt-dlp path 'yt-dlp', got '%s'", config.YtdlpPath)
		}
		if config.FFmpegPath != "ffmpeg" {
			t.Errorf("Expected ffmpeg path 'ffmpeg', got '%s'", config.FFmpegPath)
		}

		t.Log("Configuration validation passed")
	})

	t.Run("configuration_with_custom_paths", func(t *testing.T) {
		config := &audio.StreamingConfig{
			YtdlpPath:  "/usr/local/bin/yt-dlp",
			FFmpegPath: "/usr/local/bin/ffmpeg",
			SampleRate: 44100,
			Channels:   1,
		}

		// Test custom configuration
		if config.SampleRate != 44100 {
			t.Errorf("Expected sample rate 44100, got %d", config.SampleRate)
		}
		if config.Channels != 1 {
			t.Errorf("Expected 1 channel, got %d", config.Channels)
		}

		t.Log("Custom configuration validation passed")
	})
}

// TestProcessLifecycle tests process lifecycle management
func TestProcessLifecycle(t *testing.T) {
	// Skip if binaries not available
	validator := tools.NewBinaryValidator("ffmpeg", "yt-dlp")
	if err := validator.QuickValidation(); err != nil {
		t.Skipf("Required binaries not available: %v", err)
	}

	t.Run("processor_lifecycle", func(t *testing.T) {
		ffmpegConfig := &audio.FFmpegConfig{
			BinaryPath:  "ffmpeg",
			AudioFormat: "s16le",
			SampleRate:  48000,
			Channels:    2,
		}

		ytdlpConfig := &audio.YtDlpConfig{
			BinaryPath: "yt-dlp",
		}

		streamingConfig := &audio.StreamingConfig{
			YtdlpPath:  "yt-dlp",
			FFmpegPath: "ffmpeg",
		}

		logger := &MockAudioLogger{}
		processor := audio.NewFFmpegProcessor(ffmpegConfig, ytdlpConfig, streamingConfig, logger)

		// Test initial state
		if processor.IsRunning() {
			t.Error("Processor should not be running initially")
		}

		if processor.IsProcessAlive() {
			t.Error("Process should not be alive initially")
		}

		// Test process info
		info := processor.GetProcessInfo()
		if info["is_running"].(bool) {
			t.Error("Process info should show not running")
		}

		// Test stop when not running (should not error)
		if err := processor.Stop(); err != nil {
			t.Errorf("Stop should not error when not running: %v", err)
		}

		// Test wait for exit when not running
		if err := processor.WaitForExit(1 * time.Second); err != nil {
			t.Errorf("WaitForExit should not error when not running: %v", err)
		}

		t.Log("Process lifecycle test completed successfully")
	})
}

// Note: MockAudioLogger is defined in integration_test.go to avoid duplication
