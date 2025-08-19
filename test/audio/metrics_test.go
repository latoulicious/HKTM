package audio_test

import (
	"errors"
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/pkg/database/models"
)

// MockAudioRepository for metrics testing
type MockMetricsRepository struct {
	savedMetrics    []*models.AudioMetric
	savedErrors     []*models.AudioError
	savedLogs       []*models.AudioLog
	saveMetricErr   error
	saveErrorErr    error
	saveLogErr      error
	metricsStats    *audio.MetricsStats
	getStatsErr     error
}

func NewMockMetricsRepository() *MockMetricsRepository {
	return &MockMetricsRepository{
		metricsStats: &audio.MetricsStats{
			SuccessfulPlays:      0,
			ErrorCount:           0,
			AverageStartupTime:   0,
			TotalPlaybackTime:    0,
		},
	}
}

func (m *MockMetricsRepository) SaveMetric(metric *models.AudioMetric) error {
	if m.saveMetricErr != nil {
		return m.saveMetricErr
	}
	m.savedMetrics = append(m.savedMetrics, metric)
	return nil
}

func (m *MockMetricsRepository) SaveError(error *models.AudioError) error {
	if m.saveErrorErr != nil {
		return m.saveErrorErr
	}
	m.savedErrors = append(m.savedErrors, error)
	return nil
}

func (m *MockMetricsRepository) SaveLog(log *models.AudioLog) error {
	if m.saveLogErr != nil {
		return m.saveLogErr
	}
	m.savedLogs = append(m.savedLogs, log)
	return nil
}

func (m *MockMetricsRepository) GetErrorStats(guildID string) (*audio.ErrorStats, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetMetricsStats(guildID string) (*audio.MetricsStats, error) {
	if m.getStatsErr != nil {
		return nil, m.getStatsErr
	}
	return m.metricsStats, nil
}

// Test BasicMetrics functionality

func TestBasicMetrics_RecordStartupTime(t *testing.T) {
	repo := NewMockMetricsRepository()
	repo.getStatsErr = errors.New("database unavailable") // Force fallback to in-memory stats
	guildID := "test-guild-123"
	metrics := audio.NewBasicMetrics(repo, guildID)
	
	// Record startup time
	startupTime := 2500 * time.Millisecond
	metrics.RecordStartupTime(startupTime)
	
	// Verify metric was saved to repository
	if len(repo.savedMetrics) != 1 {
		t.Errorf("Expected 1 saved metric, got %d", len(repo.savedMetrics))
	}
	
	savedMetric := repo.savedMetrics[0]
	if savedMetric.GuildID != guildID {
		t.Errorf("Expected guild ID %s, got %s", guildID, savedMetric.GuildID)
	}
	
	if savedMetric.MetricType != "startup_time" {
		t.Errorf("Expected metric type 'startup_time', got %s", savedMetric.MetricType)
	}
	
	if savedMetric.Value != startupTime.Seconds() {
		t.Errorf("Expected value %f, got %f", startupTime.Seconds(), savedMetric.Value)
	}
	
	// Verify stats are updated (should use in-memory stats)
	stats := metrics.GetStats()
	if stats.AverageStartupTime == 0 {
		t.Error("Expected average startup time to be recorded")
	}
}

func TestBasicMetrics_RecordStartupTime_RepositoryFailure(t *testing.T) {
	repo := NewMockMetricsRepository()
	repo.saveMetricErr = errors.New("database connection failed")
	
	guildID := "test-guild-123"
	metrics := audio.NewBasicMetrics(repo, guildID)
	
	// Record startup time (should not panic on repository failure)
	startupTime := 2500 * time.Millisecond
	metrics.RecordStartupTime(startupTime)
	
	// Verify in-memory tracking still works
	stats := metrics.GetStats()
	if stats.AverageStartupTime == 0 {
		t.Error("Expected startup time to be tracked in memory despite repository failure")
	}
}

func TestBasicMetrics_RecordError(t *testing.T) {
	repo := NewMockMetricsRepository()
	repo.getStatsErr = errors.New("database unavailable") // Force fallback to in-memory stats
	guildID := "test-guild-123"
	metrics := audio.NewBasicMetrics(repo, guildID)
	
	// Record different types of errors
	metrics.RecordError("network")
	metrics.RecordError("ffmpeg")
	metrics.RecordError("network") // Duplicate to test counting
	
	// Verify metrics were saved to repository
	if len(repo.savedMetrics) != 3 {
		t.Errorf("Expected 3 saved metrics, got %d", len(repo.savedMetrics))
	}
	
	// Verify errors were saved to repository
	if len(repo.savedErrors) != 3 {
		t.Errorf("Expected 3 saved errors, got %d", len(repo.savedErrors))
	}
	
	// Verify error counting through stats (should use in-memory stats)
	stats := metrics.GetStats()
	if stats.ErrorCount != 3 {
		t.Errorf("Expected 3 total errors, got %d", stats.ErrorCount)
	}
}

func TestBasicMetrics_RecordPlaybackDuration(t *testing.T) {
	repo := NewMockMetricsRepository()
	repo.getStatsErr = errors.New("database unavailable") // Force fallback to in-memory stats
	guildID := "test-guild-123"
	metrics := audio.NewBasicMetrics(repo, guildID)
	
	// Record playback durations
	duration1 := 3*time.Minute + 30*time.Second
	duration2 := 4*time.Minute + 15*time.Second
	
	metrics.RecordPlaybackDuration(duration1)
	metrics.RecordPlaybackDuration(duration2)
	
	// Verify metrics were saved to repository
	if len(repo.savedMetrics) != 2 {
		t.Errorf("Expected 2 saved metrics, got %d", len(repo.savedMetrics))
	}
	
	// Verify playback tracking through stats (should use in-memory stats)
	stats := metrics.GetStats()
	if stats.SuccessfulPlays != 2 {
		t.Errorf("Expected 2 successful plays, got %d", stats.SuccessfulPlays)
	}
	
	expectedTotal := duration1 + duration2
	if stats.TotalPlaybackTime != expectedTotal {
		t.Errorf("Expected total playback time %v, got %v", expectedTotal, stats.TotalPlaybackTime)
	}
}

func TestBasicMetrics_GetStats_InMemoryOnly(t *testing.T) {
	repo := NewMockMetricsRepository()
	repo.getStatsErr = errors.New("database unavailable")
	
	guildID := "test-guild-123"
	metrics := audio.NewBasicMetrics(repo, guildID)
	
	// Record some metrics
	metrics.RecordStartupTime(2 * time.Second)
	metrics.RecordStartupTime(3 * time.Second)
	metrics.RecordPlaybackDuration(5 * time.Minute)
	metrics.RecordError("network")
	
	// Get stats (should fall back to in-memory data)
	stats := metrics.GetStats()
	
	if stats.SuccessfulPlays != 1 {
		t.Errorf("Expected 1 successful play, got %d", stats.SuccessfulPlays)
	}
	
	if stats.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", stats.ErrorCount)
	}
	
	expectedAvgStartup := 2500 * time.Millisecond // (2s + 3s) / 2
	if stats.AverageStartupTime != expectedAvgStartup {
		t.Errorf("Expected average startup time %v, got %v", expectedAvgStartup, stats.AverageStartupTime)
	}
	
	if stats.TotalPlaybackTime != 5*time.Minute {
		t.Errorf("Expected total playback time %v, got %v", 5*time.Minute, stats.TotalPlaybackTime)
	}
}

