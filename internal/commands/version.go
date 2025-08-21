package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/internal/version"
)

// VersionCommand displays version information
func VersionCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
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
		s.ChannelMessageSend(m.ChannelID, "❌ Error displaying version information: "+err.Error())
	}
}
