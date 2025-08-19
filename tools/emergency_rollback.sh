#!/bin/bash

# emergency_rollback.sh - Quick rollback script for Discord Bot V2
# Usage: ./emergency_rollback.sh [--confirm] [--full]

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="${PROJECT_ROOT}/config/audio.yaml"
ENV_FILE="${PROJECT_ROOT}/.env"
BACKUP_DIR="${PROJECT_ROOT}/backups"

# Load environment variables
if [[ -f "$ENV_FILE" ]]; then
    source "$ENV_FILE"
fi

# Default values
DB_HOST="${DB_HOST:-localhost}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-discord_bot}"
SERVICE_NAME="${SERVICE_NAME:-discord-bot}"

# Flags
CONFIRM=false
FULL_ROLLBACK=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --confirm)
            CONFIRM=true
            shift
            ;;
        --full)
            FULL_ROLLBACK=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--confirm] [--full]"
            echo "  --confirm  Skip confirmation prompts (use with caution)"
            echo "  --full     Perform full rollback including database"
            echo ""
            echo "Rollback modes:"
            echo "  Default: Disable V2 pipeline, keep V2 code"
            echo "  --full:  Restore V1 code and database backup"
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

# Confirmation function
confirm_action() {
    if [[ "$CONFIRM" == true ]]; then
        return 0
    fi
    
    echo -n "$1 (y/N): "
    read -r response
    case "$response" in
        [yY][eE][sS]|[yY])
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Backup current state before rollback
create_rollback_backup() {
    log_info "Creating rollback backup..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local rollback_backup_dir="${BACKUP_DIR}/rollback_${timestamp}"
    
    mkdir -p "$rollback_backup_dir"
    
    # Backup current config
    if [[ -f "$CONFIG_FILE" ]]; then
        cp "$CONFIG_FILE" "${rollback_backup_dir}/audio.yaml"
        log_info "Config backed up to ${rollback_backup_dir}/audio.yaml"
    fi
    
    # Backup current env
    if [[ -f "$ENV_FILE" ]]; then
        cp "$ENV_FILE" "${rollback_backup_dir}/.env"
        log_info "Environment backed up to ${rollback_backup_dir}/.env"
    fi
    
    # Backup current binary if exists
    if command -v systemctl > /dev/null && systemctl is-active "$SERVICE_NAME" > /dev/null 2>&1; then
        local binary_path=$(systemctl show "$SERVICE_NAME" -p ExecStart --value | awk '{print $1}')
        if [[ -f "$binary_path" ]]; then
            cp "$binary_path" "${rollback_backup_dir}/discord-bot-v2"
            log_info "Binary backed up to ${rollback_backup_dir}/discord-bot-v2"
        fi
    fi
    
    echo "$rollback_backup_dir" > "${BACKUP_DIR}/latest_rollback_backup"
    log_info "Rollback backup created: $rollback_backup_dir"
}

# Quick rollback - disable V2 pipeline only
quick_rollback() {
    log_info "Performing quick rollback (disable V2 pipeline)..."
    
    # Method 1: Update config file
    if [[ -f "$CONFIG_FILE" ]]; then
        log_info "Updating configuration to disable V2..."
        
        # Backup current config
        cp "$CONFIG_FILE" "${CONFIG_FILE}.pre-rollback"
        
        # Disable V2 pipeline
        if command -v yq > /dev/null; then
            yq eval '.pipeline.version = "v1"' -i "$CONFIG_FILE"
            yq eval '.pipeline.fallback_to_v1 = true' -i "$CONFIG_FILE"
        else
            # Fallback to sed
            sed -i 's/version: "v2"/version: "v1"/' "$CONFIG_FILE"
            sed -i 's/fallback_to_v1: false/fallback_to_v1: true/' "$CONFIG_FILE"
        fi
        
        log_info "Configuration updated to use V1 pipeline"
    fi
    
    # Method 2: Update environment variables
    if [[ -f "$ENV_FILE" ]]; then
        log_info "Updating environment variables..."
        
        # Backup current env
        cp "$ENV_FILE" "${ENV_FILE}.pre-rollback"
        
        # Update environment
        sed -i 's/AUDIO_PIPELINE_VERSION=v2/AUDIO_PIPELINE_VERSION=v1/' "$ENV_FILE"
        
        # Add rollback flag
        if ! grep -q "EMERGENCY_ROLLBACK" "$ENV_FILE"; then
            echo "EMERGENCY_ROLLBACK=true" >> "$ENV_FILE"
            echo "ROLLBACK_TIMESTAMP=$(date -Iseconds)" >> "$ENV_FILE"
        fi
        
        log_info "Environment variables updated"
    fi
    
    # Method 3: Database flag (if available)
    if command -v psql > /dev/null; then
        log_info "Setting database rollback flag..."
        
        psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c "
            INSERT INTO system_config (key, value, updated_at) 
            VALUES ('pipeline_v2_enabled', 'false', NOW())
            ON CONFLICT (key) 
            DO UPDATE SET value = 'false', updated_at = NOW();" 2>/dev/null || log_warn "Could not set database flag"
    fi
    
    # Restart service
    restart_service
}

