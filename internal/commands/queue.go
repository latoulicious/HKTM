package commands

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/internal/presence"
	"github.com/latoulicious/HKTM/pkg/common"
	"github.com/latoulicious/HKTM/pkg/embed"
	"github.com/latoulicious/HKTM/pkg/logging"
	"gorm.io/gorm"
)

var (
	// Global queue manager to track queues per guild
	queues     = make(map[string]*common.MusicQueue)
	queueMutex sync.RWMutex

	// Global database connection for pipeline creation
	queueDB *gorm.DB

	// Global presence manager
	presenceManager *presence.PresenceManager

	// Enhanced timeout manager
	timeoutManager *common.TimeoutManager

	// Centralized embed builder and logger
	embedBuilder embed.AudioEmbedBuilder
	logger       logging.Logger
)

// SetPresenceManager sets the global presence manager
func SetPresenceManager(pm *presence.PresenceManager) {
	presenceManager = pm
}

// InitializeEnhancedTimeout initializes the enhanced timeout system
func InitializeEnhancedTimeout(session *discordgo.Session) {
	// Initialize centralized systems
	embedBuilder = embed.GetGlobalAudioEmbedBuilder()
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger = loggerFactory.CreateLogger("queue-commands")
	
	// Create timeout manager with presence manager interface
	var pm common.PresenceManager
	if presenceManager != nil {
		pm = &presenceManagerAdapter{presenceManager}
	}
	
	// Create queue getter adapter
	queueGetter := &queueGetterAdapter{}
	
	timeoutManager = common.NewTimeoutManager(session, pm, queueGetter)
	timeoutManager.StartMonitoring()
	
	logger.Info("Enhanced timeout system initialized", map[string]interface{}{
		"has_presence_manager": presenceManager != nil,
	})
}

// presenceManagerAdapter adapts the internal presence manager to the common interface
type presenceManagerAdapter struct {
	pm *presence.PresenceManager
}

func (pma *presenceManagerAdapter) ClearMusicPresence() {
	if pma.pm != nil {
		pma.pm.ClearMusicPresence()
	}
}

// queueGetterAdapter adapts the internal queue management to the common interface
type queueGetterAdapter struct{}

func (qga *queueGetterAdapter) GetQueue(guildID string) *common.MusicQueue {
	return getQueue(guildID)
}

// updateActivity updates the last activity time for a guild using enhanced timeout manager
func updateActivity(guildID string) {
	if timeoutManager != nil {
		timeoutManager.UpdateActivity(guildID)
	}
	
	// Log activity update
	if logger != nil {
		logger.Debug("Updated activity for guild", map[string]interface{}{
			"guild_id": guildID,
		})
	}
}

// GetIdleMonitor returns the enhanced idle monitor initialization function
func GetIdleMonitor() func(*discordgo.Session) {
	return InitializeEnhancedTimeout
}

// QueueCommand handles the queue command
func QueueCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		// Show current queue
		showQueue(s, m)
		return
	}

	// Handle subcommands
	subcommand := strings.ToLower(args[0])
	switch subcommand {
	case "add":
		if len(args) < 2 {
			sendEmbedMessage(s, m.ChannelID, "❌ Usage Error", "Usage: `!queue add <youtube_url>`", 0xff0000)
			return
		}
		addToQueue(s, m, args[1:])
	case "remove":
		if len(args) < 2 {
			sendEmbedMessage(s, m.ChannelID, "❌ Usage Error", "Usage: `!queue remove <index>`", 0xff0000)
			return
		}
		removeFromQueue(s, m, args[1:])
	case "clear":
		clearQueue(s, m)
	case "list":
		showQueue(s, m)
	default:
		sendEmbedMessage(s, m.ChannelID, "❌ Usage Error", "Usage: `!queue [add|remove|clear|list] [args...]`", 0xff0000)
	}
}

// sendEmbedMessage is a helper function to send embed messages using centralized embeds
func sendEmbedMessage(s *discordgo.Session, channelID, title, description string, color int) {
	var embed *discordgo.MessageEmbed
	
	// Use centralized embed builder based on color
	switch color {
	case 0x00ff00: // Green - Success
		embed = embedBuilder.Success(title, description)
	case 0xff0000: // Red - Error
		embed = embedBuilder.Error(title, description)
	case 0xffa500: // Orange - Warning
		embed = embedBuilder.Warning(title, description)
	default: // Default - Info
		embed = embedBuilder.Info(title, description)
	}
	
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil && logger != nil {
		logger.Error("Failed to send embed message", err, map[string]interface{}{
			"channel_id": channelID,
			"title":      title,
		})
	}
}

