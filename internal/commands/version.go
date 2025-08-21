package commands

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/internal/version"
	"github.com/latoulicious/HKTM/pkg/logging"
)

func code(s string) string {
	if s == "" {
		return "`n/a`"
	}
	return "`" + s + "`"
}

func VersionCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	logger := logging.GetGlobalLoggerFactory().CreateCommandLogger("version")
	logger.Info("Version command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
	})

	info := version.Get()

	// Parse build time -> human readable fallback
	buildTime := info.BuildTime
	if t, err := time.Parse(time.RFC3339, info.BuildTime); err == nil {
		buildTime = t.UTC().Format("02 Jan 2006 15:04 UTC")
	}

	commitURL := fmt.Sprintf("https://github.com/latoulicious/HKTM/commit/%s", info.GitCommit)

	embed := &discordgo.MessageEmbed{
		Title: "HKTM Version",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			// First row
			{Name: "Version", Value: code(info.Version), Inline: true},
			{Name: "Commit", Value: fmt.Sprintf("[%s](%s)", info.ShortCommit, commitURL), Inline: true},

			// Second row
			{Name: "Go", Value: code(info.GoVersion), Inline: true},
			{Name: "Build Time", Value: code(buildTime), Inline: true},
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

	_, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embed: embed,
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{},
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
