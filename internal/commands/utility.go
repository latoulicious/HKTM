package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/logging"
	"github.com/latoulicious/HKTM/pkg/uma/handler"
)

// UtilityCommand handles utility-related commands
func UtilityCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("utility")
	logger.Info("Utility command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
		"args_count": len(args),
	})
	if len(args) == 0 {
		logger.Warn("Utility command called without subcommand", map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ Please specify a subcommand.\n\n**Usage:** `!utility <subcommand>`\n**Available subcommands:**\n• `cron` - Check cron job status (Bot Owner Only)\n• `cron-refresh` - Manually trigger build ID refresh (Bot Owner Only)\n\n**Examples:**\n• `!utility cron`\n• `!utility cron-refresh`")
		return
	}

	subcommand := strings.ToLower(args[0])

	logger.Info("Utility subcommand called", map[string]interface{}{
		"user_id":    m.Author.ID,
		"guild_id":   m.GuildID,
		"subcommand": subcommand,
	})

	switch subcommand {
	case "cron":
		CronStatusCommand(s, m, args[1:], logger)
	case "cron-refresh":
		CronRefreshCommand(s, m, args[1:], logger)
	default:
		logger.Warn("Unknown utility subcommand", map[string]interface{}{
			"user_id":    m.Author.ID,
			"guild_id":   m.GuildID,
			"subcommand": subcommand,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ Unknown subcommand.\n\n**Available subcommands:**\n• `cron` - Check cron job status (Bot Owner Only)\n• `cron-refresh` - Manually trigger build ID refresh (Bot Owner Only)\n\n**Examples:**\n• `!utility cron`\n• `!utility cron-refresh`")
	}
}

// CronStatusCommand shows the status of cron jobs
func CronStatusCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string, logger logging.Logger) {
	logger.Info("Cron status command executed", map[string]interface{}{
		"user_id":  m.Author.ID,
		"guild_id": m.GuildID,
	})

	// Check if user is bot owner
	if m.Author.ID != s.State.User.ID {
		logger.Warn("Cron status command denied - not bot owner", map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ This command is restricted to the bot owner only.")
		return
	}

	client := handler.GetGametoraClient()
	if client == nil {
		logger.Error("Gametora client not available", nil, map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
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

	_, sendErr := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if sendErr != nil {
		logger.Error("Failed to send cron status embed", sendErr, map[string]interface{}{
			"channel_id": m.ChannelID,
			"guild_id":   m.GuildID,
		})
	}
}

// CronRefreshCommand manually triggers a build ID refresh
func CronRefreshCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string, logger logging.Logger) {
	logger.Info("Cron refresh command executed", map[string]interface{}{
		"user_id":  m.Author.ID,
		"guild_id": m.GuildID,
	})

	// Check if user is bot owner
	if m.Author.ID != s.State.User.ID {
		logger.Warn("Cron refresh command denied - not bot owner", map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ This command is restricted to the bot owner only.")
		return
	}

	client := handler.GetGametoraClient()
	if client == nil {
		logger.Error("Gametora client not available for refresh", nil, map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ Gametora client not available.")
		return
	}

	// Send initial message
	msg, _ := s.ChannelMessageSend(m.ChannelID, "🔄 Manually triggering build ID refresh...")

	logger.Info("Starting manual build ID refresh", map[string]interface{}{
		"user_id":  m.Author.ID,
		"guild_id": m.GuildID,
	})

	// Trigger refresh
	err := client.RefreshBuildID()

	if err != nil {
		logger.Error("Build ID refresh failed", err, map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})

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

		logger.Info("Build ID refresh completed successfully", map[string]interface{}{
			"user_id":      m.Author.ID,
			"guild_id":     m.GuildID,
			"new_build_id": newBuildID,
		})

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