// sendSongFinishedEmbed sends an embed when a song finishes playing using centralized embeds
func sendSongFinishedEmbed(s *discordgo.Session, channelID, songTitle, requestedBy string) {
	embed := embedBuilder.SongFinished(songTitle, requestedBy)
	
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil && logger != nil {
		logger.Error("Failed to send song finished embed", err, map[string]interface{}{
			"channel_id":   channelID,
			"song_title":   songTitle,
			"requested_by": requestedBy,
		})
	}
}

// sendQueueEndedEmbed sends an embed when the queue ends using centralized embeds
func sendQueueEndedEmbed(s *discordgo.Session, channelID string) {
	embed := embedBuilder.QueueEnded()
	
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil && logger != nil {
		logger.Error("Failed to send queue ended embed", err, map[string]interface{}{
			"channel_id": channelID,
		})
	}
}

// sendSongSkippedEmbed sends an embed when a song is skipped using centralized embeds
func sendSongSkippedEmbed(s *discordgo.Session, channelID, songTitle, requestedBy, skippedBy string) {
	embed := embedBuilder.SongSkipped(songTitle, requestedBy, skippedBy)
	
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil && logger != nil {
		logger.Error("Failed to send song skipped embed", err, map[string]interface{}{
			"channel_id":   channelID,
			"song_title":   songTitle,
			"requested_by": requestedBy,
			"skipped_by":   skippedBy,
		})
	}
}

// sendBotStoppedEmbed sends an embed when the bot stops/disconnects using centralized embeds
func sendBotStoppedEmbed(s *discordgo.Session, channelID, stoppedBy string) {
	embed := embedBuilder.PlaybackStopped(stoppedBy)
	
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil && logger != nil {
		logger.Error("Failed to send playback stopped embed", err, map[string]interface{}{
			"channel_id": channelID,
			"stopped_by": stoppedBy,
		})
	}
}

// addToQueue adds a song to the queue
func addToQueue(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID
	url := args[0]

	// Update activity
	updateActivity(guildID)

	// Get or create queue for this guild
	queue := getOrCreateQueue(guildID)

	// Validate and get stream URL with metadata
	streamURL, title, duration, err := common.GetYouTubeAudioStreamWithMetadata(url)
	if err != nil {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Failed to get audio stream. Please check the URL.", 0xff0000)
		return
	}

	// Check if it's a YouTube URL and extract video ID
	var videoID string
	var originalURL string
	if common.IsYouTubeURL(url) {
		videoID = common.ExtractYouTubeVideoID(url)
		originalURL = url
		// Use the new method for YouTube videos
		queue.AddWithYouTubeData(streamURL, originalURL, videoID, title, m.Author.Username, duration)
	} else {
		// Use the original method for non-YouTube URLs
		queue.Add(streamURL, title, m.Author.Username)
	}

	// Send confirmation with embed
	queueSize := queue.Size()
	description := fmt.Sprintf("✅ Added **%s** to queue (Position: %d)", title, queueSize)
	sendEmbedMessage(s, m.ChannelID, "🎵 Song Added", description, 0x00ff00)
	
	// Log queue operation with centralized logging
	queue.LogQueueOperation("song_added", map[string]interface{}{
		"title":        title,
		"url":          url,
		"requested_by": m.Author.Username,
		"user_id":      m.Author.ID,
		"channel_id":   m.ChannelID,
		"position":     queueSize,
	})

	// Check if we should start playing - only if the queue can start playing
	if queue.CanStartPlaying() {
		log.Printf("Starting playback for guild %s - queue size: %d, isPlaying: %v, hasPipeline: %v",
			guildID, queueSize, queue.IsPlaying(), queue.HasActivePipeline())
		startNextInQueue(s, m, queue)
	} else {
		log.Printf("Song added to queue for guild %s but not starting playback - queue size: %d, isPlaying: %v, hasPipeline: %v",
			guildID, queueSize, queue.IsPlaying(), queue.HasActivePipeline())
	}
}

