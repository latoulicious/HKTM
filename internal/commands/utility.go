package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/uma/handler"
)

// UtilityCommand handles utility-related commands
func UtilityCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "❌ Please specify a subcommand.\n\n**Usage:** `!utility <subcommand>`\n**Available subcommands:**\n• `cron` - Check cron job status (Bot Owner Only)\n• `cron-refresh` - Manually trigger build ID refresh (Bot Owner Only)\n\n**Examples:**\n• `!utility cron`\n• `!utility cron-refresh`")
		return
	}

	subcommand := strings.ToLower(args[0])

	switch subcommand {
	case "cron":
		CronStatusCommand(s, m, args[1:])
	case "cron-refresh":
		CronRefreshCommand(s, m, args[1:])
	default:
		s.ChannelMessageSend(m.ChannelID, "❌ Unknown subcommand.\n\n**Available subcommands:**\n• `cron` - Check cron job status (Bot Owner Only)\n• `cron-refresh` - Manually trigger build ID refresh (Bot Owner Only)\n\n**Examples:**\n• `!utility cron`\n• `!utility cron-refresh`")
	}
}

// CronStatusCommand shows the status of cron jobs
func CronStatusCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if user is bot owner
	if m.Author.ID != s.State.User.ID {
		s.ChannelMessageSend(m.ChannelID, "❌ This command is restricted to the bot owner only.")
		return
	}

	client := handler.GetGametoraClient()
	if client == nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Gametora client not available.")
		return
	}

	// Get build ID manager
	buildIDManager := client.GetBuildIDManager()
	if buildIDManager == nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Build ID manager not available.")
		return
	}

	// Get current build ID
	buildID, err := client.GetBuildID()
	if err != nil {
		buildID = "Error fetching build ID"
	}

	// Get next run time
	nextRun := buildIDManager.GetNextRun()
	var nextRunStr string
	if nextRun.IsZero() {
		nextRunStr = "Not scheduled"
	} else {
		nextRunStr = nextRun.Format("2006-01-02 15:04:05")
	}

	// Create status embed
	embed := &discordgo.MessageEmbed{
		Title:       "⏰ Cron Job Status",
		Description: "Current status of automated build ID refresh jobs",
		Color:       0x7289DA, // Discord blue
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae | Utility Commands",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🔄 Build ID Refresh",
				Value:  "Active",
				Inline: true,
			},
			{
				Name:   "📅 Schedule",
				Value:  buildIDManager.GetSchedule(),
				Inline: true,
			},
			{
				Name:   "⏭️ Next Run",
				Value:  nextRunStr,
				Inline: true,
			},
			{
				Name:   "🏃‍♂️ Currently Running",
				Value:  fmt.Sprintf("%t", buildIDManager.IsRunning()),
				Inline: true,
			},
			{
				Name:   "🆔 Current Build ID",
				Value:  buildID,
				Inline: true,
			},
		},
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// CronRefreshCommand manually triggers a build ID refresh
func CronRefreshCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if user is bot owner
	if m.Author.ID != s.State.User.ID {
		s.ChannelMessageSend(m.ChannelID, "❌ This command is restricted to the bot owner only.")
		return
	}

	client := handler.GetGametoraClient()
	if client == nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Gametora client not available.")
		return
	}

	// Send initial message
	msg, _ := s.ChannelMessageSend(m.ChannelID, "🔄 Manually triggering build ID refresh...")

	// Trigger refresh
	err := client.RefreshBuildID()

	if err != nil {
		// Update message with error
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Build ID Refresh Failed",
			Description: "Failed to refresh build ID",
			Color:       0xff0000, // Red
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Hokko Tarumae | Utility Commands",
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "🔧 Error",
					Value:  err.Error(),
					Inline: false,
				},
			},
		}
		s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, embed)
	} else {
		// Get new build ID
		newBuildID, _ := client.GetBuildID()

		// Update message with success
		embed := &discordgo.MessageEmbed{
			Title:       "✅ Build ID Refresh Complete",
			Description: "Successfully refreshed build ID",
			Color:       0x00ff00, // Green
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Hokko Tarumae | Utility Commands",
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "🆔 New Build ID",
					Value:  newBuildID,
					Inline: true,
				},
				{
					Name:   "⏰ Refreshed At",
					Value:  time.Now().Format("2006-01-02 15:04:05"),
					Inline: true,
				},
			},
		}
		s.ChannelMessageEditEmbed(m.ChannelID, msg.ID, embed)
	}
}
