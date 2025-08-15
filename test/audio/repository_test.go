package audio

// import (
// 	"testing"
// 	"time"

// 	"github.com/google/uuid"
// 	"github.com/latoulicious/HKTM/pkg/database/models"
// 	"gorm.io/driver/postgres"
// 	"gorm.io/gorm"
// )

// func setupTestDB(t *testing.T) *gorm.DB {
// 	// Use the provided PostgreSQL URL
// 	testDSN := "your psql url"

// 	db, err := gorm.Open(postgres.Open(testDSN), &gorm.Config{})
// 	if err != nil {
// 		t.Fatalf("Failed to connect to test database: %v", err)
// 	}

// 	// Auto-migrate the models
// 	err = db.AutoMigrate(&models.AudioError{}, &models.AudioMetric{}, &models.AudioLog{})
// 	if err != nil {
// 		t.Fatalf("Failed to migrate test database: %v", err)
// 	}

// 	// Clean up any existing test data
// 	db.Where("guild_id LIKE ?", "test-guild-%").Delete(&models.AudioError{})
// 	db.Where("guild_id LIKE ?", "test-guild-%").Delete(&models.AudioMetric{})
// 	db.Where("guild_id LIKE ?", "test-guild-%").Delete(&models.AudioLog{})

// 	return db
// }

// func TestAudioRepositoryImpl_SaveError(t *testing.T) {
// 	db := setupTestDB(t)
// 	repo := NewAudioRepository(db)

// 	audioError := &models.AudioError{
// 		ID:        uuid.New(),
// 		GuildID:   "test-guild-123",
// 		ErrorType: "ffmpeg_failure",
// 		ErrorMsg:  "FFmpeg process crashed",
// 		Context:   "Playing YouTube URL",
// 		Timestamp: time.Now(),
// 		Resolved:  false,
// 	}

// 	err := repo.SaveError(audioError)
// 	if err != nil {
// 		t.Errorf("SaveError failed: %v", err)
// 	}

// 	// Verify the error was saved
// 	var savedError models.AudioError
// 	result := db.First(&savedError, "guild_id = ?", "test-guild-123")
// 	if result.Error != nil {
// 		t.Errorf("Failed to retrieve saved error: %v", result.Error)
// 	}

// 	if savedError.ErrorType != "ffmpeg_failure" {
// 		t.Errorf("Expected error type 'ffmpeg_failure', got '%s'", savedError.ErrorType)
// 	}
// }

// func TestAudioRepositoryImpl_SaveMetric(t *testing.T) {
// 	db := setupTestDB(t)
// 	repo := NewAudioRepository(db)

// 	metric := &models.AudioMetric{
// 		ID:         uuid.New(),
// 		GuildID:    "test-guild-123",
// 		MetricType: "startup_time",
// 		Value:      2.5, // 2.5 seconds
// 		Timestamp:  time.Now(),
// 	}

// 	err := repo.SaveMetric(metric)
// 	if err != nil {
// 		t.Errorf("SaveMetric failed: %v", err)
// 	}

// 	// Verify the metric was saved
// 	var savedMetric models.AudioMetric
// 	result := db.First(&savedMetric, "guild_id = ?", "test-guild-123")
// 	if result.Error != nil {
// 		t.Errorf("Failed to retrieve saved metric: %v", result.Error)
// 	}

// 	if savedMetric.MetricType != "startup_time" {
// 		t.Errorf("Expected metric type 'startup_time', got '%s'", savedMetric.MetricType)
// 	}
// }

// func TestAudioRepositoryImpl_SaveLog(t *testing.T) {
// 	db := setupTestDB(t)
// 	repo := NewAudioRepository(db)

// 	log := &models.AudioLog{
// 		ID:        uuid.New(),
// 		GuildID:   "test-guild-123",
// 		Level:     "ERROR",
// 		Message:   "Failed to start audio stream",
// 		Error:     "connection timeout",
// 		Fields:    map[string]interface{}{"url": "https://youtube.com/watch?v=test"},
// 		Timestamp: time.Now(),
// 	}

// 	err := repo.SaveLog(log)
// 	if err != nil {
// 		t.Errorf("SaveLog failed: %v", err)
// 	}

// 	// Verify the log was saved by checking basic fields (skip JSONB field for now)
// 	var savedLog models.AudioLog
// 	result := db.Select("id", "guild_id", "level", "message", "error", "timestamp").First(&savedLog, "guild_id = ?", "test-guild-123")
// 	if result.Error != nil {
// 		t.Errorf("Failed to retrieve saved log: %v", result.Error)
// 	}

// 	if savedLog.Level != "ERROR" {
// 		t.Errorf("Expected log level 'ERROR', got '%s'", savedLog.Level)
// 	}

// 	if savedLog.Message != "Failed to start audio stream" {
// 		t.Errorf("Expected message 'Failed to start audio stream', got '%s'", savedLog.Message)
// 	}
// }

// func TestAudioRepositoryImpl_GetErrorStats(t *testing.T) {
// 	db := setupTestDB(t)
// 	repo := NewAudioRepository(db)

