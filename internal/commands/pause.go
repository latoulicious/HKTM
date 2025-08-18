package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/embed"
	"github.com/latoulicious/HKTM/pkg/logging"
)

func PauseCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID

	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("pause")
	logger.Info("Pause command executed", map[string]interface{}{
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

	// Check if there's a pipeline that can be paused
	pipeline := queue.GetPipeline()
	if pipeline == nil || !pipeline.IsPlaying() {
		logger.Warn("No active audio pipeline found", map[string]interface{}{
			"guild_id":   guildID,
			"user_id":    m.Author.ID,
			"has_pipeline": pipeline != nil,
			"is_playing": pipeline != nil && pipeline.IsPlaying(),
		})
		
		errorEmbed := embedBuilder.Error("❌ Error", "No audio is currently playing.")
		s.ChannelMessageSendEmbed(m.ChannelID, errorEmbed)
		return
	}

	// For now, pause is not implemented in the current audio pipeline
	// This would require implementing pause/resume functionality in the AudioPipeline
	logger.Warn("Pause functionality not yet implemented", map[string]interface{}{
		"guild_id": guildID,
		"user_id":  m.Author.ID,
	})
	
	warningEmbed := embedBuilder.Warning("⏸️ Pause Not Available", "Pause functionality is not yet implemented. Use `!stop` to stop playback.")
	s.ChannelMessageSendEmbed(m.ChannelID, warningEmbed)
}
