package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/embed"
	"github.com/latoulicious/HKTM/pkg/logging"
)

func SkipCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID

	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("skip")
	logger.Info("Skip command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"guild_id":   guildID,
		"channel_id": m.ChannelID,
	})

	// Initialize centralized embed builder
	embedBuilder := embed.GetGlobalAudioEmbedBuilder()

	// Update activity for idle monitoring
	updateActivity(guildID)

	// Get queue for this guild
	queue := getQueue(guildID)
	if queue == nil {
		logger.Error("No queue found for guild", nil, map[string]interface{}{
			"guild_id": guildID,
			"user_id":  m.Author.ID,
		})
		
		errorEmbed := embedBuilder.Error("❌ Error", "No queue found for this guild.")
		s.ChannelMessageSendEmbed(m.ChannelID, errorEmbed)
		return
	}

	if !queue.IsPlaying() {
		logger.Info("Nothing currently playing to skip", map[string]interface{}{
			"guild_id":   guildID,
			"user_id":    m.Author.ID,
			"queue_size": queue.Size(),
		})
		
		infoEmbed := embedBuilder.Info("⏭️ Nothing to Skip", "Nothing is currently playing.")
		s.ChannelMessageSendEmbed(m.ChannelID, infoEmbed)
		return
	}

	// Get current song info before stopping
	currentSong := queue.Current()
	var songTitle, requestedBy string
	if currentSong != nil {
		songTitle = currentSong.Title
		requestedBy = currentSong.RequestedBy
	}

	logger.Info("Skipping current song", map[string]interface{}{
		"guild_id":     guildID,
		"user_id":      m.Author.ID,
		"skipped_by":   m.Author.Username,
		"song_title":   songTitle,
		"requested_by": requestedBy,
		"queue_size":   queue.Size(),
	})

	// Stop current pipeline and mark as skipped
	if pipeline := queue.GetPipeline(); pipeline != nil {
		// Set a flag to indicate this was a skip operation
		queue.SetSkipped(true)
		
		if err := pipeline.Stop(); err != nil {
			logger.Error("Error stopping pipeline during skip", err, map[string]interface{}{
				"guild_id":   guildID,
				"user_id":    m.Author.ID,
				"song_title": songTitle,
			})
		}
	}

	// Send skip embed using centralized embed system
	var skipEmbed *discordgo.MessageEmbed
	if currentSong != nil {
		skipEmbed = embedBuilder.SongSkipped(songTitle, requestedBy, m.Author.Username)
	} else {
		skipEmbed = embedBuilder.Warning("⏭️ Song Skipped", "Current song has been skipped.")
	}
	
	_, err := s.ChannelMessageSendEmbed(m.ChannelID, skipEmbed)
	if err != nil {
		logger.Error("Failed to send skip embed", err, map[string]interface{}{
			"channel_id": m.ChannelID,
			"guild_id":   guildID,
		})
	}

	// Start next song in queue
	logger.Debug("Starting next song in queue", map[string]interface{}{
		"guild_id":         guildID,
		"remaining_queue":  queue.Size(),
	})
	startNextInQueue(s, m, queue)
}
