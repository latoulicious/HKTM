package commands

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/common"
	"github.com/latoulicious/HKTM/pkg/embed"
	"github.com/latoulicious/HKTM/pkg/logging"
)

var (
	// Global pipeline manager to track active streams
	activePipelines = make(map[string]*common.AudioPipeline)
	pipelineMutex   sync.RWMutex

	// Centralized systems for play command
	playCommandEmbedBuilder embed.AudioEmbedBuilder
	playCommandLogger       logging.Logger
)

// InitializePlayCommand initializes the centralized systems for play command
func InitializePlayCommand() {
	playCommandEmbedBuilder = embed.GetGlobalAudioEmbedBuilder()
	loggerFactory := logging.GetGlobalLoggerFactory()
	playCommandLogger = loggerFactory.CreateCommandLogger("play")
}

// PlayCommand handles the play command with queue integration
func PlayCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Initialize centralized systems if not already done
	if playCommandEmbedBuilder == nil || playCommandLogger == nil {
		InitializePlayCommand()
	}

	// Log command execution with centralized logging
	playCommandLogger.Info("Play command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
		"args_count": len(args),
	})

	if len(args) < 1 {
		// Use centralized embed system for error messages
		embed := playCommandEmbedBuilder.Error("❌ Usage Error", "Please provide a YouTube URL or search query.")
		s.ChannelMessageSendEmbed(m.ChannelID, embed)

		playCommandLogger.Warn("Play command called without arguments", map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		return
	}

	guildID := m.GuildID

	// Update activity for idle monitoring
	updateActivity(guildID)

	input := args[0]
	var url, title string
	var duration time.Duration
	var videoURL string // Store the video URL for search results

	// Check if input is a URL or search query
	if common.IsURL(input) {
		// Input is a URL, use existing logic
		playCommandLogger.Debug("Processing direct URL", map[string]interface{}{
			"url":      input,
			"user_id":  m.Author.ID,
			"guild_id": guildID,
		})

		streamURL, streamTitle, streamDuration, err := common.GetYouTubeAudioStreamWithMetadata(input)
		if err != nil {
			playCommandLogger.Error("Error fetching stream URL", err, map[string]interface{}{
				"url":      input,
				"user_id":  m.Author.ID,
				"guild_id": guildID,
			})

			// Use centralized embed system for error messages
			embed := playCommandEmbedBuilder.Error("❌ Error", "Failed to get audio stream. Please check the URL.")
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
			return
		}
		url = streamURL
		title = streamTitle
		duration = streamDuration
		videoURL = input // For direct URLs, use the input as video URL
	} else {
		// Input is a search query, search YouTube and get the first result
		searchQuery := strings.Join(args, " ") // Join all args as search query
		playCommandLogger.Debug("Processing search query", map[string]interface{}{
			"search_query": searchQuery,
			"user_id":      m.Author.ID,
			"guild_id":     guildID,
		})

		// Search for the video and get its URL
		foundVideoURL, _, _, searchErr := common.SearchYouTubeAndGetURL(searchQuery)
		if searchErr != nil {
			playCommandLogger.Error("Error searching YouTube", searchErr, map[string]interface{}{
				"search_query": searchQuery,
				"user_id":      m.Author.ID,
				"guild_id":     guildID,
			})

			// Use centralized embed system for error messages
			embed := playCommandEmbedBuilder.Error("❌ Search Error", "Failed to find any videos for your search query.")
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
			return
		}

		// Get metadata only (no stream URL extraction to prevent expiration)
		metadataTitle, metadataDuration, metadataErr := common.GetYouTubeMetadata(foundVideoURL)
		if metadataErr != nil {
			playCommandLogger.Error("Error fetching metadata from search result", metadataErr, map[string]interface{}{
				"found_video_url": foundVideoURL,
				"search_query":    searchQuery,
				"user_id":         m.Author.ID,
				"guild_id":        guildID,
			})

			// Use centralized embed system for error messages
			embed := playCommandEmbedBuilder.Error("❌ Error", "Failed to get video metadata from search result.")
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
			return
		}

		// Use the original YouTube URL, not a pre-extracted stream URL
		url = foundVideoURL
		title = metadataTitle
		duration = metadataDuration
		videoURL = foundVideoURL // Store the found video URL
	}

	// Get or create queue for this guild
	queue := getOrCreateQueue(guildID)

	// Check if it's a YouTube URL and extract video ID
	var videoID string
	var originalURL string

	// Check if the video URL is a YouTube URL
	if common.IsYouTubeURL(videoURL) {
		videoID = common.ExtractYouTubeVideoID(videoURL)
		originalURL = videoURL
		// Pass original YouTube URL - audio pipeline will extract stream URL just-in-time
		queue.AddWithYouTubeData("", originalURL, videoID, title, m.Author.Username, duration)

		playCommandLogger.Info("Added YouTube video to queue", map[string]interface{}{
			"title":        title,
			"video_id":     videoID,
			"original_url": originalURL,
			"duration":     duration.String(),
			"user_id":      m.Author.ID,
			"username":     m.Author.Username,
			"guild_id":     guildID,
			"queue_size":   queue.Size(),
		})
	} else {
		// Use the original method for non-YouTube URLs
		queue.Add(url, title, m.Author.Username)

		playCommandLogger.Info("Added non-YouTube URL to queue", map[string]interface{}{
			"title":      title,
			"url":        url,
			"user_id":    m.Author.ID,
			"username":   m.Author.Username,
			"guild_id":   guildID,
			"queue_size": queue.Size(),
		})
	}

	// Send confirmation with centralized embed system
	queueSize := queue.Size()
	description := fmt.Sprintf("✅ Added **%s** to queue (Position: %d)", title, queueSize)
	embed := playCommandEmbedBuilder.Success("🎵 Song Added", description)
	s.ChannelMessageSendEmbed(m.ChannelID, embed)

	// Check if we should start playing - only if the queue can start playing
	if queue.CanStartPlaying() {
		playCommandLogger.Info("Starting playback", map[string]interface{}{
			"guild_id":     guildID,
			"queue_size":   queueSize,
			"is_playing":   queue.IsPlaying(),
			"has_pipeline": queue.HasActivePipeline(),
			"can_start":    queue.CanStartPlaying(),
		})
		startNextInQueue(s, m, queue)
	} else {
		playCommandLogger.Debug("Song added to queue but not starting playback", map[string]interface{}{
			"guild_id":     guildID,
			"queue_size":   queueSize,
			"is_playing":   queue.IsPlaying(),
			"has_pipeline": queue.HasActivePipeline(),
			"can_start":    queue.CanStartPlaying(),
		})
	}
}