// removeFromQueue removes a song from the queue
func removeFromQueue(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID

	// Update activity
	updateActivity(guildID)

	queue := getQueue(guildID)

	if queue == nil {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "No queue found for this server.", 0xff0000)
		return
	}

	// Parse index
	var index int
	_, err := fmt.Sscanf(args[0], "%d", &index)
	if err != nil {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Invalid index. Use `!queue list` to see queue positions.", 0xff0000)
		return
	}

	// Adjust for 1-based indexing
	index--

	err = queue.Remove(index)
	if err != nil {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", err.Error(), 0xff0000)
		return
	}

	sendEmbedMessage(s, m.ChannelID, "✅ Success", "Removed song from queue.", 0x00ff00)
	
	// Log queue operation with centralized logging
	queue.LogQueueOperation("song_removed", map[string]interface{}{
		"index":      index + 1, // Convert back to 1-based for logging
		"user_id":    m.Author.ID,
		"channel_id": m.ChannelID,
	})
}

// clearQueue clears the entire queue
func clearQueue(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID

	// Update activity
	updateActivity(guildID)

	queue := getQueue(guildID)

	if queue == nil {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "No queue found for this server.", 0xff0000)
		return
	}

	queueSizeBefore := queue.Size()
	queue.Clear()
	sendEmbedMessage(s, m.ChannelID, "✅ Success", "Queue cleared.", 0x00ff00)
	
	// Log queue operation with centralized logging
	queue.LogQueueOperation("queue_cleared", map[string]interface{}{
		"items_cleared": queueSizeBefore,
		"user_id":       m.Author.ID,
		"channel_id":    m.ChannelID,
	})
}

// showQueue shows the current queue using centralized embeds and logging
func showQueue(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID

	// Update activity
	updateActivity(guildID)

	queue := getQueue(guildID)

	if queue == nil || (queue.Size() == 0 && queue.Current() == nil) {
		sendEmbedMessage(s, m.ChannelID, "📭 Queue Empty", "No songs in the queue.", 0x808080)
		
		// Log queue status request
		if logger != nil {
			logger.Info("Queue status requested - empty queue", map[string]interface{}{
				"guild_id":   guildID,
				"user_id":    m.Author.ID,
				"channel_id": m.ChannelID,
			})
		}
		return
	}

	// Use centralized queue status embed
	embed := queue.GetQueueStatusEmbed()
	
	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil && logger != nil {
		logger.Error("Failed to send queue status embed", err, map[string]interface{}{
			"guild_id":   guildID,
			"channel_id": m.ChannelID,
		})
	} else if logger != nil {
		// Log successful queue status display
		logger.Info("Queue status displayed", map[string]interface{}{
			"guild_id":     guildID,
			"user_id":      m.Author.ID,
			"channel_id":   m.ChannelID,
			"queue_size":   queue.Size(),
			"has_current":  queue.Current() != nil,
		})
	}
}

// startNextInQueue starts playing the next song in the queue
func startNextInQueue(s *discordgo.Session, m *discordgo.MessageCreate, queue *common.MusicQueue) {
	// Check if there's already an active pipeline and clean it up
	if queue.HasActivePipeline() {
		log.Printf("Cleaning up existing pipeline before starting new one")
		queue.StopAndCleanup()
	}

	item := queue.Next()
	if item == nil {
		queue.SetPlaying(false)
		// Clear presence when no more songs
		if presenceManager != nil {
			presenceManager.ClearMusicPresence()
		}
		// Send queue ended embed
		sendQueueEndedEmbed(s, m.ChannelID)
		return
	}

	queue.SetPlaying(true)

	// Find user's voice channel and connect
	vc, err := common.FindAndJoinUserVoiceChannel(s, m.Author.ID, m.GuildID)
	if err != nil {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", err.Error(), 0xff0000)
		queue.SetPlaying(false)
		return
	}

	queue.SetVoiceConnection(vc)

	// Update bot presence to show current song
	if presenceManager != nil {
		log.Printf("Updating presence to show: %s", item.Title)
		presenceManager.UpdateMusicPresence(item.Title)
	} else {
		log.Printf("Warning: presenceManager is nil, cannot update presence")
	}

	// Send now playing message with embed
	description := item.Title
	sendEmbedMessage(s, m.ChannelID, "🎶 Now Playing", description, 0x00ff00)

	// Use the new pipeline system with enhanced error handling
	// The queue will handle pipeline creation and URL refresh automatically
	var playbackURL string
	if item.OriginalURL != "" {
		// For YouTube URLs, get a fresh URL to avoid expiration
		freshURL, urlErr := queue.GetFreshStreamURLForCurrent()
		if urlErr != nil {
			log.Printf("Failed to get fresh URL, using original: %v", urlErr)
			playbackURL = item.URL
		} else {
			playbackURL = freshURL
		}
	} else {
		playbackURL = item.URL
	}

	// Start playback using the new pipeline system
	err = queue.StartPlayback(playbackURL, vc)
	if err != nil {
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Failed to start audio playback.", 0xff0000)
		queue.StopAndCleanup()
		if presenceManager != nil {
			presenceManager.ClearMusicPresence()
		}
		return
	}

	// Monitor the pipeline and handle completion
	go func() {
		// Get the pipeline for monitoring
		pipeline := queue.GetPipeline()
		if pipeline == nil {
			log.Printf("Warning: No pipeline available for monitoring")
			return
		}

		// Wait for pipeline to finish
		for pipeline.IsPlaying() {
			time.Sleep(1 * time.Second)
		}

		// Only send song finished embed if the song wasn't skipped
		if !queue.WasSkipped() {
			sendSongFinishedEmbed(s, m.ChannelID, item.Title, item.RequestedBy)
		}

		// Clean up the pipeline
		queue.SetPipeline(nil)
		queue.SetSkipped(false) // Reset the skipped flag

		// Play next song in queue
		startNextInQueue(s, m, queue)
	}()
}

