package common

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/pkg/embed"
	"github.com/latoulicious/HKTM/pkg/logging"
	"gorm.io/gorm"
)

// QueueItem represents a single item in the music queue
type QueueItem struct {
	URL         string // Stream URL
	OriginalURL string // Original YouTube URL (if applicable)
	VideoID     string // YouTube video ID (if applicable)
	Title       string
	RequestedBy string
	AddedAt     time.Time
	StartedAt   time.Time
	Duration    time.Duration
}

// MusicQueue manages the queue for a specific guild
type MusicQueue struct {
	guildID      string
	items        []*QueueItem
	current      *QueueItem
	isPlaying    bool
	wasSkipped   bool // Flag to track if current song was skipped
	mu           sync.RWMutex
	voiceConn    *discordgo.VoiceConnection
	pipeline     audio.AudioPipeline // Updated to use new AudioPipeline interface
	logger       logging.Logger      // Centralized logging
	db           *gorm.DB           // Database connection for pipeline creation
	embedBuilder embed.AudioEmbedBuilder // Centralized embeds for queue status
}

// NewMusicQueue creates a new music queue for a guild
func NewMusicQueue(guildID string) *MusicQueue {
	// Create centralized logger for queue operations
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateQueueLogger(guildID)

	return &MusicQueue{
		guildID:      guildID,
		items:        make([]*QueueItem, 0),
		logger:       logger,
		embedBuilder: embed.GetGlobalAudioEmbedBuilder(),
	}
}

// NewMusicQueueWithDB creates a new music queue with database connection for pipeline creation
func NewMusicQueueWithDB(guildID string, db *gorm.DB) *MusicQueue {
	// Create centralized logger for queue operations
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateQueueLogger(guildID)

	return &MusicQueue{
		guildID:      guildID,
		items:        make([]*QueueItem, 0),
		logger:       logger,
		db:           db,
		embedBuilder: embed.GetGlobalAudioEmbedBuilder(),
	}
}

// Add adds a new item to the queue
func (mq *MusicQueue) Add(url, title, requestedBy string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	item := &QueueItem{
		URL:         url,
		Title:       title,
		RequestedBy: requestedBy,
		AddedAt:     time.Now(),
	}

	mq.items = append(mq.items, item)
	
	// Use centralized logging
	if mq.logger != nil {
		mq.logger.Info("Added item to queue", map[string]interface{}{
			"title":        title,
			"url":          url,
			"requested_by": requestedBy,
			"queue_size":   len(mq.items),
		})
	} else {
		// Fallback to standard logging if centralized logger not available
		log.Printf("Added '%s' to queue for guild %s", title, mq.guildID)
	}
}

// AddWithYouTubeData adds a new item to the queue with YouTube-specific data
func (mq *MusicQueue) AddWithYouTubeData(url, originalURL, videoID, title, requestedBy string, duration time.Duration) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	item := &QueueItem{
		URL:         url,
		OriginalURL: originalURL,
		VideoID:     videoID,
		Title:       title,
		RequestedBy: requestedBy,
		AddedAt:     time.Now(),
		Duration:    duration,
	}

	mq.items = append(mq.items, item)
	
	// Use centralized logging
	if mq.logger != nil {
		mq.logger.Info("Added YouTube item to queue", map[string]interface{}{
			"title":        title,
			"url":          url,
			"original_url": originalURL,
			"video_id":     videoID,
			"requested_by": requestedBy,
			"duration":     duration.String(),
			"queue_size":   len(mq.items),
		})
	} else {
		// Fallback to standard logging if centralized logger not available
		log.Printf("Added '%s' (Duration: %v) to queue for guild %s", title, duration, mq.guildID)
	}
}

// Next gets the next item from the queue
func (mq *MusicQueue) Next() *QueueItem {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.items) == 0 {
		return nil
	}

	item := mq.items[0]
	mq.items = mq.items[1:]
	mq.current = item
	return item
}

// Current returns the currently playing item
func (mq *MusicQueue) Current() *QueueItem {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.current
}

// List returns all items in the queue
func (mq *MusicQueue) List() []*QueueItem {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	result := make([]*QueueItem, len(mq.items))
	copy(result, mq.items)
	return result
}

// Size returns the number of items in the queue
func (mq *MusicQueue) Size() int {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return len(mq.items)
}

// Clear clears the entire queue
func (mq *MusicQueue) Clear() {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	
	queueSize := len(mq.items)
	mq.items = make([]*QueueItem, 0)
	mq.current = nil
	
	// Use centralized logging
	if mq.logger != nil {
		mq.logger.Info("Cleared queue", map[string]interface{}{
			"items_cleared": queueSize,
		})
	}
}

