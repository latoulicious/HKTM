package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/common"
	"github.com/latoulicious/HKTM/pkg/embed"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// StopCommand stops the current audio playback and clears queue
func StopCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID
	
	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("stop")
	logger.Info("Stop command executed", map[string]interface{}{
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

	// Check if anything is playing
	if !queue.IsPlaying() {
		logger.Info("No audio currently playing", map[string]interface{}{
			"guild_id": guildID,
			"user_id":  m.Author.ID,
		})
		
		infoEmbed := embedBuilder.Info("⏹️ Nothing Playing", "No audio is currently playing.")
		s.ChannelMessageSendEmbed(m.ChannelID, infoEmbed)
		return
	}

	// Get current song info before stopping
	var currentSong string
	if current := queue.Current(); current != nil {
		currentSong = current.Title
	}

	// Stop current pipeline
	if pipeline := queue.GetPipeline(); pipeline != nil {
		logger.Debug("Stopping audio pipeline", map[string]interface{}{
			"guild_id":     guildID,
			"current_song": currentSong,
		})
		
		if err := pipeline.Stop(); err != nil {
			logger.Error("Error stopping pipeline", err, map[string]interface{}{
				"guild_id": guildID,
				"user_id":  m.Author.ID,
			})
		}
	}

	// Clear queue and stop playing
	queueSizeBefore := queue.Size()
	queue.Clear()
	queue.SetPlaying(false)

	logger.Info("Queue cleared and playback stopped", map[string]interface{}{
		"guild_id":           guildID,
		"user_id":            m.Author.ID,
		"queue_size_before":  queueSizeBefore,
		"current_song":       currentSong,
		"stopped_by":         m.Author.Username,
	})

	// Clear presence
	if presenceManager != nil {
		presenceManager.ClearMusicPresence()
		logger.Debug("Music presence cleared", map[string]interface{}{
			"guild_id": guildID,
		})
	}

	// Send stop embed using centralized embed system
	stopEmbed := embedBuilder.PlaybackStopped(m.Author.Username)
	_, err := s.ChannelMessageSendEmbed(m.ChannelID, stopEmbed)
	if err != nil {
		logger.Error("Failed to send stop embed", err, map[string]interface{}{
			"channel_id": m.ChannelID,
			"guild_id":   guildID,
		})
	}

	// Find and disconnect from voice channel
	logger.Debug("Disconnecting from voice channel", map[string]interface{}{
		"guild_id": guildID,
	})
	common.DisconnectFromVoiceChannel(s, guildID)
}
