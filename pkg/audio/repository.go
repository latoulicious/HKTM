package audio

import (
	"time"

	"github.com/latoulicious/HKTM/pkg/database/models"
	"gorm.io/gorm"
)

// AudioRepositoryImpl implements the AudioRepository interface using GORM
type AudioRepositoryImpl struct {
	db *gorm.DB
}

// NewAudioRepository creates a new AudioRepository implementation
func NewAudioRepository(db *gorm.DB) AudioRepository {
	return &AudioRepositoryImpl{
		db: db,
	}
}

// SaveError saves an audio error to the database
func (r *AudioRepositoryImpl) SaveError(audioError *models.AudioError) error {
	return r.db.Create(audioError).Error
}

// SaveMetric saves an audio metric to the database
func (r *AudioRepositoryImpl) SaveMetric(metric *models.AudioMetric) error {
	return r.db.Create(metric).Error
}

// SaveLog saves an audio log entry to the database
func (r *AudioRepositoryImpl) SaveLog(log *models.AudioLog) error {
	return r.db.Create(log).Error
}

// GetErrorStats retrieves aggregated error statistics for a guild
func (r *AudioRepositoryImpl) GetErrorStats(guildID string) (*ErrorStats, error) {
	stats := &ErrorStats{
		ErrorsByType: make(map[string]int),
	}

	// Get total error count
	var totalErrors int64
	if err := r.db.Model(&models.AudioError{}).
		Where("guild_id = ?", guildID).
		Count(&totalErrors).Error; err != nil {
		return nil, err
	}
	stats.TotalErrors = int(totalErrors)

	// Get errors by type
	var errorTypeCounts []struct {
		ErrorType string
		Count     int64
	}
	if err := r.db.Model(&models.AudioError{}).
		Select("error_type, COUNT(*) as count").
		Where("guild_id = ?", guildID).
		Group("error_type").
		Scan(&errorTypeCounts).Error; err != nil {
		return nil, err
	}

	for _, typeCount := range errorTypeCounts {
		stats.ErrorsByType[typeCount.ErrorType] = int(typeCount.Count)
	}

	// Get recent errors (last 24 hours) - actual error records
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	var recentErrors []models.AudioError
	if err := r.db.Where("guild_id = ? AND timestamp > ?", guildID, twentyFourHoursAgo).
		Order("timestamp DESC").
		Limit(10). // Limit to last 10 recent errors
		Find(&recentErrors).Error; err != nil {
		return nil, err
	}
	stats.RecentErrors = recentErrors

	// Get last error time
	var lastError models.AudioError
	if err := r.db.Where("guild_id = ?", guildID).
		Order("timestamp DESC").
		First(&lastError).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		// No errors found, use zero time
		stats.LastErrorTime = time.Time{}
	} else {
		stats.LastErrorTime = lastError.Timestamp
	}

	return stats, nil
}

// GetMetricsStats retrieves aggregated performance metrics for a guild
func (r *AudioRepositoryImpl) GetMetricsStats(guildID string) (*MetricsStats, error) {
	stats := &MetricsStats{}

	// Get total playback time (sum of all playback_duration metrics in seconds)
	var totalPlaybackTimeSeconds float64
	if err := r.db.Model(&models.AudioMetric{}).
		Select("COALESCE(SUM(value), 0)").
		Where("guild_id = ? AND metric_type = ?", guildID, "playback_duration").
		Scan(&totalPlaybackTimeSeconds).Error; err != nil {
		return nil, err
	}
	stats.TotalPlaybackTime = time.Duration(totalPlaybackTimeSeconds * float64(time.Second))

	// Get average startup time in seconds
	var avgStartupTimeSeconds float64
	if err := r.db.Model(&models.AudioMetric{}).
		Select("COALESCE(AVG(value), 0)").
		Where("guild_id = ? AND metric_type = ?", guildID, "startup_time").
		Scan(&avgStartupTimeSeconds).Error; err != nil {
		return nil, err
	}
	stats.AverageStartupTime = time.Duration(avgStartupTimeSeconds * float64(time.Second))

	// Get error count
	var errorCount int64
	if err := r.db.Model(&models.AudioError{}).
		Where("guild_id = ?", guildID).
		Count(&errorCount).Error; err != nil {
		return nil, err
	}
	stats.ErrorCount = int(errorCount)

	// Get successful plays count (assuming playback_duration metrics indicate successful plays)
	var successfulPlays int64
	if err := r.db.Model(&models.AudioMetric{}).
		Where("guild_id = ? AND metric_type = ?", guildID, "playback_duration").
		Count(&successfulPlays).Error; err != nil {
		return nil, err
	}
	stats.SuccessfulPlays = int(successfulPlays)

	return stats, nil
}
