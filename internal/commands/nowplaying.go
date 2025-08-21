package commands

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/common"
	"github.com/latoulicious/HKTM/pkg/embed"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// NowPlayingCommand handles the nowplaying command
func NowPlayingCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID

	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("nowplaying")
	logger.Info("Now playing command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"guild_id":   guildID,
		"channel_id": m.ChannelID,
	})

	// Initialize centralized embed builder
	embedBuilder := embed.GetGlobalAudioEmbedBuilder()

	// Update activity for idle monitoring
	updateActivity(guildID)

	// Get the queue for this guild
	queue := getQueue(guildID)
	if queue == nil {
		logger.Error("No queue found for guild", nil, map[string]interface{}{
			"guild_id": guildID,
			"user_id":  m.Author.ID,
		})
		
		infoEmbed := embedBuilder.Info("🎵 Now Playing", "Nothing is currently playing. Use `/play` to start playing music.")
		s.ChannelMessageSendEmbed(m.ChannelID, infoEmbed)
		return
	}

	// Get current playing item
	currentItem := queue.Current()
	if currentItem == nil || !queue.IsPlaying() {
		logger.Info("No current item playing", map[string]interface{}{
			"guild_id":     guildID,
			"user_id":      m.Author.ID,
			"has_current":  currentItem != nil,
			"is_playing":   queue.IsPlaying(),
		})
		
		infoEmbed := embedBuilder.Info("🎵 Now Playing", "Nothing is currently playing. Use `/play` to start playing music.")
		s.ChannelMessageSendEmbed(m.ChannelID, infoEmbed)
		return
	}

	logger.Info("Displaying now playing info", map[string]interface{}{
		"guild_id":     guildID,
		"user_id":      m.Author.ID,
		"song_title":   currentItem.Title,
		"requested_by": currentItem.RequestedBy,
		"duration":     currentItem.Duration.String(),
	})

	// Get pipeline for duration/position info if available
	pipeline := queue.GetPipeline()
	voiceConn := queue.GetVoiceConnection()

	// Send now playing embed using centralized system
	sendNowPlayingEmbed(s, m.ChannelID, currentItem, pipeline, voiceConn, embedBuilder, logger)
}

// sendNowPlayingEmbed sends a detailed now playing embed using centralized systems
func sendNowPlayingEmbed(s *discordgo.Session, channelID string, item *common.QueueItem, pipeline interface{}, voiceConn *discordgo.VoiceConnection, embedBuilder embed.AudioEmbedBuilder, logger logging.Logger) {
	// Determine connection status
	var isPlaying bool

	// Handle both old and new pipeline types
	if pipeline != nil {
		// Try new AudioPipeline interface first
		if newPipeline, ok := pipeline.(interface{ IsPlaying() bool }); ok {
			isPlaying = newPipeline.IsPlaying()
		} else if oldPipeline, ok := pipeline.(*common.AudioPipeline); ok {
			// Fallback to old AudioPipeline type
			isPlaying = oldPipeline.IsPlaying()
		}
	}

	logger.Debug("Pipeline status checked", map[string]interface{}{
		"has_pipeline":    pipeline != nil,
		"is_playing":      isPlaying,
		"voice_ready":     voiceConn != nil && voiceConn.Ready,
	})

	// Use centralized embed system for now playing
	var nowPlayingEmbed *discordgo.MessageEmbed
	
	// Get the original URL for the embed
	originalURL := item.OriginalURL
	if originalURL == "" {
		// Fallback to constructing YouTube URL if we have video ID
		if item.VideoID != "" {
			originalURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.VideoID)
		}
	}

	// Create now playing embed using centralized system
	nowPlayingEmbed = embedBuilder.NowPlaying(item.Title, originalURL, item.Duration)

	// Add additional fields for enhanced information
	statusEmoji := "🔴"
	statusText := "Stopped"
	
	if isPlaying {
		if voiceConn != nil && voiceConn.Ready {
			statusEmoji = "🟢"
			statusText = "Playing"
		} else {
			statusEmoji = "🟡"
			statusText = "Connecting..."
		}
	}

	// Add custom fields to the centralized embed
	nowPlayingEmbed.Fields = append(nowPlayingEmbed.Fields, 
		&discordgo.MessageEmbedField{
			Name:   "Requested by",
			Value:  item.RequestedBy,
			Inline: true,
		},
		&discordgo.MessageEmbedField{
			Name:   "Status",
			Value:  fmt.Sprintf("%s %s", statusEmoji, statusText),
			Inline: true,
		},
		&discordgo.MessageEmbedField{
			Name:   "Added to queue",
			Value:  item.AddedAt.Format("Jan 2, 2006 3:04 PM"),
			Inline: false,
		},
	)

	// Add YouTube thumbnail if video ID is available
	if item.VideoID != "" {
		thumbnailURL := common.GetYouTubeThumbnailURL(item.VideoID)
		nowPlayingEmbed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: thumbnailURL,
		}
	}

	// Send the embed
	_, err := s.ChannelMessageSendEmbed(channelID, nowPlayingEmbed)
	if err != nil {
		logger.Error("Failed to send now playing embed", err, map[string]interface{}{
			"channel_id": channelID,
			"song_title": item.Title,
		})
	}
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	hours := minutes / 60
	minutes = minutes % 60

	return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
}
