package audio

import (
	"sync"
	"time"
)

// BasicMetrics implements the MetricsCollector interface
// It provides simple metrics collection with database persistence
type BasicMetrics struct {
	repository AudioRepository // Interface injection
	guildID    string

	// Simple in-memory counters for performance
	startupTimes  []time.Duration
	errorCounts   map[string]int
	playbackTimes []time.Duration
	mu            sync.RWMutex
}

// NewBasicMetrics creates a new BasicMetrics instance
func NewBasicMetrics(repository AudioRepository, guildID string) MetricsCollector {
	return &BasicMetrics{
		repository:    repository,
		guildID:       guildID,
		startupTimes:  make([]time.Duration, 0),
		errorCounts:   make(map[string]int),
		playbackTimes: make([]time.Duration, 0),
	}
}

// RecordStartupTime records how long it takes to start playing audio
func (m *BasicMetrics) RecordStartupTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to in-memory collection for quick stats
	m.startupTimes = append(m.startupTimes, duration)

	// Create metric using shared utility
	metric := CreateAudioMetric(m.guildID, "startup_time", duration.Seconds())

	// Delegate persistence to repository
	if err := m.repository.SaveMetric(metric); err != nil {
		// Don't fail metrics collection on storage errors
		// This prevents metrics from breaking the main pipeline
		// Use shared context creation utility for consistent logging
		fields := CreateContextFieldsWithComponent(m.guildID, "", "", "metrics")
		fields["metric_type"] = "startup_time"
		fields["value"] = duration.Seconds()
		fields["error"] = err.Error()

		// Note: We can't use logger here as it would create circular dependency
		// The error is logged but doesn't break the pipeline
	}
}

// RecordError counts errors by simple categories
func (m *BasicMetrics) RecordError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment in-memory counter
	m.errorCounts[errorType]++

	// Create error metric using shared utility
	metric := CreateAudioMetric(m.guildID, "error_count", 1.0)

	// Add error type to the metric context
	// We'll store this as a separate field in the database
	if err := m.repository.SaveMetric(metric); err != nil {
		// Don't fail metrics collection on storage errors
		fields := CreateContextFieldsWithComponent(m.guildID, "", "", "metrics")
		fields["metric_type"] = "error_count"
		fields["error_type"] = errorType
		fields["error"] = err.Error()
	}

	// Also create an AudioError record for detailed error tracking
	audioError := CreateAudioError(m.guildID, errorType, "Error recorded by metrics collector", "metrics")
	if err := m.repository.SaveError(audioError); err != nil {
		// Log but don't fail
		fields := CreateContextFieldsWithComponent(m.guildID, "", "", "metrics")
		fields["error_type"] = errorType
		fields["error"] = err.Error()
	}
}

// RecordPlaybackDuration records the duration of completed playback
func (m *BasicMetrics) RecordPlaybackDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to in-memory collection
	m.playbackTimes = append(m.playbackTimes, duration)

	// Create metric using shared utility
	metric := CreateAudioMetric(m.guildID, "playback_duration", duration.Seconds())

	// Delegate persistence to repository
	if err := m.repository.SaveMetric(metric); err != nil {
		// Don't fail metrics collection on storage errors
		fields := CreateContextFieldsWithComponent(m.guildID, "", "", "metrics")
		fields["metric_type"] = "playback_duration"
		fields["value"] = duration.Seconds()
		fields["error"] = err.Error()
	}
}

// GetStats returns basic aggregated statistics
func (m *BasicMetrics) GetStats() MetricsStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Start with in-memory stats as baseline
	stats := MetricsStats{
		SuccessfulPlays: len(m.playbackTimes),
		ErrorCount:      m.getTotalErrorCount(),
	}

	// Calculate average startup time from in-memory data
	if len(m.startupTimes) > 0 {
		var total time.Duration
		for _, duration := range m.startupTimes {
			total += duration
		}
		stats.AverageStartupTime = total / time.Duration(len(m.startupTimes))
	}

	// Calculate total playback time from in-memory data
	if len(m.playbackTimes) > 0 {
		var total time.Duration
		for _, duration := range m.playbackTimes {
			total += duration
		}
		stats.TotalPlaybackTime = total
	}

	// Try to get more comprehensive stats from database
	// Database stats are more accurate as they include historical data
	if dbStats, err := m.repository.GetMetricsStats(m.guildID); err == nil {
		// Merge database stats with in-memory stats
		// Database stats take precedence for historical accuracy
		stats.TotalPlaybackTime = dbStats.TotalPlaybackTime
		stats.AverageStartupTime = dbStats.AverageStartupTime
		stats.ErrorCount = dbStats.ErrorCount
		stats.SuccessfulPlays = dbStats.SuccessfulPlays
	}

	return stats
}