// getOrCreateQueue gets or creates a queue for a guild
func getOrCreateQueue(guildID string) *common.MusicQueue {
	queueMutex.Lock()
	defer queueMutex.Unlock()

	if queue, exists := queues[guildID]; exists {
		// Update existing queue with database connection if available and not already set
		if queueDB != nil && queue.GetDB() == nil {
			queue.SetDB(queueDB)
		}
		return queue
	}

	// Create queue with database connection if available
	var queue *common.MusicQueue
	if queueDB != nil {
		queue = common.NewMusicQueueWithDB(guildID, queueDB)
	} else {
		// Fallback to basic queue if no database connection available
		queue = common.NewMusicQueue(guildID)
	}
	
	queues[guildID] = queue
	return queue
}

// getOrCreateQueueWithDB gets or creates a queue for a guild with database connection
func getOrCreateQueueWithDB(guildID string, db *gorm.DB) *common.MusicQueue {
	queueMutex.Lock()
	defer queueMutex.Unlock()

	if queue, exists := queues[guildID]; exists {
		// Update existing queue with database connection if not already set
		if queue.GetDB() == nil {
			queue.SetDB(db)
		}
		return queue
	}

	// Create new queue with database connection
	queue := common.NewMusicQueueWithDB(guildID, db)
	queues[guildID] = queue
	return queue
}

// getQueue gets a queue for a guild
func getQueue(guildID string) *common.MusicQueue {
	queueMutex.RLock()
	defer queueMutex.RUnlock()
	return queues[guildID]
}

// InitializeQueueCommands initializes the queue commands with database connection
func InitializeQueueCommands(db *gorm.DB) {
	queueDB = db
	log.Println("Queue commands initialized with database connection")
}

// InitializeCommandsWithDB initializes the commands package with database connection for audio pipeline support
func InitializeCommandsWithDB(db *gorm.DB) {
	queueDB = db
	
	// Initialize centralized systems
	embedBuilder = embed.GetGlobalAudioEmbedBuilder()
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger = loggerFactory.CreateLogger("commands")
	
	logger.Info("Commands package initialized with database connection", map[string]interface{}{
		"database_connected": true,
		"audio_pipeline_support": true,
	})
}

// ShutdownAllAudioPipelines gracefully shuts down all active audio pipelines
func ShutdownAllAudioPipelines() {
	queueMutex.Lock()
	defer queueMutex.Unlock()
	
	loggerFactory := logging.GetGlobalLoggerFactory()
	shutdownLogger := loggerFactory.CreateLogger("shutdown")
	
	shutdownCount := 0
	for guildID, queue := range queues {
		if queue != nil && queue.HasActivePipeline() {
			shutdownLogger.Info("Shutting down audio pipeline", map[string]interface{}{
				"guild_id": guildID,
			})
			
			// Stop current playbook
			queue.StopAndCleanup()
			shutdownCount++
		}
	}
	
	shutdownLogger.Info("Audio pipeline shutdown complete", map[string]interface{}{
		"pipelines_shutdown": shutdownCount,
		"total_queues": len(queues),
	})
}
