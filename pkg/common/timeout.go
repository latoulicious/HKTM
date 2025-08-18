package common

import (
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/embed"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// TimeoutManager manages idle timeouts for music queues
type TimeoutManager struct {
	lastActivityTime map[string]time.Time
	mu               sync.RWMutex
	logger           logging.Logger
	embedBuilder     embed.AudioEmbedBuilder
	session          *discordgo.Session
	presenceManager  PresenceManager // Interface for presence management
	queueGetter      QueueGetter     // Interface to get queues
}

// QueueGetter interface for getting queues (to avoid circular dependencies)
type QueueGetter interface {
	GetQueue(guildID string) *MusicQueue
}

// PresenceManager interface for managing bot presence
type PresenceManager interface {
	ClearMusicPresence()
}

// NewTimeoutManager creates a new TimeoutManager
func NewTimeoutManager(session *discordgo.Session, presenceManager PresenceManager, queueGetter QueueGetter) *TimeoutManager {
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateLogger("timeout")
	
	return &TimeoutManager{
		lastActivityTime: make(map[string]time.Time),
		logger:           logger,
		embedBuilder:     embed.GetGlobalAudioEmbedBuilder(),
		session:          session,
		presenceManager:  presenceManager,
		queueGetter:      queueGetter,
	}
}

// UpdateActivity updates the last activity time for a guild
func (tm *TimeoutManager) UpdateActivity(guildID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.lastActivityTime[guildID] = time.Now()
	
	tm.logger.Debug("Updated activity for guild", map[string]interface{}{
		"guild_id": guildID,
		"time":     time.Now(),
	})
}

// RemoveGuild removes a guild from timeout tracking
func (tm *TimeoutManager) RemoveGuild(guildID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	delete(tm.lastActivityTime, guildID)
	
	tm.logger.Debug("Removed guild from timeout tracking", map[string]interface{}{
		"guild_id": guildID,
	})
}

// StartMonitoring starts the idle timeout monitoring
func (tm *TimeoutManager) StartMonitoring() {
	tm.logger.Info("Starting idle timeout monitoring", map[string]interface{}{
		"check_interval": "30s",
		"timeout_duration": "5m",
	})
	
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
		defer ticker.Stop()

		for range ticker.C {
			tm.checkForIdleTimeouts()
		}
	}()
}

// checkForIdleTimeouts checks for guilds that have exceeded the idle timeout
func (tm *TimeoutManager) checkForIdleTimeouts() {
	now := time.Now()
	tm.mu.RLock()
	
	var timeoutGuilds []string
	for guildID, lastActivity := range tm.lastActivityTime {
		// Check if more than 5 minutes have passed since last activity
		if now.Sub(lastActivity) > 5*time.Minute {
			timeoutGuilds = append(timeoutGuilds, guildID)
		}
	}
	
	tm.mu.RUnlock()
	
	// Process timeouts outside of the read lock
	for _, guildID := range timeoutGuilds {
		tm.handleIdleTimeout(guildID)
	}
}

// handleIdleTimeout handles the idle timeout for a specific guild
func (tm *TimeoutManager) handleIdleTimeout(guildID string) {
	tm.logger.Info("Handling idle timeout for guild", map[string]interface{}{
		"guild_id": guildID,
	})
	
	// Get queue for this guild
	var queue *MusicQueue
	if tm.queueGetter != nil {
		queue = tm.queueGetter.GetQueue(guildID)
	}
	
	if queue == nil {
		tm.logger.Debug("No queue found for guild during timeout", map[string]interface{}{
			"guild_id": guildID,
		})
		tm.RemoveGuild(guildID)
		return
	}
	
	// Only timeout if the queue is actually playing
	if !queue.IsCurrentlyPlaying() {
		tm.logger.Debug("Queue not playing, skipping timeout", map[string]interface{}{
			"guild_id": guildID,
			"is_playing": queue.IsPlaying(),
			"has_pipeline": queue.HasActivePipeline(),
		})
		tm.RemoveGuild(guildID)
		return
	}
	
	tm.logger.Info("Executing idle timeout for guild", map[string]interface{}{
		"guild_id": guildID,
		"queue_size": queue.Size(),
		"current_song": func() string {
			if current := queue.Current(); current != nil {
				return current.Title
			}
			return "none"
		}(),
	})
	
	// Stop the queue and clean up resources
	queue.StopAndCleanup()
	
	// Clear presence
	if tm.presenceManager != nil {
		tm.presenceManager.ClearMusicPresence()
		tm.logger.Debug("Cleared music presence due to timeout", map[string]interface{}{
			"guild_id": guildID,
		})
	}
	
	// Send timeout notification to a text channel
	tm.sendTimeoutNotification(guildID)
	
	// Remove from idle tracking
	tm.RemoveGuild(guildID)
}

// sendTimeoutNotification sends a timeout notification to the guild
func (tm *TimeoutManager) sendTimeoutNotification(guildID string) {
	// Find a text channel to send the embed
	_, err := tm.session.Guild(guildID)
	if err != nil {
		tm.logger.Error("Failed to get guild for timeout notification", err, map[string]interface{}{
			"guild_id": guildID,
		})
		return
	}
	
	channels, err := tm.session.GuildChannels(guildID)
	if err != nil {
		tm.logger.Error("Failed to get guild channels for timeout notification", err, map[string]interface{}{
			"guild_id": guildID,
		})
		return
	}
	
	// Find the first available text channel
	var targetChannelID string
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildText {
			targetChannelID = channel.ID
			break
		}
	}
	
	if targetChannelID == "" {
		tm.logger.Warn("No text channel found for timeout notification", map[string]interface{}{
			"guild_id": guildID,
		})
		return
	}
	
	// Create and send timeout embed using centralized embed system
	timeoutEmbed := tm.embedBuilder.IdleTimeout()
	
	_, err = tm.session.ChannelMessageSendEmbed(targetChannelID, timeoutEmbed)
	if err != nil {
		tm.logger.Error("Failed to send timeout notification", err, map[string]interface{}{
			"guild_id": guildID,
			"channel_id": targetChannelID,
		})
	} else {
		tm.logger.Info("Sent idle timeout notification", map[string]interface{}{
			"guild_id": guildID,
			"channel_id": targetChannelID,
		})
	}
}

// GetActiveGuilds returns the list of guilds currently being tracked
func (tm *TimeoutManager) GetActiveGuilds() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	guilds := make([]string, 0, len(tm.lastActivityTime))
	for guildID := range tm.lastActivityTime {
		guilds = append(guilds, guildID)
	}
	
	return guilds
}

// GetLastActivity returns the last activity time for a guild
func (tm *TimeoutManager) GetLastActivity(guildID string) (time.Time, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	lastActivity, exists := tm.lastActivityTime[guildID]
	return lastActivity, exists
}