// Remove removes an item at the specified index
func (mq *MusicQueue) Remove(index int) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if index < 0 || index >= len(mq.items) {
		return fmt.Errorf("invalid index: %d", index)
	}

	removed := mq.items[index]
	mq.items = append(mq.items[:index], mq.items[index+1:]...)
	
	// Use centralized logging
	if mq.logger != nil {
		mq.logger.Info("Removed item from queue", map[string]interface{}{
			"title":      removed.Title,
			"index":      index,
			"queue_size": len(mq.items),
		})
	} else {
		// Fallback to standard logging if centralized logger not available
		log.Printf("Removed '%s' from queue for guild %s", removed.Title, mq.guildID)
	}
	return nil
}

// SetPlaying sets the playing state
func (mq *MusicQueue) SetPlaying(playing bool) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.isPlaying = playing
}

// IsPlaying returns whether something is currently playing
func (mq *MusicQueue) IsPlaying() bool {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.isPlaying
}

// HasActivePipeline returns whether there's an active pipeline
func (mq *MusicQueue) HasActivePipeline() bool {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.pipeline != nil && mq.pipeline.IsPlaying()
}

// IsCurrentlyPlaying returns whether there's actually active playback
func (mq *MusicQueue) IsCurrentlyPlaying() bool {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.isPlaying && mq.pipeline != nil && mq.pipeline.IsPlaying()
}

// CanStartPlaying returns whether the queue is in a valid state to start playing
func (mq *MusicQueue) CanStartPlaying() bool {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	// Can start if not currently playing and either no pipeline or pipeline is not playing
	return !mq.isPlaying || mq.pipeline == nil || !mq.pipeline.IsPlaying()
}

// SetVoiceConnection sets the voice connection for this queue
func (mq *MusicQueue) SetVoiceConnection(vc *discordgo.VoiceConnection) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.voiceConn = vc
}

// GetVoiceConnection returns the voice connection
func (mq *MusicQueue) GetVoiceConnection() *discordgo.VoiceConnection {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.voiceConn
}

// SetPipeline sets the audio pipeline for this queue
func (mq *MusicQueue) SetPipeline(pipeline audio.AudioPipeline) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.pipeline = pipeline
	
	// Log pipeline state change
	if mq.logger != nil {
		if pipeline != nil {
			mq.logger.Debug("Pipeline set for queue", map[string]interface{}{
				"pipeline_initialized": pipeline.IsInitialized(),
				"pipeline_playing":     pipeline.IsPlaying(),
			})
		} else {
			mq.logger.Debug("Pipeline cleared for queue", nil)
		}
	}
}

// GetPipeline returns the audio pipeline
func (mq *MusicQueue) GetPipeline() audio.AudioPipeline {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.pipeline
}

// SetSkipped sets the skipped flag
func (mq *MusicQueue) SetSkipped(skipped bool) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.wasSkipped = skipped
}

// WasSkipped returns whether the current song was skipped
func (mq *MusicQueue) WasSkipped() bool {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.wasSkipped
}

// StopAndCleanup safely stops the current pipeline and cleans up resources
func (mq *MusicQueue) StopAndCleanup() {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	// Log cleanup operation
	if mq.logger != nil {
		mq.logger.Info("Starting queue cleanup", map[string]interface{}{
			"has_pipeline":   mq.pipeline != nil,
			"has_voice_conn": mq.voiceConn != nil,
			"is_playing":     mq.isPlaying,
		})
	}

	// Stop and shutdown pipeline using new interface
	if mq.pipeline != nil {
		if err := mq.pipeline.Stop(); err != nil {
			if mq.logger != nil {
				mq.logger.Error("Error stopping pipeline during cleanup", err, nil)
			} else {
				log.Printf("Error stopping pipeline during cleanup: %v", err)
			}
		}
		
		// Shutdown pipeline to clean up all resources
		if err := mq.pipeline.Shutdown(); err != nil {
			if mq.logger != nil {
				mq.logger.Error("Error shutting down pipeline during cleanup", err, nil)
			} else {
				log.Printf("Error shutting down pipeline during cleanup: %v", err)
			}
		}
		
		mq.pipeline = nil
	}

	if mq.voiceConn != nil {
		mq.voiceConn.Disconnect()
		mq.voiceConn = nil
	}

	mq.isPlaying = false
	
	// Log successful cleanup
	if mq.logger != nil {
		mq.logger.Info("Queue cleanup completed", nil)
	}
}

// GetFreshStreamURLForCurrent gets a fresh stream URL for the current item
// This is useful for handling YouTube URL expiration issues
func (mq *MusicQueue) GetFreshStreamURLForCurrent() (string, error) {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	if mq.current == nil {
		return "", fmt.Errorf("no current item in queue")
	}

	// If it's a YouTube URL, get a fresh stream URL
	if mq.current.OriginalURL != "" {
		if mq.logger != nil {
			mq.logger.Debug("Getting fresh stream URL for YouTube video", map[string]interface{}{
				"title":        mq.current.Title,
				"original_url": mq.current.OriginalURL,
			})
		} else {
			log.Printf("Getting fresh stream URL for YouTube video: %s", mq.current.Title)
		}
		return GetFreshYouTubeStreamURL(mq.current.OriginalURL)
	}

	// For non-YouTube URLs, return the stored URL
	return mq.current.URL, nil
}