// StatusCommand shows the current playback status
func StatusCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Initialize centralized systems if not already done
	if playCommandEmbedBuilder == nil || playCommandLogger == nil {
		InitializePlayCommand()
	}

	guildID := m.GuildID

	// Log status command execution
	playCommandLogger.Info("Status command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   guildID,
		"channel_id": m.ChannelID,
	})

	pipelineMutex.RLock()
	pipeline, exists := activePipelines[guildID]
	pipelineMutex.RUnlock()

	if !exists || !pipeline.IsPlaying() {
		// Use centralized embed system
		embed := playCommandEmbedBuilder.Info("🔇 No Audio", "No audio is currently playing.")
		s.ChannelMessageSendEmbed(m.ChannelID, embed)

		playCommandLogger.Debug("Status check - no audio playing", map[string]interface{}{
			"guild_id":         guildID,
			"pipeline_exists":  exists,
			"pipeline_playing": exists && pipeline.IsPlaying(),
		})
		return
	}

	// Use centralized embed system
	embed := playCommandEmbedBuilder.Success("🎵 Audio Playing", "Audio is currently playing.")
	s.ChannelMessageSendEmbed(m.ChannelID, embed)

	playCommandLogger.Debug("Status check - audio playing", map[string]interface{}{
		"guild_id": guildID,
	})
}
