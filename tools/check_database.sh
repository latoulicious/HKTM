#!/bin/bash

# PostgreSQL Database Accessibility Check Script
# This script verifies that the PostgreSQL database is accessible and ready for deployment

set -e  # Exit on any error

echo "=== PostgreSQL Database Accessibility Check ==="
echo "Timestamp: $(date)"
echo ""

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
        "SUCCESS")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "ERROR")
            echo -e "${RED}❌ $message${NC}"
            ;;
        "WARNING")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "INFO")
            echo -e "${BLUE}ℹ️  $message${NC}"
            ;;
    esac
}

# Check if .env file exists and load DATABASE_URL
if [ -f ".env" ]; then
    print_status "SUCCESS" ".env file found"
    # Extract DATABASE_URL from .env file safely
    DATABASE_URL=$(grep -E "^DATABASE_URL=" .env | cut -d '=' -f2- | sed 's/^"//' | sed 's/"$//')
    export DATABASE_URL
else
    print_status "WARNING" ".env file not found, checking environment variables"
fi

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    print_status "ERROR" "DATABASE_URL environment variable is not set"
    echo "Please set DATABASE_URL in your .env file or environment"
    echo "Format: postgres://username:password@host:port/database?sslmode=require"
    exit 1
fi

print_status "SUCCESS" "DATABASE_URL is configured"

# Extract database connection details for validation
# Handle complex hostnames like Neon's format
DB_HOST=$(echo "$DATABASE_URL" | sed -n 's/.*@\([^\/]*\)\/.*/\1/p')
DB_PORT="5432"  # Default PostgreSQL port for most cloud providers
DB_NAME=$(echo "$DATABASE_URL" | sed -n 's/.*\/\([^?]*\).*/\1/p')

# Try to extract port if explicitly specified (Neon uses default port, so this may be empty)
if echo "$DATABASE_URL" | grep -q ":[0-9][0-9]*/" 2>/dev/null; then
    EXTRACTED_PORT=$(echo "$DATABASE_URL" | sed -n 's/.*:\([0-9][0-9]*\)\/.*/\1/p')
    if [ ! -z "$EXTRACTED_PORT" ]; then
        DB_PORT="$EXTRACTED_PORT"
    fi
fi

print_status "INFO" "Database host: $DB_HOST"
print_status "INFO" "Database port: $DB_PORT"
print_status "INFO" "Database name: $DB_NAME"

# Check if Go is available for running the detailed check
if command -v go &> /dev/null; then
    print_status "SUCCESS" "Go runtime available"
    
    # Check if we can build the database check tool
    if [ -f "tools/db_check.go" ]; then
        print_status "INFO" "Running detailed database connectivity check..."
        echo ""
        
        # Run the Go-based database check
        if go run tools/db_check.go; then
            print_status "SUCCESS" "Detailed database check passed"
        else
            print_status "ERROR" "Detailed database check failed"
            exit 1
        fi
    else
        print_status "WARNING" "Database check tool not found, running basic checks only"
    fi
else
    print_status "WARNING" "Go runtime not available, running basic checks only"
fi

# Basic connectivity test using psql if available
if command -v psql &> /dev/null; then
    print_status "INFO" "Testing basic connectivity with psql..."
    
    # Test connection with a simple query
    if psql "$DATABASE_URL" -c "SELECT 1;" > /dev/null 2>&1; then
        print_status "SUCCESS" "Basic psql connectivity test passed"
        
        # Check PostgreSQL version
        PG_VERSION=$(psql "$DATABASE_URL" -t -c "SELECT version();" 2>/dev/null | head -1 | xargs)
        if [ ! -z "$PG_VERSION" ]; then
            print_status "INFO" "PostgreSQL version: $PG_VERSION"
        fi
        
        # Check if uuid-ossp extension is available
        UUID_EXT=$(psql "$DATABASE_URL" -t -c "SELECT EXISTS(SELECT 1 FROM pg_available_extensions WHERE name = 'uuid-ossp');" 2>/dev/null | xargs)
        if [ "$UUID_EXT" = "t" ]; then
            print_status "SUCCESS" "uuid-ossp extension is available"
        else
            print_status "WARNING" "uuid-ossp extension not available - may need to install postgresql-contrib"
        fi
        
    else
        print_status "ERROR" "Basic psql connectivity test failed"
        echo "Please check your database connection settings and ensure the database is running"
        exit 1
    fi
else
    print_status "WARNING" "psql not available, skipping basic connectivity test"
fi

# Check if migration binary exists
if [ -f "./migration" ] || [ -f "./cmd/migration/main.go" ]; then
    print_status "SUCCESS" "Database migration tools available"
else
    print_status "WARNING" "Migration tools not found - ensure migrations can be run"
fi

# Check disk space (basic check)
AVAILABLE_SPACE=$(df . | tail -1 | awk '{print $4}')
if [ "$AVAILABLE_SPACE" -gt 1000000 ]; then  # More than 1GB
    print_status "SUCCESS" "Sufficient disk space available"
else
    print_status "WARNING" "Low disk space - ensure sufficient space for database operations"
fi

# Network connectivity test to database host
if command -v nc &> /dev/null && [ ! -z "$DB_HOST" ] && [ ! -z "$DB_PORT" ]; then
    print_status "INFO" "Testing network connectivity to database..."
    if nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
        print_status "SUCCESS" "Network connectivity to database host confirmed"
    else
        print_status "ERROR" "Cannot reach database host $DB_HOST:$DB_PORT"
        echo "Please check network connectivity and firewall settings"
        exit 1
    fi
else
    print_status "WARNING" "Network connectivity test skipped (nc not available or host/port not parsed)"
fi

echo ""
echo "=== Database Accessibility Check Summary ==="
print_status "SUCCESS" "PostgreSQL database accessibility verified"
echo ""
echo "Next steps:"
echo "1. Run database migrations: ./migration -migrate"
echo "2. Start the application"
echo "3. Monitor logs for any database-related issues"
echo ""
echo "For troubleshooting, check:"
echo "- Database server status"
echo "- Network connectivity"
echo "- Authentication credentials"
echo "- SSL/TLS configuration"
echo ""