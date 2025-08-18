package audio_test

import (
	"testing"

	"github.com/latoulicious/HKTM/pkg/audio"
)

// TestValidateSystemDependencies tests the system dependency validation
func TestValidateSystemDependencies(t *testing.T) {
	// This test will fail if ffmpeg or yt-dlp are not installed
	// but that's expected behavior - the system should validate dependencies
	err := audio.ValidateSystemDependencies()
	if err != nil {
		t.Logf("System dependencies validation failed (expected if binaries not installed): %v", err)
		// Don't fail the test - this is expected in CI environments
		return
	}
	t.Log("System dependencies validation passed")
}

// TestBinaryValidationFunctions tests individual binary validation functions
func TestBinaryValidationFunctions(t *testing.T) {
	tests := []struct {
		name       string
		binaryName string
		binaryPath string
		expectErr  bool
	}{
		{
			name:       "empty binary path",
			binaryName: "test",
			binaryPath: "",
			expectErr:  true,
		},
		{
			name:       "nonexistent binary",
			binaryName: "test",
			binaryPath: "nonexistent-binary-12345",
			expectErr:  true,
		},
		{
			name:       "valid binary (sh should exist on most systems)",
			binaryName: "sh",
			binaryPath: "sh",
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := audio.ValidateBinaryDependency(tt.binaryName, tt.binaryPath)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.name, err)
			}
		})
	}
}

// TestValidateAllBinaryDependencies tests the comprehensive binary validation
func TestValidateAllBinaryDependencies(t *testing.T) {
	tests := []struct {
		name       string
		ffmpegPath string
		ytdlpPath  string
		expectErr  bool
	}{
		{
			name:       "empty paths",
			ffmpegPath: "",
			ytdlpPath:  "",
			expectErr:  true,
		},
		{
			name:       "nonexistent binaries",
			ffmpegPath: "nonexistent-ffmpeg",
			ytdlpPath:  "nonexistent-ytdlp",
			expectErr:  true,
		},
		{
			name:       "mixed valid/invalid",
			ffmpegPath: "sh", // Should exist
			ytdlpPath:  "nonexistent-ytdlp",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := audio.ValidateAllBinaryDependencies(tt.ffmpegPath, tt.ytdlpPath)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error for %s, but got: %v", tt.name, err)
			}
		})
	}
}