package commands

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// LeaveCommand allows the bot owner to make the bot leave a specific server
func LeaveCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("leave")
	logger.Info("Leave command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
		"args_count": len(args),
	})
	// Check if the user is the bot owner
	ownerID := os.Getenv("BOT_OWNER_ID")
	if ownerID == "" {
		logger.Error("Bot owner ID not configured", nil, map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ Bot owner ID not configured.")
		return
	}

	if m.Author.ID != ownerID {
		logger.Warn("Leave command denied - not bot owner", map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
			"owner_id": ownerID,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ You don't have permission to use this command.")
		return
	}

	// Require a server ID argument
	if len(args) < 1 {
		logger.Warn("Leave command called without server ID", map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ Please provide a server ID. Usage: `!leave <server_id>`\n💡 Use `!servers` to see available server IDs.")
		return
	}

	// Get the server ID from arguments
	serverID := args[0]

	logger.Info("Attempting to leave server", map[string]interface{}{
		"user_id":          m.Author.ID,
		"guild_id":         m.GuildID,
		"target_server_id": serverID,
	})

	// Validate server ID format (Discord IDs are 17-19 digits)
	if len(serverID) < 17 || len(serverID) > 19 {
		logger.Warn("Invalid server ID format provided", map[string]interface{}{
			"user_id":          m.Author.ID,
			"guild_id":         m.GuildID,
			"target_server_id": serverID,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ Invalid server ID format.")
		return
	}

	// Check if the bot is actually in the specified server
	guild, err := s.Guild(serverID)
	if err != nil {
		logger.Error("Server not found or bot not in server", err, map[string]interface{}{
			"user_id":          m.Author.ID,
			"guild_id":         m.GuildID,
			"target_server_id": serverID,
		})
		s.ChannelMessageSend(m.ChannelID, "❌ Server not found or bot is not in that server.")
		return
	}

	// Leave the server directly
	err = s.GuildLeave(serverID)
	if err != nil {
		logger.Error("Failed to leave server", err, map[string]interface{}{
			"user_id":          m.Author.ID,
			"guild_id":         m.GuildID,
			"target_server_id": serverID,
			"server_name":      guild.Name,
		})
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("❌ Failed to leave server: %v", err))
		return
	}

	logger.Info("Successfully left server", map[string]interface{}{
		"user_id":          m.Author.ID,
		"guild_id":         m.GuildID,
		"target_server_id": serverID,
		"server_name":      guild.Name,
	})

	// Send confirmation message
	leaveMsg := fmt.Sprintf("✅ Successfully left **%s** (ID: %s)", guild.Name, serverID)
	s.ChannelMessageSend(m.ChannelID, leaveMsg)
}