// GetErrorBreakdown returns detailed error statistics by type
func (m *BasicMetrics) GetErrorBreakdown() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy of the error counts to avoid race conditions
	breakdown := make(map[string]int)
	for errorType, count := range m.errorCounts {
		breakdown[errorType] = count
	}

	return breakdown
}

// GetRecentStartupTimes returns the last N startup times for trend analysis
func (m *BasicMetrics) GetRecentStartupTimes(limit int) []time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || len(m.startupTimes) == 0 {
		return []time.Duration{}
	}

	// Return the most recent startup times
	start := len(m.startupTimes) - limit
	if start < 0 {
		start = 0
	}

	// Create a copy to avoid race conditions
	recent := make([]time.Duration, len(m.startupTimes[start:]))
	copy(recent, m.startupTimes[start:])

	return recent
}

// GetAveragePlaybackDuration calculates the average duration of completed playbacks
func (m *BasicMetrics) GetAveragePlaybackDuration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.playbackTimes) == 0 {
		return 0
	}

	var total time.Duration
	for _, duration := range m.playbackTimes {
		total += duration
	}

	return total / time.Duration(len(m.playbackTimes))
}

// Reset clears all in-memory metrics (useful for testing or periodic cleanup)
func (m *BasicMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startupTimes = make([]time.Duration, 0)
	m.errorCounts = make(map[string]int)
	m.playbackTimes = make([]time.Duration, 0)
}

// getTotalErrorCount calculates total errors from in-memory counters
func (m *BasicMetrics) getTotalErrorCount() int {
	total := 0
	for _, count := range m.errorCounts {
		total += count
	}
	return total
}

// GetSuccessRate calculates the success rate as a percentage
func (m *BasicMetrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalAttempts := len(m.playbackTimes) + m.getTotalErrorCount()
	if totalAttempts == 0 {
		return 0.0
	}

	successfulPlays := len(m.playbackTimes)
	return (float64(successfulPlays) / float64(totalAttempts)) * 100.0
}

// GetMostCommonError returns the most frequently occurring error type
func (m *BasicMetrics) GetMostCommonError() (string, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.errorCounts) == 0 {
		return "", 0
	}

	var mostCommonError string
	var maxCount int

	for errorType, count := range m.errorCounts {
		if count > maxCount {
			maxCount = count
			mostCommonError = errorType
		}
	}

	return mostCommonError, maxCount
}

// IsHealthy returns true if the metrics indicate a healthy system
// This is a simple heuristic based on error rate and recent performance
func (m *BasicMetrics) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Consider system healthy if:
	// 1. Success rate is above 80%
	// 2. No single error type dominates (>50% of errors)
	// 3. Average startup time is reasonable (<10 seconds)

	successRate := m.GetSuccessRate()
	if successRate < 80.0 && len(m.playbackTimes) > 5 {
		return false
	}

	// Check if any single error type is too dominant
	totalErrors := m.getTotalErrorCount()
	if totalErrors > 10 {
		_, maxErrorCount := m.GetMostCommonError()
		if float64(maxErrorCount)/float64(totalErrors) > 0.5 {
			return false
		}
	}

	// Check average startup time
	if len(m.startupTimes) > 0 {
		var total time.Duration
		for _, duration := range m.startupTimes {
			total += duration
		}
		avgStartup := total / time.Duration(len(m.startupTimes))
		if avgStartup > 10*time.Second {
			return false
		}
	}

	return true
}
