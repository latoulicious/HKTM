package audio

import (
	"testing"

	"github.com/latoulicious/HKTM/tools"
)

func TestBinaryValidator_QuickValidation(t *testing.T) {
	tests := []struct {
		name        string
		ffmpegPath  string
		ytdlpPath   string
		expectError bool
	}{
		{
			name:        "valid binaries",
			ffmpegPath:  "ffmpeg",
			ytdlpPath:   "yt-dlp",
			expectError: false, // This will fail if binaries aren't installed, which is expected
		},
		{
			name:        "invalid ffmpeg path",
			ffmpegPath:  "nonexistent-ffmpeg",
			ytdlpPath:   "yt-dlp",
			expectError: true,
		},
		{
			name:        "invalid yt-dlp path",
			ffmpegPath:  "ffmpeg",
			ytdlpPath:   "nonexistent-yt-dlp",
			expectError: true,
		},
		{
			name:        "both invalid",
			ffmpegPath:  "nonexistent-ffmpeg",
			ytdlpPath:   "nonexistent-yt-dlp",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := tools.NewBinaryValidator(tt.ffmpegPath, tt.ytdlpPath)
			err := validator.QuickValidation()

			if tt.expectError && err == nil {
				t.Errorf("QuickValidation() expected error but got none")
			}
			if !tt.expectError && err != nil {
				// Only log this as it might fail in CI environments without binaries
				t.Logf("QuickValidation() unexpected error (may be expected in CI): %v", err)
			}
		})
	}
}

func TestBinaryValidator_ValidateAllBinaries(t *testing.T) {
	validator := tools.NewBinaryValidator("ffmpeg", "yt-dlp")

	results, err := validator.ValidateAllBinaries()

	// Check that we get results for both binaries
	if len(results) != 2 {
		t.Errorf("ValidateAllBinaries() expected 2 results, got %d", len(results))
	}

	// Check that both expected binaries are in results
	if _, exists := results["ffmpeg"]; !exists {
		t.Error("ValidateAllBinaries() missing ffmpeg in results")
	}
	if _, exists := results["yt-dlp"]; !exists {
		t.Error("ValidateAllBinaries() missing yt-dlp in results")
	}

	// Log the status for debugging (don't fail test if binaries aren't available)
	if err != nil {
		t.Logf("Binary validation failed (expected in environments without binaries): %v", err)
	}

	// Test the status output
	status := validator.GetBinaryStatus()
	if status == "" {
		t.Error("GetBinaryStatus() returned empty string")
	}
	t.Logf("Binary status:\n%s", status)
}

func TestBinaryValidator_ValidateFFmpeg(t *testing.T) {
	validator := tools.NewBinaryValidator("ffmpeg", "yt-dlp")

	info := validator.ValidateFFmpeg()

	if info == nil {
		t.Fatal("ValidateFFmpeg() returned nil")
	}

	if info.Name != "FFmpeg" {
		t.Errorf("ValidateFFmpeg() expected name 'FFmpeg', got '%s'", info.Name)
	}

	// Log the result for debugging
	if info.IsAvailable {
		t.Logf("FFmpeg available: %s at %s", info.Version, info.Path)
	} else {
		t.Logf("FFmpeg not available: %s", info.ErrorMessage)
	}
}

func TestBinaryValidator_ValidateYtDlp(t *testing.T) {
	validator := tools.NewBinaryValidator("ffmpeg", "yt-dlp")

	info := validator.ValidateYtDlp()

	if info == nil {
		t.Fatal("ValidateYtDlp() returned nil")
	}

	if info.Name != "yt-dlp" {
		t.Errorf("ValidateYtDlp() expected name 'yt-dlp', got '%s'", info.Name)
	}

	// Log the result for debugging
	if info.IsAvailable {
		t.Logf("yt-dlp available: %s at %s", info.Version, info.Path)
	} else {
		t.Logf("yt-dlp not available: %s", info.ErrorMessage)
	}
}

func TestBinaryValidator_ErrorMessages(t *testing.T) {
	// Test with definitely invali	d paths
	validator := tools.NewBinaryValidator("/nonexistent/ffmpeg", "/nonexistent/yt-dlp")

	ffmpegInfo := validator.ValidateFFmpeg()
	if ffmpegInfo.IsAvailable {
		t.Error("ValidateFFmpeg() should fail with nonexistent path")
	}
	if ffmpegInfo.ErrorMessage == "" {
		t.Error("ValidateFFmpeg() should provide error message for missing binary")
	}
	if !containsInstallInstructions(ffmpegInfo.ErrorMessage) {
		t.Error("ValidateFFmpeg() error message should contain installation instructions")
	}

	ytdlpInfo := validator.ValidateYtDlp()
	if ytdlpInfo.IsAvailable {
		t.Error("ValidateYtDlp() should fail with nonexistent path")
	}
	if ytdlpInfo.ErrorMessage == "" {
		t.Error("ValidateYtDlp() should provide error message for missing binary")
	}
	if !containsInstallInstructions(ytdlpInfo.ErrorMessage) {
		t.Error("ValidateYtDlp() error message should contain installation instructions")
	}
}

// Helper function to check if error message contains installation instructions
func containsInstallInstructions(message string) bool {
	// Check for common installation instruction keywords
	keywords := []string{"Installation Instructions", "sudo apt", "brew install", "pip install"}
	for _, keyword := range keywords {
		if len(message) > 0 && contains(message, keyword) {
			return true
		}
	}
	return false
}

// Simple contains function since we can't import strings in test
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || findSubstring(s, substr) >= 0)
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
