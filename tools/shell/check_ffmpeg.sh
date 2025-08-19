#!/bin/bash

# FFmpeg Binary Verification Script
# This script checks if FFmpeg is available in PATH or at the configured location

echo "=== FFmpeg Binary Verification ==="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "success")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "error")
            echo -e "${RED}❌ $message${NC}"
            ;;
        "warning")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "info")
            echo -e "${BLUE}ℹ️  $message${NC}"
            ;;
    esac
}

# Function to check if FFmpeg is in PATH
check_ffmpeg_in_path() {
    print_status "info" "Checking if FFmpeg is available in PATH..."
    
    if command -v ffmpeg >/dev/null 2>&1; then
        local ffmpeg_path=$(which ffmpeg)
        print_status "success" "FFmpeg found in PATH: $ffmpeg_path"
        
        # Test FFmpeg version
        local version=$(ffmpeg -version 2>/dev/null | head -n1 | cut -d' ' -f3)
        print_status "info" "FFmpeg version: $version"
        
        return 0
    else
        print_status "error" "FFmpeg not found in PATH"
        return 1
    fi
}

# Function to check configured FFmpeg path
check_configured_ffmpeg() {
    print_status "info" "Checking configured FFmpeg path from config files..."
    
    # Check if config file exists
    if [ -f "config/audio.yaml" ]; then
        # Extract FFmpeg binary path from YAML config
        local configured_path=$(grep -A 10 "^ffmpeg:" config/audio.yaml | grep "binary_path:" | sed 's/.*binary_path: *"\?\([^"]*\)"\?.*/\1/')
        
        if [ -n "$configured_path" ] && [ "$configured_path" != "ffmpeg" ]; then
            print_status "info" "Configured FFmpeg path: $configured_path"
            
            if [ -x "$configured_path" ]; then
                print_status "success" "Configured FFmpeg binary is executable"
                
                # Test configured FFmpeg version
                local version=$($configured_path -version 2>/dev/null | head -n1 | cut -d' ' -f3)
                print_status "info" "Configured FFmpeg version: $version"
                
                return 0
            else
                print_status "error" "Configured FFmpeg binary is not executable or not found: $configured_path"
                return 1
            fi
        else
            print_status "info" "Using default FFmpeg path from PATH"
            return 2  # Use PATH check
        fi
    else
        print_status "warning" "Config file config/audio.yaml not found, checking PATH only"
        return 2  # Use PATH check
    fi
}

# Function to test FFmpeg functionality
test_ffmpeg_functionality() {
    local ffmpeg_cmd=${1:-ffmpeg}
    
    print_status "info" "Testing FFmpeg basic functionality..."
    
    # Test FFmpeg with a simple null input/output
    if $ffmpeg_cmd -f lavfi -i testsrc=duration=1:size=320x240:rate=1 -f null - >/dev/null 2>&1; then
        print_status "success" "FFmpeg functionality test passed"
        return 0
    else
        print_status "error" "FFmpeg functionality test failed"
        return 1
    fi
}

# Function to print troubleshooting guide
print_troubleshooting() {
    echo
    echo "🔧 Troubleshooting Guide:"
    echo
    echo "1. Install FFmpeg:"
    echo "   Ubuntu/Debian: sudo apt update && sudo apt install ffmpeg"
    echo "   CentOS/RHEL:   sudo yum install ffmpeg"
    echo "   macOS:         brew install ffmpeg"
    echo "   Windows:       Download from https://ffmpeg.org/download.html"
    echo
    echo "2. Verify FFmpeg is in PATH:"
    echo "   Run: which ffmpeg"
    echo "   Or:  ffmpeg -version"
    echo
    echo "3. Configure custom FFmpeg path in config/audio.yaml:"
    echo "   ffmpeg:"
    echo "     binary_path: \"/path/to/your/ffmpeg\""
    echo
    echo "4. Or set environment variable:"
    echo "   export AUDIO_FFMPEG_BINARY=\"/path/to/your/ffmpeg\""
    echo
}

# Main execution
main() {
    local exit_code=0
    local ffmpeg_found=false
    local ffmpeg_path="ffmpeg"
    
    # Check configured FFmpeg first
    check_configured_ffmpeg
    local config_result=$?
    
    if [ $config_result -eq 0 ]; then
        # Configured path works
        ffmpeg_found=true
        ffmpeg_path=$(grep -A 10 "^ffmpeg:" config/audio.yaml | grep "binary_path:" | sed 's/.*binary_path: *"\?\([^"]*\)"\?.*/\1/')
    else
        # No custom config or config failed, check PATH
        if check_ffmpeg_in_path; then
            ffmpeg_found=true
            ffmpeg_path="ffmpeg"
        fi
    fi
    
    if [ "$ffmpeg_found" = true ]; then
        # Test functionality
        if test_ffmpeg_functionality "$ffmpeg_path"; then
            echo
            print_status "success" "FFmpeg binary verification completed successfully!"
            print_status "info" "The audio pipeline should be able to process streams."
        else
            exit_code=1
        fi
    else
        print_status "error" "FFmpeg binary not found or not functional"
        print_troubleshooting
        exit_code=1
    fi
    
    echo
    exit $exit_code
}

# Run main function
main "$@"