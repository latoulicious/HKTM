package commands

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// ShowHelpCommand displays all available commands with their descriptions using embeds
func ShowHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("help")
	logger.Info("Help command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
	})
	// Create embed
	embed := &discordgo.MessageEmbed{
		Title: "Here are all the available commands for the bot:",
		// Description: "Here are all the available commands for the bot:",
		Color:     0x00ff00, // Green color
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Hokko Tarumae",
			IconURL: "https://cdn.discordapp.com/attachments/1378031194356060280/1402891387061403718/footer.gif?ex=68958feb&is=68943e6b&hm=21cdbed6dde8e956c55af9345d23755a617cf20f9f098fde6369a73164b67ca0&", // Replace with custom image URL
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "Music Commands",
				Value: strings.Join([]string{
					"• `!play <url>` / `!p <url>` - Play a YouTube video by URL",
					"• `!p <keywords>` - Search and play a YouTube video",
					"• `!nowplaying` / `!np` - Show the currently playing track",
					"• `!queue add <url>` - Add a YouTube video to the queue",
					"• `!queue list` - List the current queue",
					"• `!queue remove <position>` - Remove a track from the queue",
					"• `!clear` - Clear the entire queue",
					"• `!shuffle` - Shuffle the queue",
					"• `!pause` - Pause the current playback",
					"• `!resume` - Resume paused playback",
					"• `!skip` - Skip the currently playing track",
					"• `!stop` - Stop playback and disconnect from voice channel",
				}, "\n"),
				Inline: false,
			},
			{
				Name: "Information Commands",
				Value: strings.Join([]string{
					"• `!about` - Show bot info, uptime, and stats",
					"• `!servers` - List servers the bot is connected to (bot owner only)",
					"• `!help` / `!h` - Show this help message",
				}, "\n"),
				Inline: false,
			},
			{
				Name: "Moderation Commands",
				Value: strings.Join([]string{
					"• `!delete <number>` - Delete the specified number of recent messages",
				}, "\n"),
				Inline: false,
			},
			{
				Name:   "Fun Commands",
				Value:  "• `!gremlin` - Post a random gremlin image\n• `!uma char <name>` - Search for Uma Musume characters\n• `!uma support <name>` - Search for Uma Musume support cards\n• `!uma skills <name>` - Get skills for a support card",
				Inline: false,
			},
			{
				Name: "Utility Commands (Bot Owner Only)",
				Value: strings.Join([]string{
					"• `!utility cron` - Check cron job status",
					"• `!utility cron-refresh` - Manually trigger build ID refresh",
				}, "\n"),
				Inline: false,
			},
			{
				Name: "Admin Commands (Bot Owner Only)",
				Value: strings.Join([]string{
					"• `!leave <server_id>` - Force bot to leave a server by ID",
				}, "\n"),
				Inline: false,
			},
			{
				Name: "💡 Tips",
				Value: strings.Join([]string{
					"• Join a voice channel **before** using music commands",
					"• Only **YouTube links and searches** are currently supported",
				}, "\n"),
				Inline: false,
			},
		},
	}

	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		logger.Error("Failed to send help embed", err, map[string]interface{}{
			"channel_id": m.ChannelID,
			"guild_id":   m.GuildID,
		})
	}
}

// Unused commands
// • `!shuffle announce` - Shuffle and always announce the new top song