// CreateNewPipeline creates a new AudioPipeline using the new AudioPipelineController
// This method provides backward compatibility while using the new pipeline system
func (mq *MusicQueue) CreateNewPipeline() (audio.AudioPipeline, error) {
	if mq.db == nil {
		return nil, fmt.Errorf("database connection not available - use NewMusicQueueWithDB")
	}

	// Create new pipeline with all dependencies
	pipeline, err := audio.NewAudioPipelineWithDependencies(mq.db, mq.guildID)
	if err != nil {
		if mq.logger != nil {
			mq.logger.Error("Failed to create new audio pipeline", err, nil)
		}
		return nil, fmt.Errorf("failed to create audio pipeline: %w", err)
	}

	if mq.logger != nil {
		mq.logger.Info("Created new audio pipeline", map[string]interface{}{
			"pipeline_initialized": pipeline.IsInitialized(),
		})
	}

	return pipeline, nil
}

// StartPlayback starts playback using the new AudioPipelineController
// This method provides a high-level interface for starting playback with proper error handling
func (mq *MusicQueue) StartPlayback(url string, voiceConn *discordgo.VoiceConnection) error {
	// Create new pipeline if needed
	if mq.pipeline == nil {
		pipeline, err := mq.CreateNewPipeline()
		if err != nil {
			return fmt.Errorf("failed to create pipeline: %w", err)
		}
		mq.SetPipeline(pipeline)
	}

	// Set voice connection
	mq.SetVoiceConnection(voiceConn)

	// Start playback using new pipeline interface
	if err := mq.pipeline.PlayURL(url, voiceConn); err != nil {
		if mq.logger != nil {
			mq.logger.Error("Failed to start playback", err, map[string]interface{}{
				"url": url,
			})
		}
		return fmt.Errorf("failed to start playback: %w", err)
	}

	// Update playing state
	mq.SetPlaying(true)

	if mq.logger != nil {
		mq.logger.Info("Playback started successfully", map[string]interface{}{
			"url": url,
		})
	}

	return nil
}

// GetDB returns the database connection
func (mq *MusicQueue) GetDB() *gorm.DB {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	return mq.db
}

// SetDB sets the database connection for pipeline creation
func (mq *MusicQueue) SetDB(db *gorm.DB) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.db = db
	
	if mq.logger != nil {
		mq.logger.Debug("Database connection set for queue", map[string]interface{}{
			"has_db": db != nil,
		})
	}
}

// GetQueueStatusEmbed creates a centralized embed for queue status
func (mq *MusicQueue) GetQueueStatusEmbed() *discordgo.MessageEmbed {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	
	// Get current song info
	var currentSong string
	if mq.current != nil {
		currentSong = fmt.Sprintf("**%s** (Requested by: %s)", mq.current.Title, mq.current.RequestedBy)
	}
	
	// Get queue items as strings
	queueItems := make([]string, len(mq.items))
	for i, item := range mq.items {
		queueItems[i] = fmt.Sprintf("**%s** (Requested by: %s)", item.Title, item.RequestedBy)
	}
	
	// Log queue status request
	mq.logger.Debug("Generated queue status embed", map[string]interface{}{
		"current_song": currentSong,
		"queue_size":   len(mq.items),
		"is_playing":   mq.isPlaying,
	})
	
	return mq.embedBuilder.QueueStatus(currentSong, queueItems, len(mq.items))
}

// GetDetailedStatus returns detailed queue status information
func (mq *MusicQueue) GetDetailedStatus() map[string]interface{} {
	mq.mu.RLock()
	defer mq.mu.RUnlock()
	
	status := map[string]interface{}{
		"guild_id":         mq.guildID,
		"queue_size":       len(mq.items),
		"is_playing":       mq.isPlaying,
		"has_pipeline":     mq.pipeline != nil,
		"pipeline_playing": mq.pipeline != nil && mq.pipeline.IsPlaying(),
		"has_voice_conn":   mq.voiceConn != nil,
		"was_skipped":      mq.wasSkipped,
	}
	
	if mq.current != nil {
		status["current_song"] = map[string]interface{}{
			"title":        mq.current.Title,
			"url":          mq.current.URL,
			"requested_by": mq.current.RequestedBy,
			"started_at":   mq.current.StartedAt,
			"duration":     mq.current.Duration,
		}
	}
	
	// Log status request
	mq.logger.Debug("Generated detailed queue status", status)
	
	return status
}

// LogQueueOperation logs queue operations with centralized logging
func (mq *MusicQueue) LogQueueOperation(operation string, details map[string]interface{}) {
	if mq.logger == nil {
		return
	}
	
	// Merge operation details with queue context
	logFields := map[string]interface{}{
		"operation":   operation,
		"queue_size":  len(mq.items),
		"is_playing":  mq.isPlaying,
		"has_pipeline": mq.pipeline != nil,
	}
	
	// Add provided details
	for k, v := range details {
		logFields[k] = v
	}
	
	mq.logger.Info("Queue operation", logFields)
}
