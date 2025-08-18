package commands

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/common"
)

// NowPlayingCommand handles the nowplaying command
func NowPlayingCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID

	// Update activity for idle monitoring
	updateActivity(guildID)

	// Get the queue for this guild
	queue := getQueue(guildID)
	if queue == nil {
		sendNothingPlayingEmbed(s, m.ChannelID)
		return
	}

	// Get current playing item
	currentItem := queue.Current()
	if currentItem == nil || !queue.IsPlaying() {
		sendNothingPlayingEmbed(s, m.ChannelID)
		return
	}

	// Get pipeline for duration/position info if available
	pipeline := queue.GetPipeline()
	voiceConn := queue.GetVoiceConnection()

	// Send now playing embed
	sendNowPlayingEmbed(s, m.ChannelID, currentItem, pipeline, voiceConn)
}

// sendNothingPlayingEmbed sends an embed when nothing is playing
func sendNothingPlayingEmbed(s *discordgo.Session, channelID string) {
	embed := &discordgo.MessageEmbed{
		Title:       "🎵 Now Playing",
		Description: "Nothing is currently playing",
		Color:       0x808080, // Gray color
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /play to start playing music",
		},
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}

// sendNowPlayingEmbed sends a detailed now playing embed
func sendNowPlayingEmbed(s *discordgo.Session, channelID string, item *common.QueueItem, pipeline interface{}, voiceConn *discordgo.VoiceConnection) {
	// Build description with track info
	description := fmt.Sprintf("**%s**", item.Title)

	// Show total duration instead of playback progress
	durationStr := formatDuration(item.Duration)

	// Determine connection status
	var statusEmoji string
	var statusText string
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

	if isPlaying {
		if voiceConn != nil && voiceConn.Ready {
			statusEmoji = "🟢"
			statusText = "Playing"
		} else {
			statusEmoji = "🟡"
			statusText = "Connecting..."
		}
	} else {
		statusEmoji = "🔴"
		statusText = "Stopped"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎵 Now Playing",
		Description: description,
		Color:       0x00ff00, // Green color
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Requested by",
				Value:  item.RequestedBy,
				Inline: true,
			},
			{
				Name:   "Duration",
				Value:  durationStr,
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  fmt.Sprintf("%s %s", statusEmoji, statusText),
				Inline: true,
			},
			{
				Name:   "Added to queue",
				Value:  item.AddedAt.Format("Jan 2, 2006 3:04 PM"),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
	}

	// Add YouTube thumbnail if video ID is available
	if item.VideoID != "" {
		thumbnailURL := common.GetYouTubeThumbnailURL(item.VideoID)
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: thumbnailURL,
		}
	}

	// Add YouTube link if original URL is available
	if item.OriginalURL != "" && common.IsYouTubeURL(item.OriginalURL) {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🔗 YouTube Link",
			Value:  fmt.Sprintf("[Open in YouTube](%s)", item.OriginalURL),
			Inline: true,
		})
	}

	s.ChannelMessageSendEmbed(channelID, embed)
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
