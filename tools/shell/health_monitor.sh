#!/bin/bash

# health_monitor.sh - Comprehensive health monitoring for Discord Bot V2
# Usage: ./health_monitor.sh [--alert] [--verbose]

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="${PROJECT_ROOT}/config/audio.yaml"
ENV_FILE="${PROJECT_ROOT}/.env"

# Load environment variables
if [[ -f "$ENV_FILE" ]]; then
    source "$ENV_FILE"
fi

# Default values
DB_HOST="${DB_HOST:-localhost}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-discord_bot}"
HEALTH_CHECK_PORT="${HEALTH_CHECK_PORT:-8080}"
ERROR_THRESHOLD_WARNING="${ERROR_THRESHOLD_WARNING:-5}"
ERROR_THRESHOLD_CRITICAL="${ERROR_THRESHOLD_CRITICAL:-15}"

# Flags
ALERT_MODE=false
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --alert)
            ALERT_MODE=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--alert] [--verbose]"
            echo "  --alert   Exit with error code on critical issues"
            echo "  --verbose Show detailed output"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Logging functions
log_info() {
    echo "[INFO] $1"
}

log_warn() {
    echo "[WARN] $1" >&2
}

log_error() {
    echo "[ERROR] $1" >&2
}

log_verbose() {
    if [[ "$VERBOSE" == true ]]; then
        echo "[DEBUG] $1"
    fi
}

# Health check functions
check_process() {
    log_verbose "Checking process status..."
    
    if pgrep -f "discord-bot" > /dev/null; then
        log_info "✓ Discord bot process running"
        
        # Get process details
        local pid=$(pgrep -f "discord-bot")
        local memory=$(ps -p "$pid" -o %mem --no-headers | tr -d ' ')
        local cpu=$(ps -p "$pid" -o %cpu --no-headers | tr -d ' ')
        
        log_verbose "Process PID: $pid, Memory: ${memory}%, CPU: ${cpu}%"
        
        # Check memory usage
        if (( $(echo "$memory > 80" | bc -l) )); then
            log_warn "High memory usage: ${memory}%"
            if [[ "$ALERT_MODE" == true ]] && (( $(echo "$memory > 95" | bc -l) )); then
                log_error "Critical memory usage: ${memory}%"
                return 1
            fi
        fi
        
        return 0
    else
        log_error "✗ Discord bot process not running"
        return 1
    fi
}