func TestBasicMetrics_GetStats_DatabasePreferred(t *testing.T) {
	repo := NewMockMetricsRepository()
	// Set up database stats that are different from in-memory
	repo.metricsStats = &audio.MetricsStats{
		SuccessfulPlays:      10,
		ErrorCount:           2,
		AverageStartupTime:   3 * time.Second,
		TotalPlaybackTime:    30 * time.Minute,
	}
	
	guildID := "test-guild-123"
	metrics := audio.NewBasicMetrics(repo, guildID)
	
	// Record some in-memory metrics
	metrics.RecordStartupTime(1 * time.Second)
	metrics.RecordPlaybackDuration(2 * time.Minute)
	
	// Get stats (should prefer database data)
	stats := metrics.GetStats()
	
	// Should match the mock repository data, not in-memory data
	if stats.SuccessfulPlays != 10 {
		t.Errorf("Expected 10 successful plays from database, got %d", stats.SuccessfulPlays)
	}
	
	if stats.ErrorCount != 2 {
		t.Errorf("Expected 2 errors from database, got %d", stats.ErrorCount)
	}
	
	if stats.AverageStartupTime != 3*time.Second {
		t.Errorf("Expected average startup time from database %v, got %v", 3*time.Second, stats.AverageStartupTime)
	}
	
	if stats.TotalPlaybackTime != 30*time.Minute {
		t.Errorf("Expected total playback time from database %v, got %v", 30*time.Minute, stats.TotalPlaybackTime)
	}
}

func TestBasicMetrics_InterfaceCompliance(t *testing.T) {
	repo := NewMockMetricsRepository()
	repo.getStatsErr = errors.New("database unavailable") // Force fallback to in-memory stats
	guildID := "test-guild-123"
	
	// Test that NewBasicMetrics returns a MetricsCollector interface
	var metrics audio.MetricsCollector = audio.NewBasicMetrics(repo, guildID)
	
	// Test all interface methods
	metrics.RecordStartupTime(2 * time.Second)
	metrics.RecordError("network")
	metrics.RecordPlaybackDuration(3 * time.Minute)
	
	stats := metrics.GetStats()
	if stats.SuccessfulPlays == 0 {
		t.Error("Expected successful plays to be recorded")
	}
	
	if stats.ErrorCount == 0 {
		t.Error("Expected errors to be recorded")
	}
}

func TestBasicMetrics_ConcurrentAccess(t *testing.T) {
	repo := NewMockMetricsRepository()
	repo.getStatsErr = errors.New("database unavailable") // Force fallback to in-memory stats
	guildID := "test-guild-123"
	metrics := audio.NewBasicMetrics(repo, guildID)
	
	// Test concurrent recording (should not panic or race)
	done := make(chan bool, 3)
	
	// Goroutine 1: Record startup times
	go func() {
		for i := 0; i < 100; i++ {
			metrics.RecordStartupTime(time.Duration(i) * time.Millisecond)
		}
		done <- true
	}()
	
	// Goroutine 2: Record errors
	go func() {
		for i := 0; i < 100; i++ {
			metrics.RecordError("network")
		}
		done <- true
	}()
	
	// Goroutine 3: Read stats
	go func() {
		for i := 0; i < 100; i++ {
			metrics.GetStats()
		}
		done <- true
	}()
	
	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
	
	// Verify final state is consistent
	stats := metrics.GetStats()
	if stats.ErrorCount == 0 {
		t.Error("Expected some errors to be recorded")
	}
	
	if stats.AverageStartupTime == 0 {
		t.Error("Expected some startup times to be recorded")
	}
}