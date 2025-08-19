package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/latoulicious/HKTM/pkg/audio"
)

func main() {
	fmt.Println("=== Audio Binary Dependencies Verification Tool ===")
	fmt.Println()

	// Load configuration to get binary paths
	configManager, err := audio.NewConfigManager()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		fmt.Println("   Using default binary paths")
		checkDefaultBinaries()
		os.Exit(1)
	}

	ffmpegConfig := configManager.GetFFmpegConfig()
	ytdlpConfig := configManager.GetYtDlpConfig()
	
	fmt.Printf("📋 Configuration loaded successfully\n")
	fmt.Printf("   FFmpeg binary path: %s\n", ffmpegConfig.BinaryPath)
	fmt.Printf("   yt-dlp binary path: %s\n", ytdlpConfig.BinaryPath)
	fmt.Printf("   Audio format: %s\n", ffmpegConfig.AudioFormat)
	fmt.Printf("   Sample rate: %d Hz\n", ffmpegConfig.SampleRate)
	fmt.Printf("   Channels: %d\n", ffmpegConfig.Channels)
	fmt.Println()

	var hasErrors bool

	// Check FFmpeg binary availability
	fmt.Println("🔍 Checking FFmpeg binary...")
	if err := checkFFmpegBinary(ffmpegConfig.BinaryPath); err != nil {
		fmt.Printf("❌ FFmpeg validation failed: %v\n", err)
		hasErrors = true
	} else {
		fmt.Println("✅ FFmpeg binary verification successful!")
	}
	fmt.Println()

	// Check yt-dlp binary availability
	fmt.Println("🔍 Checking yt-dlp binary...")
	if err := checkYtDlpBinary(ytdlpConfig.BinaryPath); err != nil {
		fmt.Printf("❌ yt-dlp validation failed: %v\n", err)
		hasErrors = true
	} else {
		fmt.Println("✅ yt-dlp binary verification successful!")
	}
	fmt.Println()

	if hasErrors {
		printTroubleshootingGuide()
		os.Exit(1)
	}

	fmt.Println("🎉 All audio binary dependencies verified successfully!")
	fmt.Println("   The audio pipeline should be able to process streams.")
}

func checkDefaultBinaries() {
	fmt.Println("🔍 Checking default binary paths...")
	
	var hasErrors bool
	
	if err := checkFFmpegBinary("ffmpeg"); err != nil {
		fmt.Printf("❌ FFmpeg validation failed: %v\n", err)
		hasErrors = true
	}
	
	if err := checkYtDlpBinary("yt-dlp"); err != nil {
		fmt.Printf("❌ yt-dlp validation failed: %v\n", err)
		hasErrors = true
	}
	
	if hasErrors {
		printTroubleshootingGuide()
	}
}

func checkFFmpegBinary(binaryPath string) error {
	// Use the existing validation function from audio utils
	if err := audio.ValidateFFmpegBinary(binaryPath); err != nil {
		return err
	}

	fmt.Printf("   ✅ FFmpeg binary found and functional\n")
	fmt.Printf("   📍 Path: %s\n", binaryPath)

	// Get absolute path if it's in PATH
	if !filepath.IsAbs(binaryPath) {
		if absPath, err := filepath.Abs(binaryPath); err == nil {
			fmt.Printf("   📍 Resolved to: %s\n", absPath)
		}
	}

	return nil
}

func checkYtDlpBinary(binaryPath string) error {
	// Use the existing validation function from audio utils
	if err := audio.ValidateYtDlpBinary(binaryPath); err != nil {
		return err
	}

	fmt.Printf("   ✅ yt-dlp binary found and functional\n")
	fmt.Printf("   📍 Path: %s\n", binaryPath)

	// Get absolute path if it's in PATH
	if !filepath.IsAbs(binaryPath) {
		if absPath, err := filepath.Abs(binaryPath); err == nil {
			fmt.Printf("   📍 Resolved to: %s\n", absPath)
		}
	}

	return nil
}

func printTroubleshootingGuide() {
	fmt.Println("🔧 Troubleshooting Guide:")
	fmt.Println()
	fmt.Println("📦 Install Missing Dependencies:")
	fmt.Println()
	fmt.Println("   FFmpeg:")
	fmt.Println("   Ubuntu/Debian: sudo apt update && sudo apt install ffmpeg")
	fmt.Println("   CentOS/RHEL:   sudo yum install ffmpeg")
	fmt.Println("   macOS:         brew install ffmpeg")
	fmt.Println("   Windows:       Download from https://ffmpeg.org/download.html")
	fmt.Println()
	fmt.Println("   yt-dlp:")
	fmt.Println("   pip install yt-dlp")
	fmt.Println("   Or: sudo apt install yt-dlp  (Ubuntu 22.04+)")
	fmt.Println("   Or: brew install yt-dlp      (macOS)")
	fmt.Println("   Or: Download from https://github.com/yt-dlp/yt-dlp/releases")
	fmt.Println()
	fmt.Println("🔍 Verify Installation:")
	fmt.Println("   which ffmpeg && ffmpeg -version")
	fmt.Println("   which yt-dlp && yt-dlp --version")
	fmt.Println()
	fmt.Println("⚙️  Configure Custom Paths:")
	fmt.Println("   Edit config/audio.yaml:")
	fmt.Println("   ffmpeg:")
	fmt.Println("     binary_path: \"/path/to/your/ffmpeg\"")
	fmt.Println("   ytdlp:")
	fmt.Println("     binary_path: \"/path/to/your/yt-dlp\"")
	fmt.Println()
	fmt.Println("   Or set environment variables:")
	fmt.Println("   export AUDIO_FFMPEG_BINARY=\"/path/to/your/ffmpeg\"")
	fmt.Println("   export AUDIO_YTDLP_BINARY=\"/path/to/your/yt-dlp\"")
}