package tools

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// BinaryInfo contains information about a binary dependency
type BinaryInfo struct {
	Name         string
	Path         string
	Version      string
	IsAvailable  bool
	ErrorMessage string
}

// BinaryValidator provides comprehensive binary validation functionality
type BinaryValidator struct {
	ffmpegPath string
	ytdlpPath  string
}

// NewBinaryValidator creates a new binary validator with the specified paths
func NewBinaryValidator(ffmpegPath, ytdlpPath string) *BinaryValidator {
	return &BinaryValidator{
		ffmpegPath: ffmpegPath,
		ytdlpPath:  ytdlpPath,
	}
}

// ValidateAllBinaries validates all required binary dependencies and returns detailed information
func (bv *BinaryValidator) ValidateAllBinaries() (map[string]*BinaryInfo, error) {
	results := make(map[string]*BinaryInfo)
	var errors []string

	// Validate FFmpeg
	ffmpegInfo := bv.ValidateFFmpeg()
	results["ffmpeg"] = ffmpegInfo
	if !ffmpegInfo.IsAvailable {
		errors = append(errors, ffmpegInfo.ErrorMessage)
	}

	// Validate yt-dlp
	ytdlpInfo := bv.ValidateYtDlp()
	results["yt-dlp"] = ytdlpInfo
	if !ytdlpInfo.IsAvailable {
		errors = append(errors, ytdlpInfo.ErrorMessage)
	}

	// Return aggregated error if any binary is missing
	if len(errors) > 0 {
		return results, fmt.Errorf("binary validation failed:\n%s", strings.Join(errors, "\n"))
	}

	return results, nil
}

// ValidateFFmpeg validates FFmpeg binary availability and version compatibility
func (bv *BinaryValidator) ValidateFFmpeg() *BinaryInfo {
	info := &BinaryInfo{
		Name: "FFmpeg",
		Path: bv.ffmpegPath,
	}

	// Check if binary exists in PATH
	fullPath, err := exec.LookPath(bv.ffmpegPath)
	if err != nil {
		info.ErrorMessage = bv.createFFmpegErrorMessage(err)
		return info
	}

	info.Path = fullPath

	// Get version information
	version, err := bv.getFFmpegVersion(fullPath)
	if err != nil {
		info.ErrorMessage = fmt.Sprintf("FFmpeg found at '%s' but version check failed: %v\n%s",
			fullPath, err, bv.getFFmpegInstallInstructions())
		return info
	}

	info.Version = version

	// Validate version compatibility
	if err := bv.validateFFmpegVersion(version); err != nil {
		info.ErrorMessage = fmt.Sprintf("FFmpeg version compatibility issue: %v\n%s",
			err, bv.getFFmpegInstallInstructions())
		return info
	}

	info.IsAvailable = true
	return info
}

// ValidateYtDlp validates yt-dlp binary availability and version compatibility
func (bv *BinaryValidator) ValidateYtDlp() *BinaryInfo {
	info := &BinaryInfo{
		Name: "yt-dlp",
		Path: bv.ytdlpPath,
	}

	// Check if binary exists in PATH
	fullPath, err := exec.LookPath(bv.ytdlpPath)
	if err != nil {
		info.ErrorMessage = bv.createYtDlpErrorMessage(err)
		return info
	}

	info.Path = fullPath

	// Get version information
	version, err := bv.getYtDlpVersion(fullPath)
	if err != nil {
		info.ErrorMessage = fmt.Sprintf("yt-dlp found at '%s' but version check failed: %v\n%s",
			fullPath, err, bv.getYtDlpInstallInstructions())
		return info
	}

	info.Version = version

	// Validate version compatibility
	if err := bv.validateYtDlpVersion(version); err != nil {
		info.ErrorMessage = fmt.Sprintf("yt-dlp version compatibility issue: %v\n%s",
			err, bv.getYtDlpInstallInstructions())
		return info
	}

	info.IsAvailable = true
	return info
}

// getFFmpegVersion extracts version information from FFmpeg
func (bv *BinaryValidator) getFFmpegVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute ffmpeg -version: %w", err)
	}

	outputStr := string(output)

	// Extract version from output like "ffmpeg version 4.4.2-0ubuntu0.22.04.1"
	versionRegex := regexp.MustCompile(`ffmpeg version ([^\s]+)`)
	matches := versionRegex.FindStringSubmatch(outputStr)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse version from ffmpeg output")
	}

	return matches[1], nil
}

// getYtDlpVersion extracts version information from yt-dlp
func (bv *BinaryValidator) getYtDlpVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute yt-dlp --version: %w", err)
	}

	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", fmt.Errorf("yt-dlp returned empty version")
	}

	return version, nil
}

// validateFFmpegVersion checks if FFmpeg version meets minimum requirements
func (bv *BinaryValidator) validateFFmpegVersion(version string) error {
	// Extract major version number, handling versions like "n7.1.1" or "4.4.2-0ubuntu0.22.04.1"
	versionParts := strings.Split(version, ".")
	if len(versionParts) == 0 {
		return fmt.Errorf("invalid version format: %s", version)
	}

	majorVersionStr := strings.Split(versionParts[0], "-")[0] // Handle versions like "4.4.2-0ubuntu0.22.04.1"

	// Remove leading 'n' if present (e.g., "n7.1.1" -> "7.1.1")
	if strings.HasPrefix(majorVersionStr, "n") {
		majorVersionStr = majorVersionStr[1:]
	}

	majorVersion, err := strconv.Atoi(majorVersionStr)
	if err != nil {
		return fmt.Errorf("could not parse major version from: %s", version)
	}

	// Require FFmpeg 4.0 or higher for modern codec support
	if majorVersion < 4 {
		return fmt.Errorf("FFmpeg version %s is too old (minimum required: 4.0)", version)
	}

	return nil
}