check_database() {
    log_verbose "Checking database connectivity..."
    
    if command -v psql > /dev/null; then
        if psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
            log_info "✓ Database accessible"
            
            # Check connection count
            local connections=$(psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -t -c "
                SELECT count(*) FROM pg_stat_activity WHERE state = 'active';" 2>/dev/null | tr -d ' ')
            
            log_verbose "Active database connections: $connections"
            
            return 0
        else
            log_error "✗ Database connection failed"
            return 1
        fi
    else
        log_warn "psql not available, skipping database check"
        return 0
    fi
}

check_health_endpoint() {
    log_verbose "Checking health endpoint..."
    
    if command -v curl > /dev/null; then
        if curl -f -s "http://localhost:${HEALTH_CHECK_PORT}/health" > /dev/null 2>&1; then
            log_info "✓ Health endpoint responding"
            return 0
        else
            log_warn "Health endpoint not responding (may not be enabled)"
            return 0  # Non-critical
        fi
    else
        log_verbose "curl not available, skipping health endpoint check"
        return 0
    fi
}

check_recent_activity() {
    log_verbose "Checking recent activity..."
    
    if command -v psql > /dev/null; then
        local recent_logs=$(psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -t -c "
            SELECT COUNT(*) FROM audio_logs 
            WHERE timestamp > NOW() - INTERVAL '5 minutes';" 2>/dev/null | tr -d ' ')
        
        if [[ -n "$recent_logs" ]] && [[ "$recent_logs" -gt 0 ]]; then
            log_info "✓ Recent activity detected ($recent_logs logs)"
        else
            log_info "⚠ No recent activity (may be normal during quiet periods)"
        fi
        
        return 0
    else
        log_verbose "Cannot check recent activity without database access"
        return 0
    fi
}

check_error_rate() {
    log_verbose "Checking error rates..."
    
    if command -v psql > /dev/null; then
        local error_rate=$(psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -t -c "
            SELECT COALESCE(ROUND(
                (SELECT COUNT(*)::float FROM audio_logs WHERE level='ERROR' AND timestamp > NOW() - INTERVAL '1 hour') /
                NULLIF((SELECT COUNT(*)::float FROM audio_logs WHERE timestamp > NOW() - INTERVAL '1 hour'), 0) * 100, 2
            ), 0);" 2>/dev/null | tr -d ' ')
        
        if [[ -n "$error_rate" ]]; then
            local error_rate_int=${error_rate%.*}
            
            if [[ "$error_rate_int" -lt "$ERROR_THRESHOLD_WARNING" ]]; then
                log_info "✓ Error rate acceptable (${error_rate}%)"
            elif [[ "$error_rate_int" -lt "$ERROR_THRESHOLD_CRITICAL" ]]; then
                log_warn "⚠ Error rate elevated (${error_rate}%)"
            else
                log_error "✗ Error rate critical (${error_rate}%)"
                if [[ "$ALERT_MODE" == true ]]; then
                    return 1
                fi
            fi
        else
            log_verbose "Could not determine error rate"
        fi
        
        return 0
    else
        log_verbose "Cannot check error rate without database access"
        return 0
    fi
}

check_disk_space() {
    log_verbose "Checking disk space..."
    
    local usage=$(df / | awk 'NR==2 {print $5}' | sed 's/%//')
    
    if [[ "$usage" -lt 80 ]]; then
        log_info "✓ Disk space sufficient (${usage}% used)"
    elif [[ "$usage" -lt 95 ]]; then
        log_warn "⚠ Disk space low (${usage}% used)"
    else
        log_error "✗ Disk space critical (${usage}% used)"
        if [[ "$ALERT_MODE" == true ]]; then
            return 1
        fi
    fi
    
    return 0
}

check_dependencies() {
    log_verbose "Checking external dependencies..."
    
    local deps_ok=true
    
    # Check FFmpeg
    if command -v ffmpeg > /dev/null; then
        log_verbose "✓ FFmpeg available"
    else
        log_warn "⚠ FFmpeg not found in PATH"
        deps_ok=false
    fi
    
    # Check yt-dlp
    if command -v yt-dlp > /dev/null; then
        log_verbose "✓ yt-dlp available"
    else
        log_warn "⚠ yt-dlp not found in PATH"
        deps_ok=false
    fi
    
    if [[ "$deps_ok" == true ]]; then
        log_info "✓ External dependencies available"
    else
        log_warn "Some dependencies missing (may cause audio issues)"
    fi
    
    return 0
}

generate_summary_report() {
    log_verbose "Generating summary report..."
    
    if command -v psql > /dev/null; then
        echo ""
        echo "=== System Summary (Last 24 Hours) ==="
        
        # Log volume by component
        echo "Log Volume by Component:"
        psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c "
            SELECT 
                component,
                COUNT(*) as log_count,
                MAX(timestamp) as latest_log
            FROM audio_logs 
            WHERE timestamp > NOW() - INTERVAL '24 hours'
            GROUP BY component
            ORDER BY log_count DESC;" 2>/dev/null || echo "Could not retrieve log summary"
        
        # Error summary
        echo ""
        echo "Error Summary:"
        psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c "
            SELECT 
                level,
                component,
                COUNT(*) as count
            FROM audio_logs 
            WHERE level IN ('ERROR', 'WARN')
            AND timestamp > NOW() - INTERVAL '24 hours'
            GROUP BY level, component
            ORDER BY count DESC;" 2>/dev/null || echo "Could not retrieve error summary"
        
        # Performance metrics
        echo ""
        echo "Performance Metrics:"
        psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c "
            SELECT 
                'Startup Time' as metric,
                ROUND(AVG(CAST(fields->>'value' AS FLOAT)), 2) as avg_value,
                'seconds' as unit
            FROM audio_logs 
            WHERE message LIKE '%startup_time%'
            AND timestamp > NOW() - INTERVAL '24 hours'
            UNION ALL
            SELECT 
                'Total Playbacks' as metric,
                COUNT(*)::text as avg_value,
                'count' as unit
            FROM audio_logs 
            WHERE message LIKE '%playback%started%'
            AND timestamp > NOW() - INTERVAL '24 hours';" 2>/dev/null || echo "Could not retrieve performance metrics"
    fi
}

# Main execution
main() {
    echo "=== Discord Bot Health Check ==="
    echo "Timestamp: $(date)"
    echo ""
    
    local exit_code=0
    
    # Run all checks
    check_process || exit_code=1
    check_database || exit_code=1
    check_health_endpoint || true  # Non-critical
    check_recent_activity || true  # Non-critical
    check_error_rate || exit_code=1
    check_disk_space || exit_code=1
    check_dependencies || true  # Non-critical
    
    # Generate detailed report if verbose
    if [[ "$VERBOSE" == true ]]; then
        generate_summary_report
    fi
    
    echo ""
    if [[ "$exit_code" -eq 0 ]]; then
        echo "=== Health Check PASSED ==="
    else
        echo "=== Health Check FAILED ==="
        if [[ "$ALERT_MODE" == true ]]; then
            echo "Critical issues detected. Check logs and consider rollback if necessary."
        fi
    fi
    
    exit $exit_code
}

# Run main function
main "$@"