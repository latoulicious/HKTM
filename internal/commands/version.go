package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/internal/version"
	"github.com/latoulicious/HKTM/pkg/logging"
)

func VersionCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	logger := logging.GetGlobalLoggerFactory().CreateCommandLogger("version")
	logger.Info("Version command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
	})

	info := version.Get()

	// Keep description tight; details go in fields.
	desc := fmt.Sprintf("`%s`", info.String())

	embed := &discordgo.MessageEmbed{
		Title:       "HKTM Version",
		Description: desc,
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Version", Value: code(info.Version), Inline: true},
			{Name: "Commit", Value: code(info.ShortCommit), Inline: true},
			{Name: "Build Time", Value: code(info.BuildTime), Inline: true},
			{Name: "Go", Value: code(info.GoVersion), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: func() string {
				if info.Dirty {
					return "⚠️ dirty workspace at build time"
				}
				return ""
			}(),
		},
	}

	// Prevent accidental mentions in responses.
	_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embed: embed,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{}, // no @everyone/@here
		},
	})
	if err != nil {
		logger.Error("Failed to send version embed", err, map[string]interface{}{
			"channel_id": m.ChannelID,
			"guild_id":   m.GuildID,
		})
		_, _ = s.ChannelMessageSend(m.ChannelID, "❌ Error displaying version information: "+err.Error())
	}
}

func code(s string) string {
	if s == "" {
		return "`n/a`"
	}
	return "`" + s + "`"
}