# Full rollback - restore V1 code and database
full_rollback() {
    log_info "Performing full rollback (restore V1 code and database)..."
    
    # Find latest backup
    local latest_backup=""
    if [[ -f "${BACKUP_DIR}/latest_backup" ]]; then
        latest_backup=$(cat "${BACKUP_DIR}/latest_backup")
    else
        # Find most recent backup directory
        latest_backup=$(find "$BACKUP_DIR" -name "backup_*" -type d | sort | tail -1)
    fi
    
    if [[ -z "$latest_backup" ]] || [[ ! -d "$latest_backup" ]]; then
        log_error "No backup found for full rollback"
        log_error "Available backups:"
        find "$BACKUP_DIR" -name "backup_*" -type d | sort
        return 1
    fi
    
    log_info "Using backup: $latest_backup"
    
    # Stop service
    stop_service
    
    # Restore database
    if [[ -f "${latest_backup}/database.sql" ]]; then
        if confirm_action "Restore database from backup? This will overwrite current data"; then
            log_info "Restoring database..."
            psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" < "${latest_backup}/database.sql"
            log_info "Database restored"
        fi
    else
        log_warn "No database backup found in $latest_backup"
    fi
    
    # Restore configuration
    if [[ -f "${latest_backup}/audio.yaml" ]]; then
        cp "${latest_backup}/audio.yaml" "$CONFIG_FILE"
        log_info "Configuration restored"
    fi
    
    if [[ -f "${latest_backup}/.env" ]]; then
        cp "${latest_backup}/.env" "$ENV_FILE"
        log_info "Environment restored"
    fi
    
    # Restore binary
    if [[ -f "${latest_backup}/discord-bot" ]]; then
        local binary_path="/usr/local/bin/discord-bot"
        if command -v systemctl > /dev/null; then
            binary_path=$(systemctl show "$SERVICE_NAME" -p ExecStart --value | awk '{print $1}' 2>/dev/null || echo "/usr/local/bin/discord-bot")
        fi
        
        if confirm_action "Restore binary to $binary_path"; then
            sudo cp "${latest_backup}/discord-bot" "$binary_path"
            sudo chmod +x "$binary_path"
            log_info "Binary restored"
        fi
    fi
    
    # Start service
    start_service
}

# Service management functions
stop_service() {
    log_info "Stopping $SERVICE_NAME service..."
    
    if command -v systemctl > /dev/null; then
        if systemctl is-active "$SERVICE_NAME" > /dev/null 2>&1; then
            sudo systemctl stop "$SERVICE_NAME"
            log_info "Service stopped"
        else
            log_info "Service was not running"
        fi
    elif command -v docker-compose > /dev/null && [[ -f "${PROJECT_ROOT}/docker-compose.yml" ]]; then
        cd "$PROJECT_ROOT"
        docker-compose stop
        log_info "Docker containers stopped"
    else
        log_warn "Could not determine how to stop service"
    fi
}

start_service() {
    log_info "Starting $SERVICE_NAME service..."
    
    if command -v systemctl > /dev/null; then
        sudo systemctl start "$SERVICE_NAME"
        sleep 2
        if systemctl is-active "$SERVICE_NAME" > /dev/null 2>&1; then
            log_info "Service started successfully"
        else
            log_error "Service failed to start"
            return 1
        fi
    elif command -v docker-compose > /dev/null && [[ -f "${PROJECT_ROOT}/docker-compose.yml" ]]; then
        cd "$PROJECT_ROOT"
        docker-compose up -d
        log_info "Docker containers started"
    else
        log_warn "Could not determine how to start service"
    fi
}

