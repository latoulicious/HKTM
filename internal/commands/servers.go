package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// ServersCommand displays information about which servers the bot is joined to
func ServersCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("servers")
	logger.Info("Servers command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
	})
	guilds := s.State.Guilds

	logger.Info("Displaying server list", map[string]interface{}{
		"user_id":      m.Author.ID,
		"guild_id":     m.GuildID,
		"server_count": len(guilds),
	})

	if len(guilds) == 0 {
		logger.Warn("No servers found in bot state", map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		s.ChannelMessageSend(m.ChannelID, "I'm not joined to any servers.")
		return
	}

	// Create a response message with server information including IDs
	var response string
	if len(guilds) == 1 {
		response = fmt.Sprintf("I'm joined to **1 server**:\n• **%s** (ID: `%s`)", guilds[0].Name, guilds[0].ID)
	} else {
		response = fmt.Sprintf("I'm joined to **%d servers**:\n", len(guilds))
		for i, guild := range guilds {
			response += fmt.Sprintf("• **%s** (ID: `%s`)", guild.Name, guild.ID)
			if i < len(guilds)-1 {
				response += "\n"
			}
		}
	}

	response += "\n\n💡 **Tip**: Use `!leave <server_id>` to leave a server."
	_, err := s.ChannelMessageSend(m.ChannelID, response)
	if err != nil {
		logger.Error("Failed to send servers list", err, map[string]interface{}{
			"channel_id": m.ChannelID,
			"guild_id":   m.GuildID,
		})
	}
}
