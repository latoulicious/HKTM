#!/bin/bash

# Comprehensive Dependency Verification Script
# This script checks all critical dependencies for the Discord bot deployment

echo "=== Discord Bot Dependency Verification ==="
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

# Track overall status
overall_status=0

echo "🔍 Checking Database Dependencies..."
echo "=================================="
if ./tools/check_database.sh; then
    print_status "success" "Database dependencies verified"
else
    print_status "error" "Database dependency check failed"
    overall_status=1
fi

echo
echo "🔍 Checking Audio Binary Dependencies..."
echo "======================================="
if ./tools/check_ffmpeg.sh; then
    print_status "success" "Audio binary dependencies verified"
else
    print_status "error" "Audio binary dependency check failed"
    overall_status=1
fi

echo
echo "🔍 Running Comprehensive Configuration Check..."
echo "=============================================="
if go run tools/audio_binaries_check.go >/dev/null 2>&1; then
    print_status "success" "Configuration validation passed"
else
    print_status "warning" "Configuration validation had warnings (check details above)"
    # Don't fail overall status for configuration warnings
fi

echo
echo "📋 Summary"
echo "=========="
if [ $overall_status -eq 0 ]; then
    print_status "success" "All critical dependencies verified successfully!"
    print_status "info" "System is ready for Discord bot deployment"
else
    print_status "error" "Some dependency checks failed"
    print_status "info" "Please resolve the issues above before deployment"
fi

echo
exit $overall_status