restart_service() {
    log_info "Restarting $SERVICE_NAME service..."
    
    if command -v systemctl > /dev/null; then
        sudo systemctl restart "$SERVICE_NAME"
        sleep 2
        if systemctl is-active "$SERVICE_NAME" > /dev/null 2>&1; then
            log_info "Service restarted successfully"
        else
            log_error "Service failed to restart"
            return 1
        fi
    elif command -v docker-compose > /dev/null && [[ -f "${PROJECT_ROOT}/docker-compose.yml" ]]; then
        cd "$PROJECT_ROOT"
        docker-compose restart
        log_info "Docker containers restarted"
    else
        log_warn "Could not determine how to restart service"
    fi
}

# Verify rollback success
verify_rollback() {
    log_info "Verifying rollback success..."
    
    # Wait for service to stabilize
    sleep 5
    
    # Check if service is running
    if command -v systemctl > /dev/null; then
        if systemctl is-active "$SERVICE_NAME" > /dev/null 2>&1; then
            log_info "✓ Service is running"
        else
            log_error "✗ Service is not running"
            return 1
        fi
    fi
    
    # Check database connectivity
    if command -v psql > /dev/null; then
        if psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
            log_info "✓ Database is accessible"
        else
            log_error "✗ Database is not accessible"
            return 1
        fi
    fi
    
    # Check for recent errors
    if command -v psql > /dev/null; then
        local recent_errors=$(psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -t -c "
            SELECT COUNT(*) FROM audio_logs 
            WHERE level = 'ERROR' 
            AND timestamp > NOW() - INTERVAL '2 minutes';" 2>/dev/null | tr -d ' ')
        
        if [[ -n "$recent_errors" ]] && [[ "$recent_errors" -gt 0 ]]; then
            log_warn "⚠ $recent_errors recent errors detected"
        else
            log_info "✓ No recent errors"
        fi
    fi
    
    log_info "Rollback verification complete"
    return 0
}

# Main execution
main() {
    echo "=== Emergency Rollback Script ==="
    echo "Timestamp: $(date)"
    echo ""
    
    # Ensure backup directory exists
    mkdir -p "$BACKUP_DIR"
    
    # Show current status
    log_info "Current system status:"
    if command -v systemctl > /dev/null && systemctl is-active "$SERVICE_NAME" > /dev/null 2>&1; then
        echo "  Service: Running"
    else
        echo "  Service: Stopped/Unknown"
    fi
    
    if [[ -f "$CONFIG_FILE" ]]; then
        local current_version=$(grep -o 'version: "[^"]*"' "$CONFIG_FILE" 2>/dev/null | cut -d'"' -f2 || echo "unknown")
        echo "  Pipeline Version: $current_version"
    fi
    
    echo ""
    
    # Confirm rollback
    if [[ "$FULL_ROLLBACK" == true ]]; then
        if ! confirm_action "Perform FULL rollback (restore V1 code and database)?"; then
            log_info "Rollback cancelled"
            exit 0
        fi
    else
        if ! confirm_action "Perform quick rollback (disable V2 pipeline)?"; then
            log_info "Rollback cancelled"
            exit 0
        fi
    fi
    
    # Create backup before rollback
    create_rollback_backup
    
    # Perform rollback
    if [[ "$FULL_ROLLBACK" == true ]]; then
        full_rollback
    else
        quick_rollback
    fi
    
    # Verify rollback
    if verify_rollback; then
        echo ""
        echo "=== Rollback Completed Successfully ==="
        log_info "System has been rolled back"
        log_info "Monitor the system for stability"
        log_info "Check logs: journalctl -u $SERVICE_NAME -f"
    else
        echo ""
        echo "=== Rollback Completed with Issues ==="
        log_warn "Some verification checks failed"
        log_warn "Manual intervention may be required"
    fi
    
    # Show next steps
    echo ""
    echo "Next Steps:"
    echo "1. Monitor system logs for errors"
    echo "2. Test basic functionality (Discord commands)"
    echo "3. Check health status: ./tools/health_monitor.sh --verbose"
    echo "4. If issues persist, consider full rollback: $0 --full"
}

# Run main function
main "$@"