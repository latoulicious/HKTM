package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/internal/version"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// VersionCommand displays version information
func VersionCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("version")
	logger.Info("Version command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
	})
	info := version.Get()

	embed := &discordgo.MessageEmbed{
		Title:       "🤖 HKTM Version Information",
		Description: info.String(),
		Color:       0x00ff00, // Green color
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Version",
				Value:  info.Version,
				Inline: true,
			},
			{
				Name:   "Git Commit",
				Value:  info.GitCommit,
				Inline: true,
			},
			{
				Name:   "Build Time",
				Value:  info.BuildTime,
				Inline: true,
			},
			{
				Name:   "Go Version",
				Value:  info.GoVersion,
				Inline: true,
			},
		},
	}

	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		logger.Error("Failed to send version embed", err, map[string]interface{}{
			"channel_id": m.ChannelID,
			"guild_id":   m.GuildID,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ Error displaying version information: "+err.Error())
	}
}
