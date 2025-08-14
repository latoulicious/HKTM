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
)

var (
	// Global queue manager to track queues per guild
	queues     = make(map[string]*common.MusicQueue)
	queueMutex sync.RWMutex

	// Global presence manager
	presenceManager *presence.PresenceManager

	// Idle tracking
	lastActivityTime = make(map[string]time.Time)
	idleMutex        sync.RWMutex
)

// SetPresenceManager sets the global presence manager
func SetPresenceManager(pm *presence.PresenceManager) {
	presenceManager = pm
}

// updateActivity updates the last activity time for a guild
func updateActivity(guildID string) {
	idleMutex.Lock()
	lastActivityTime[guildID] = time.Now()
	idleMutex.Unlock()
}

// sendIdleDisconnectEmbed sends an embed when the bot disconnects due to idle timeout
func sendIdleDisconnectEmbed(s *discordgo.Session, channelID string) {
	embed := &discordgo.MessageEmbed{
		Title:     "⏰ Idle Timeout",
		Color:     0xffa500, // Orange
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
		Description: "Bot has been idle for 5 minutes. Disconnected from voice channel to preserve resources.\nUse `!play` to start playing again!",
	}
	s.ChannelMessageSendEmbed(channelID, embed)
}

// startIdleMonitor starts monitoring for idle timeouts
func startIdleMonitor(s *discordgo.Session) {
	go func() {
		ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()
			idleMutex.RLock()

			for guildID, lastActivity := range lastActivityTime {
				// Check if more than 5 minutes have passed since last activity
				if now.Sub(lastActivity) > 5*time.Minute {
					// Get queue for this guild
					queue := getQueue(guildID)
					if queue != nil && queue.IsPlaying() {
						// Stop the queue and clean up resources
						queue.StopAndCleanup()

						// Clear presence
						if presenceManager != nil {
							presenceManager.ClearMusicPresence()
						}

						// Find a text channel to send the embed
						// We'll use the first available text channel
						guild, err := s.Guild(guildID)
						if err == nil && guild != nil {
							channels, err := s.GuildChannels(guildID)
							if err == nil {
								for _, channel := range channels {
									if channel.Type == discordgo.ChannelTypeGuildText {
										sendIdleDisconnectEmbed(s, channel.ID)
										break
									}
								}
							}
						}

						// Remove from idle tracking
						idleMutex.RUnlock()
						idleMutex.Lock()
						delete(lastActivityTime, guildID)
						idleMutex.Unlock()
						idleMutex.RLock()
					}
				}
			}

			idleMutex.RUnlock()
		}
	}()
}

// GetIdleMonitor returns the idle monitor function
func GetIdleMonitor() func(*discordgo.Session) {
	return startIdleMonitor
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

// sendEmbedMessage is a helper function to send embed messages
func sendEmbedMessage(s *discordgo.Session, channelID, title, description string, color int) {
	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
	}
	s.ChannelMessageSendEmbed(channelID, embed)
}

// sendSongFinishedEmbed sends an embed when a song finishes playing
func sendSongFinishedEmbed(s *discordgo.Session, channelID, songTitle, requestedBy string) {
	embed := &discordgo.MessageEmbed{
		Title:     "🎵 Song Finished",
		Color:     0x00ff00, // Green
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Finished Playing",
				Value:  fmt.Sprintf("**%s**\nRequested by: %s", songTitle, requestedBy),
				Inline: false,
			},
		},
	}
	s.ChannelMessageSendEmbed(channelID, embed)
}

// sendQueueEndedEmbed sends an embed when the queue ends
func sendQueueEndedEmbed(s *discordgo.Session, channelID string) {
	embed := &discordgo.MessageEmbed{
		Title:     "📭 Queue Ended",
		Color:     0x808080, // Gray
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
		Description: "All songs in the queue have been played. Add more songs with `!play` or `!queue add`!",
	}
	s.ChannelMessageSendEmbed(channelID, embed)
}

// sendSongSkippedEmbed sends an embed when a song is skipped
func sendSongSkippedEmbed(s *discordgo.Session, channelID, songTitle, requestedBy, skippedBy string) {
	embed := &discordgo.MessageEmbed{
		Title:     "⏭️ Song Skipped",
		Color:     0xffa500, // Orange
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Skipped Song",
				Value:  fmt.Sprintf("**%s**\nRequested by: %s", songTitle, requestedBy),
				Inline: false,
			},
			{
				Name:   "Skipped By",
				Value:  skippedBy,
				Inline: false,
			},
		},
	}
	s.ChannelMessageSendEmbed(channelID, embed)
}

// sendBotStoppedEmbed sends an embed when the bot stops/disconnects
func sendBotStoppedEmbed(s *discordgo.Session, channelID, stoppedBy string) {
	embed := &discordgo.MessageEmbed{
		Title:     "⏹️ Playback Stopped",
		Color:     0xff0000, // Red
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Stopped By",
				Value:  stoppedBy,
				Inline: false,
			},
		},
		Description: "Music playback has been stopped. Use `!play` to start playing again!",
	}
	s.ChannelMessageSendEmbed(channelID, embed)
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

	queue.Clear()
	sendEmbedMessage(s, m.ChannelID, "✅ Success", "Queue cleared.", 0x00ff00)
}

// showQueue shows the current queue
func showQueue(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID

	// Update activity
	updateActivity(guildID)

	queue := getQueue(guildID)

	if queue == nil || (queue.Size() == 0 && queue.Current() == nil) {
		sendEmbedMessage(s, m.ChannelID, "📭 Queue Empty", "No songs in the queue.", 0x808080)
		return
	}

	// Create embed for queue display
	embed := &discordgo.MessageEmbed{
		Title:     "🎵 Music Queue",
		Color:     0x0099ff,
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
	}

	var fields []*discordgo.MessageEmbedField

	// Show currently playing
	if current := queue.Current(); current != nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "🎶 Now Playing",
			Value:  fmt.Sprintf(current.Title),
			Inline: false,
		})
	}

	// Show queue items
	items := queue.List()
	if len(items) > 0 {
		var queueText strings.Builder
		for i, item := range items {
			queueText.WriteString(fmt.Sprintf("%d. **%s** (Requested by: %s)\n", i+1, item.Title, item.RequestedBy))
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "📋 Up Next",
			Value:  queueText.String(),
			Inline: false,
		})
	} else {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "📋 Up Next",
			Value:  "No songs in queue.",
			Inline: false,
		})
	}

	embed.Fields = fields
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
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

	// Create and start the audio pipeline
	pipeline := common.NewAudioPipeline(vc)
	queue.SetPipeline(pipeline)

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

	// Start streaming with URL refresh capability for YouTube URLs
	if item.OriginalURL != "" {
		// Use the new method that can refresh URLs on restart
		err = pipeline.PlayStreamWithOriginalURL(item.URL, item.OriginalURL)
	} else {
		// Use the standard method for non-YouTube URLs
		err = pipeline.PlayStream(item.URL)
	}
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
		return queue
	}

	queue := common.NewMusicQueue(guildID)
	queues[guildID] = queue
	return queue
}

// getQueue gets a queue for a guild
func getQueue(guildID string) *common.MusicQueue {
	queueMutex.RLock()
	defer queueMutex.RUnlock()
	return queues[guildID]
}
