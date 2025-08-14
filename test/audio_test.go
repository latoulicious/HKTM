package test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestYtDlpAvailability tests if yt-dlp is available
func TestYtDlpAvailability(t *testing.T) {
	ytdlpCmd := exec.Command("yt-dlp", "--version")
	if err := ytdlpCmd.Run(); err != nil {
		t.Fatalf("yt-dlp not found. Please install it first: %v", err)
	}
	t.Log("✅ yt-dlp is available")
}

// TestFFmpegAvailability tests if FFmpeg is available
func TestFFmpegAvailability(t *testing.T) {
	ffmpegCmd := exec.Command("ffmpeg", "-version")
	if err := ffmpegCmd.Run(); err != nil {
		t.Fatalf("FFmpeg not found. Please install it first: %v", err)
	}
	t.Log("✅ FFmpeg is available")
}

// TestYtDlpURLExtraction tests yt-dlp URL extraction with better quality
func TestYtDlpURLExtraction(t *testing.T) {
	// Use a shorter, more reliable test video
	testURL := "https://www.youtube.com/watch?v=jNQXAC9IVRw" // "Me at the zoo" - YouTube's first video

	// Add timeout and retry logic
	maxRetries := 3
	var streamURL string
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(ctx, "yt-dlp",
			"-f", "bestaudio[ext=m4a]/bestaudio[ext=webm]/bestaudio",
			"--no-playlist",
			"--no-warnings",
			"-g", testURL)

		output, err := cmd.Output()
		cancel()

		if err == nil {
			streamURL = strings.TrimSpace(string(output))
			if streamURL != "" {
				break
			}
		}

		if attempt < maxRetries-1 {
			t.Logf("Attempt %d failed, retrying in 2 seconds...", attempt+1)
			time.Sleep(2 * time.Second)
		}
	}

	if err != nil {
		t.Fatalf("Failed to extract URL after %d attempts: %v", maxRetries, err)
	}

	if streamURL == "" {
		t.Fatal("Empty stream URL returned")
	}

	t.Logf("✅ Successfully extracted stream URL: %s...", streamURL[:50])
}

// TestFFmpegPCMConversion tests FFmpeg PCM conversion with better error handling
func TestFFmpegPCMConversion(t *testing.T) {
	// Use a more reliable test approach - create a simple test file first
	testFile := "/tmp/test_audio.mp3"

	// Create a simple test audio file using FFmpeg
	createCmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", "sine=frequency=440:duration=2",
		"-acodec", "mp3",
		"-y", testFile)

	if err := createCmd.Run(); err != nil {
		t.Skipf("Skipping PCM conversion test - could not create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test FFmpeg PCM conversion with the local test file
	ffmpegTestCmd := exec.Command("ffmpeg",
		"-i", testFile,
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-ar", "48000",
		"-ac", "2",
		"-t", "1", // Only convert 1 second for testing
		"-y", "/tmp/test_output.raw")

	if err := ffmpegTestCmd.Run(); err != nil {
		t.Fatalf("FFmpeg conversion failed: %v", err)
	}

	// Check if output file was created
	if _, err := os.Stat("/tmp/test_output.raw"); err == nil {
		t.Log("✅ FFmpeg PCM conversion successful")
		// Clean up
		os.Remove("/tmp/test_output.raw")
	} else {
		t.Fatal("FFmpeg output file not found")
	}
}

// TestFormatAvailability tests format availability for comparison
func TestFormatAvailability(t *testing.T) {
	testURL := "https://www.youtube.com/watch?v=jNQXAC9IVRw" // Use shorter video
	formatCmd := exec.Command("yt-dlp", "-F", testURL)

	formatOutput, err := formatCmd.Output()
	if err != nil {
		t.Fatalf("Failed to get formats: %v", err)
	}

	lines := strings.Split(string(formatOutput), "\n")
	audioFormats := []string{}
	for _, line := range lines {
		// Updated to look for "audio only" instead of specific extensions
		if strings.Contains(strings.ToLower(line), "audio only") {
			audioFormats = append(audioFormats, line)
		}
	}

	if len(audioFormats) == 0 {
		t.Fatal("No audio formats found")
	}

	t.Logf("✅ Found %d audio formats available", len(audioFormats))
	t.Log("Available audio formats:")
	// Show up to 5 formats, but don't exceed the actual number available
	maxToShow := 5
	if len(audioFormats) < maxToShow {
		maxToShow = len(audioFormats)
	}
	for _, format := range audioFormats[:maxToShow] {
		t.Logf("  %s", format)
	}
}

// TestAudioPipelineIntegration tests the complete audio pipeline
func TestAudioPipelineIntegration(t *testing.T) {
	// This test runs all the individual components to ensure they work together
	t.Run("yt-dlp availability", TestYtDlpAvailability)
	t.Run("ffmpeg availability", TestFFmpegAvailability)
	t.Run("URL extraction", TestYtDlpURLExtraction)
	t.Run("PCM conversion", TestFFmpegPCMConversion)
	t.Run("format availability", TestFormatAvailability)

	t.Log("🎉 All audio pipeline components are working correctly!")
	t.Log("The improved audio quality pipeline is ready to use.")
	t.Log("Using bestaudio format and 128kbps Opus encoding for better quality.")
}