// validateYtDlpVersion checks if yt-dlp version meets minimum requirements
func (bv *BinaryValidator) validateYtDlpVersion(version string) error {
	// yt-dlp versions are typically in format like "2023.07.06" or "2023.07.06.1"
	// We'll do a basic check to ensure it's a reasonable recent version

	// Extract year from version
	versionParts := strings.Split(version, ".")
	if len(versionParts) < 3 {
		return fmt.Errorf("invalid yt-dlp version format: %s", version)
	}

	yearStr := versionParts[0]
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return fmt.Errorf("could not parse year from yt-dlp version: %s", version)
	}

	// Require yt-dlp from 2022 or later for modern YouTube support
	if year < 2022 {
		return fmt.Errorf("yt-dlp version %s is too old (minimum required: 2022.x.x)", version)
	}

	return nil
}

// createFFmpegErrorMessage creates a detailed error message for FFmpeg issues
func (bv *BinaryValidator) createFFmpegErrorMessage(err error) string {
	return fmt.Sprintf(`FFmpeg binary not found at path '%s': %v

%s`, bv.ffmpegPath, err, bv.getFFmpegInstallInstructions())
}

// createYtDlpErrorMessage creates a detailed error message for yt-dlp issues
func (bv *BinaryValidator) createYtDlpErrorMessage(err error) string {
	return fmt.Sprintf(`yt-dlp binary not found at path '%s': %v

%s`, bv.ytdlpPath, err, bv.getYtDlpInstallInstructions())
}

// getFFmpegInstallInstructions provides platform-specific installation instructions for FFmpeg
func (bv *BinaryValidator) getFFmpegInstallInstructions() string {
	return `FFmpeg Installation Instructions:

Ubuntu/Debian:
  sudo apt update && sudo apt install ffmpeg

CentOS/RHEL/Fedora:
  sudo dnf install ffmpeg  # or: sudo yum install ffmpeg

macOS (with Homebrew):
  brew install ffmpeg

Windows:
  1. Download from https://ffmpeg.org/download.html
  2. Extract to a folder (e.g., C:\ffmpeg)
  3. Add C:\ffmpeg\bin to your PATH environment variable

Docker:
  Use a base image that includes ffmpeg, or add to your Dockerfile:
  RUN apt-get update && apt-get install -y ffmpeg

After installation, verify with: ffmpeg -version

If FFmpeg is installed in a custom location, update your configuration:
  ffmpeg:
    binary_path: "/path/to/your/ffmpeg"`
}

// getYtDlpInstallInstructions provides platform-specific installation instructions for yt-dlp
func (bv *BinaryValidator) getYtDlpInstallInstructions() string {
	return `yt-dlp Installation Instructions:

Python pip (recommended):
  pip install yt-dlp
  # or for user installation: pip install --user yt-dlp

Ubuntu/Debian:
  sudo apt update && sudo apt install yt-dlp
  # or from pip if package is outdated

macOS (with Homebrew):
  brew install yt-dlp

Windows:
  1. Install Python from https://python.org
  2. Run: pip install yt-dlp
  3. Or download standalone executable from https://github.com/yt-dlp/yt-dlp/releases

Docker:
  Add to your Dockerfile:
  RUN pip install yt-dlp

After installation, verify with: yt-dlp --version

If yt-dlp is installed in a custom location, update your configuration:
  ytdlp:
    binary_path: "/path/to/your/yt-dlp"

Note: yt-dlp is actively maintained and preferred over youtube-dl for better YouTube support.`
}

// GetBinaryStatus returns a human-readable status summary of all binaries
func (bv *BinaryValidator) GetBinaryStatus() string {
	results, err := bv.ValidateAllBinaries()

	var status strings.Builder
	status.WriteString("Binary Dependency Status:\n")
	status.WriteString("========================\n\n")

	for name, info := range results {
		if info.IsAvailable {
			status.WriteString(fmt.Sprintf("✅ %s: OK\n", info.Name))
			status.WriteString(fmt.Sprintf("   Path: %s\n", info.Path))
			status.WriteString(fmt.Sprintf("   Version: %s\n\n", info.Version))
		} else {
			status.WriteString(fmt.Sprintf("❌ %s: NOT AVAILABLE\n", info.Name))
			status.WriteString(fmt.Sprintf("   Expected path: %s\n\n", bv.getBinaryPath(name)))
		}
	}

	if err != nil {
		status.WriteString("Issues found:\n")
		status.WriteString(err.Error())
	} else {
		status.WriteString("All required binaries are available and compatible!")
	}

	return status.String()
}

// getBinaryPath returns the configured path for a binary by name
func (bv *BinaryValidator) getBinaryPath(name string) string {
	switch name {
	case "ffmpeg":
		return bv.ffmpegPath
	case "yt-dlp":
		return bv.ytdlpPath
	default:
		return "unknown"
	}
}

// QuickValidation performs a fast check without detailed version validation
// Useful for startup checks where speed is important
func (bv *BinaryValidator) QuickValidation() error {
	// Quick check for FFmpeg
	if _, err := exec.LookPath(bv.ffmpegPath); err != nil {
		return fmt.Errorf("FFmpeg not found at '%s': %w", bv.ffmpegPath, err)
	}

	// Quick check for yt-dlp
	if _, err := exec.LookPath(bv.ytdlpPath); err != nil {
		return fmt.Errorf("yt-dlp not found at '%s': %w", bv.ytdlpPath, err)
	}

	return nil
}
