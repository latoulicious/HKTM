# Database Migrations

This directory contains database migrations for the HKTM Discord bot.

## Centralized Logging Migration

### Overview
The centralized logging migration adds support for system-wide logging by extending the `audio_logs` table with additional columns:

- `component` - Identifies which system component generated the log (audio, commands, database, etc.)
- `user_id` - Optional user context for better tracking
- `channel_id` - Optional channel context for better tracking

### Files
- `20250119_add_centralized_logging_columns.go` - Main migration to add new columns and indexes
- `20250119_rollback_centralized_logging_columns.go` - Rollback migration to remove changes

### Migration Details

#### New Columns Added
```sql
ALTER TABLE audio_logs ADD COLUMN component VARCHAR(50) DEFAULT 'audio';
ALTER TABLE audio_logs ADD COLUMN user_id VARCHAR(50);
ALTER TABLE audio_logs ADD COLUMN channel_id VARCHAR(50);
```

#### Indexes Created
```sql
CREATE INDEX idx_audio_logs_component ON audio_logs(component);
CREATE INDEX idx_audio_logs_user_id ON audio_logs(user_id);
CREATE INDEX idx_audio_logs_channel_id ON audio_logs(channel_id);
CREATE INDEX idx_audio_logs_component_guild ON audio_logs(component, guild_id);
CREATE INDEX idx_audio_logs_user_guild ON audio_logs(user_id, guild_id) WHERE user_id IS NOT NULL;
```

### Running Migrations

Migrations are automatically run when the application starts via `migration.RunMigration(db)` in the main migration file.

### Rollback

If you need to rollback the centralized logging changes:

```go
import "github.com/latoulicious/HKTM/pkg/database/migration"

// In your rollback code
err := migration.RollbackCentralizedLoggingColumns(db)
if err != nil {
    log.Fatalf("Rollback failed: %v", err)
}
```

### Safety Features

- **Idempotent**: Migrations can be run multiple times safely
- **Column Checks**: Migrations check if columns exist before adding them
- **Default Values**: Existing records get appropriate default values
- **Index Safety**: Uses `IF NOT EXISTS` to prevent duplicate index errors

### Requirements Satisfied

This migration satisfies requirements:
- **8.5**: Centralized logging system with component categorization
- **8.6**: Enhanced context tracking with user and channel information