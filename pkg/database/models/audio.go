package models

import (
	"time"

	"github.com/google/uuid"
)

// AudioError represents an error that occurred in the audio pipeline
type AudioError struct {
	ID        uuid.UUID `gorm:"primaryKey" json:"id"`
	GuildID   string    `gorm:"index;not null" json:"guild_id"`
	ErrorType string    `gorm:"index;not null" json:"error_type"`
	ErrorMsg  string    `gorm:"type:text;not null" json:"error_msg"`
	Context   string    `gorm:"type:text" json:"context"`
	Timestamp time.Time `gorm:"index;not null" json:"timestamp"`
	Resolved  bool      `gorm:"default:false" json:"resolved"`
}

// AudioMetric represents a performance metric from the audio pipeline
type AudioMetric struct {
	ID         uuid.UUID `gorm:"primaryKey" json:"id"`
	GuildID    string    `gorm:"index;not null" json:"guild_id"`
	MetricType string    `gorm:"index;not null" json:"metric_type"` // startup_time, error_count, playback_duration
	Value      float64   `gorm:"not null" json:"value"`
	Timestamp  time.Time `gorm:"index;not null" json:"timestamp"`
}

// AudioLog represents a log entry from the audio pipeline
type AudioLog struct {
	ID        uuid.UUID              `gorm:"primaryKey" json:"id"`
	GuildID   string                 `gorm:"index;not null" json:"guild_id"`
	Component string                 `gorm:"index;not null;default:'audio'" json:"component"` // "audio", "commands", "database", etc.
	Level     string                 `gorm:"index;not null" json:"level"`                     // INFO, ERROR, WARN, DEBUG
	Message   string                 `gorm:"type:text;not null" json:"message"`
	Error     string                 `gorm:"type:text" json:"error"`
	Fields    map[string]interface{} `gorm:"type:jsonb" json:"fields"`
	UserID    string                 `gorm:"index" json:"user_id"`    // Optional user context
	ChannelID string                 `gorm:"index" json:"channel_id"` // Optional channel context
	Timestamp time.Time              `gorm:"index;not null" json:"timestamp"`
}

// QueueTimeout tracks queue timeout information for guilds
type QueueTimeout struct {
	ID           uuid.UUID `gorm:"primaryKey" json:"id"`
	GuildID      string    `gorm:"uniqueIndex;not null" json:"guild_id"`
	ChannelID    string    `gorm:"not null" json:"channel_id"`
	LastActivity time.Time `gorm:"not null" json:"last_activity"`
	TimeoutAt    time.Time `gorm:"index;not null" json:"timeout_at"`
}

// TableName returns the table name for AudioError
func (AudioError) TableName() string {
	return "audio_errors"
}

// TableName returns the table name for AudioMetric
func (AudioMetric) TableName() string {
	return "audio_metrics"
}

// TableName returns the table name for AudioLog
func (AudioLog) TableName() string {
	return "audio_logs"
}

// TableName returns the table name for QueueTimeout
func (QueueTimeout) TableName() string {
	return "queue_timeouts"
}