// 	// Create test errors
// 	errors := []*models.AudioError{
// 		{
// 			ID:        uuid.New(),
// 			GuildID:   "test-guild-123",
// 			ErrorType: "ffmpeg_failure",
// 			ErrorMsg:  "FFmpeg crashed",
// 			Timestamp: time.Now().Add(-1 * time.Hour),
// 		},
// 		{
// 			ID:        uuid.New(),
// 			GuildID:   "test-guild-123",
// 			ErrorType: "network_error",
// 			ErrorMsg:  "Connection timeout",
// 			Timestamp: time.Now().Add(-30 * time.Minute),
// 		},
// 		{
// 			ID:        uuid.New(),
// 			GuildID:   "test-guild-123",
// 			ErrorType: "ffmpeg_failure",
// 			ErrorMsg:  "Another FFmpeg crash",
// 			Timestamp: time.Now().Add(-10 * time.Minute),
// 		},
// 	}

// 	for _, err := range errors {
// 		if saveErr := repo.SaveError(err); saveErr != nil {
// 			t.Fatalf("Failed to save test error: %v", saveErr)
// 		}
// 	}

// 	// Get error stats
// 	stats, err := repo.GetErrorStats("test-guild-123")
// 	if err != nil {
// 		t.Errorf("GetErrorStats failed: %v", err)
// 	}

// 	if stats.TotalErrors != 3 {
// 		t.Errorf("Expected 3 total errors, got %d", stats.TotalErrors)
// 	}

// 	if stats.ErrorsByType["ffmpeg_failure"] != 2 {
// 		t.Errorf("Expected 2 ffmpeg_failure errors, got %d", stats.ErrorsByType["ffmpeg_failure"])
// 	}

// 	if stats.ErrorsByType["network_error"] != 1 {
// 		t.Errorf("Expected 1 network_error, got %d", stats.ErrorsByType["network_error"])
// 	}

// 	if len(stats.RecentErrors) != 3 {
// 		t.Errorf("Expected 3 recent errors, got %d", len(stats.RecentErrors))
// 	}
// }

// func TestAudioRepositoryImpl_GetMetricsStats(t *testing.T) {
// 	db := setupTestDB(t)
// 	repo := NewAudioRepository(db)

// 	// Create test metrics
// 	metrics := []*models.AudioMetric{
// 		{
// 			ID:         uuid.New(),
// 			GuildID:    "test-guild-123",
// 			MetricType: "startup_time",
// 			Value:      2.0, // 2 seconds
// 			Timestamp:  time.Now().Add(-1 * time.Hour),
// 		},
// 		{
// 			ID:         uuid.New(),
// 			GuildID:    "test-guild-123",
// 			MetricType: "startup_time",
// 			Value:      3.0, // 3 seconds
// 			Timestamp:  time.Now().Add(-30 * time.Minute),
// 		},
// 		{
// 			ID:         uuid.New(),
// 			GuildID:    "test-guild-123",
// 			MetricType: "playback_duration",
// 			Value:      180.0, // 3 minutes
// 			Timestamp:  time.Now().Add(-20 * time.Minute),
// 		},
// 		{
// 			ID:         uuid.New(),
// 			GuildID:    "test-guild-123",
// 			MetricType: "playback_duration",
// 			Value:      240.0, // 4 minutes
// 			Timestamp:  time.Now().Add(-10 * time.Minute),
// 		},
// 	}

// 	for _, metric := range metrics {
// 		if err := repo.SaveMetric(metric); err != nil {
// 			t.Fatalf("Failed to save test metric: %v", err)
// 		}
// 	}

// 	// Create a test error for error count
// 	testError := &models.AudioError{
// 		ID:        uuid.New(),
// 		GuildID:   "test-guild-123",
// 		ErrorType: "test_error",
// 		ErrorMsg:  "Test error",
// 		Timestamp: time.Now(),
// 	}
// 	if err := repo.SaveError(testError); err != nil {
// 		t.Fatalf("Failed to save test error: %v", err)
// 	}

// 	// Get metrics stats
// 	stats, err := repo.GetMetricsStats("test-guild-123")
// 	if err != nil {
// 		t.Errorf("GetMetricsStats failed: %v", err)
// 	}

// 	// Check total playback time (180 + 240 = 420 seconds = 7 minutes)
// 	expectedPlaybackTime := 7 * time.Minute
// 	if stats.TotalPlaybackTime != expectedPlaybackTime {
// 		t.Errorf("Expected total playback time %v, got %v", expectedPlaybackTime, stats.TotalPlaybackTime)
// 	}

// 	// Check average startup time (2 + 3 = 5, 5/2 = 2.5 seconds)
// 	expectedAvgStartup := 2*time.Second + 500*time.Millisecond
// 	if stats.AverageStartupTime != expectedAvgStartup {
// 		t.Errorf("Expected average startup time %v, got %v", expectedAvgStartup, stats.AverageStartupTime)
// 	}

// 	// Check successful plays count (2 playback_duration metrics)
// 	if stats.SuccessfulPlays != 2 {
// 		t.Errorf("Expected 2 successful plays, got %d", stats.SuccessfulPlays)
// 	}

// 	// Check error count
// 	if stats.ErrorCount != 1 {
// 		t.Errorf("Expected 1 error, got %d", stats.ErrorCount)
// 	}
// }
