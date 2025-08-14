package common

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
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
	guildID    string
	items      []*QueueItem
	current    *QueueItem
	isPlaying  bool
	wasSkipped bool // Flag to track if current song was skipped
	mu         sync.RWMutex
	voiceConn  *discordgo.VoiceConnection
	pipeline   *AudioPipeline
}

// NewMusicQueue creates a new music queue for a guild
func NewMusicQueue(guildID string) *MusicQueue {
	return &MusicQueue{
		guildID: guildID,
		items:   make([]*QueueItem, 0),
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
	log.Printf("Added '%s' to queue for guild %s", title, mq.guildID)
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
	log.Printf("Added '%s' (Duration: %v) to queue for guild %s", title, duration, mq.guildID)
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
	mq.items = make([]*QueueItem, 0)
	mq.current = nil
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
	log.Printf("Removed '%s' from queue for guild %s", removed.Title, mq.guildID)
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
func (mq *MusicQueue) SetPipeline(pipeline *AudioPipeline) {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.pipeline = pipeline
}

// GetPipeline returns the audio pipeline
func (mq *MusicQueue) GetPipeline() *AudioPipeline {
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

	if mq.pipeline != nil {
		mq.pipeline.Stop()
		mq.pipeline = nil
	}

	if mq.voiceConn != nil {
		mq.voiceConn.Disconnect()
		mq.voiceConn = nil
	}

	mq.isPlaying = false
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
		log.Printf("Getting fresh stream URL for YouTube video: %s", mq.current.Title)
		return GetFreshYouTubeStreamURL(mq.current.OriginalURL)
	}

	// For non-YouTube URLs, return the stored URL
	return mq.current.URL, nil
}
