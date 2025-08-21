package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/embed"
	"github.com/latoulicious/HKTM/pkg/logging"
)

func ResumeCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	guildID := m.GuildID

	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("resume")
	logger.Info("Resume command executed", map[string]interface{}{
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

	// Check if there's a pipeline that can be resumed
	pipeline := queue.GetPipeline()
	if pipeline == nil {
		logger.Warn("No pipeline found for resume", map[string]interface{}{
			"guild_id": guildID,
			"user_id":  m.Author.ID,
		})
		
		errorEmbed := embedBuilder.Error("❌ Error", "No audio pipeline found.")
		s.ChannelMessageSendEmbed(m.ChannelID, errorEmbed)
		return
	}

	// Check if already playing
	if pipeline.IsPlaying() {
		logger.Info("Audio already playing, no need to resume", map[string]interface{}{
			"guild_id": guildID,
			"user_id":  m.Author.ID,
		})
		
		infoEmbed := embedBuilder.Info("▶️ Already Playing", "Audio is already playing.")
		s.ChannelMessageSendEmbed(m.ChannelID, infoEmbed)
		return
	}

	// For now, resume is not implemented in the current audio pipeline
	// This would require implementing pause/resume functionality in the AudioPipeline
	logger.Warn("Resume functionality not yet implemented", map[string]interface{}{
		"guild_id": guildID,
		"user_id":  m.Author.ID,
	})
	
	warningEmbed := embedBuilder.Warning("▶️ Resume Not Available", "Resume functionality is not yet implemented. Use `!play` to start new playback.")
	s.ChannelMessageSendEmbed(m.ChannelID, warningEmbed)
}
