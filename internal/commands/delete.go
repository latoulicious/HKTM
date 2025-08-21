package commands

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// DeleteCommand handles the !delete command to delete recent messages
func DeleteCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Initialize centralized logging for this command
	loggerFactory := logging.GetGlobalLoggerFactory()
	logger := loggerFactory.CreateCommandLogger("delete")
	logger.Info("Delete command executed", map[string]interface{}{
		"user_id":    m.Author.ID,
		"username":   m.Author.Username,
		"guild_id":   m.GuildID,
		"channel_id": m.ChannelID,
		"args_count": len(args),
	})
	// Check if user has manage messages permission
	hasPermission := hasManageMessagesPermission(s, m.GuildID, m.Author.ID)
	if !hasPermission {
		logger.Warn("Delete command denied - insufficient permissions", map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		sendEmbedMessage(s, m.ChannelID, "❌ Permission Denied", "You need 'Manage Messages' permission to use this command.", 0xff0000)
		return
	}

	// Check if number of messages is provided
	if len(args) == 0 {
		logger.Warn("Delete command called without arguments", map[string]interface{}{
			"user_id":  m.Author.ID,
			"guild_id": m.GuildID,
		})
		sendEmbedMessage(s, m.ChannelID, "❌ Invalid Usage", "Usage: `!delete <number>` - Delete the specified number of recent messages.", 0xff0000)
		return
	}

	// Parse the number of messages to delete
	numStr := args[0]
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		logger.Warn("Delete command called with invalid number", map[string]interface{}{
			"user_id":     m.Author.ID,
			"guild_id":    m.GuildID,
			"input":       numStr,
			"parse_error": err,
		})
		sendEmbedMessage(s, m.ChannelID, "❌ Invalid Number", "Please provide a valid positive number of messages to delete.", 0xff0000)
		return
	}

	// Limit the number of messages to delete (Discord API limit is 100)
	if num > 100 {
		num = 100
	}

	logger.Info("Starting message deletion", map[string]interface{}{
		"user_id":         m.Author.ID,
		"guild_id":        m.GuildID,
		"channel_id":      m.ChannelID,
		"requested_count": num,
	})

	// Get recent messages from the channel
	messages, err := s.ChannelMessages(m.ChannelID, num+1, "", "", "") // +1 to include the command message
	if err != nil {
		logger.Error("Failed to fetch messages from channel", err, map[string]interface{}{
			"user_id":    m.Author.ID,
			"guild_id":   m.GuildID,
			"channel_id": m.ChannelID,
		})
		sendEmbedMessage(s, m.ChannelID, "❌ Error", "Failed to fetch messages from the channel.", 0xff0000)
		return
	}

	// Filter out messages that are too old (older than 14 days)
	var messageIDs []string
	var deletedCount int
	var skippedCount int

	for _, msg := range messages {
		// Skip the command message itself
		if msg.ID == m.ID {
			continue
		}

		// Check if message is older than 14 days
		messageTime := msg.Timestamp

		// Discord doesn't allow bulk deletion of messages older than 14 days
		if time.Since(messageTime) > 14*24*time.Hour {
			skippedCount++
			continue
		}

		messageIDs = append(messageIDs, msg.ID)
		deletedCount++
	}

	// Delete messages in bulk if there are any to delete
	if len(messageIDs) > 0 {
		err = s.ChannelMessagesBulkDelete(m.ChannelID, messageIDs)
		if err != nil {
			logger.Error("Failed to bulk delete messages", err, map[string]interface{}{
				"user_id":       m.Author.ID,
				"guild_id":      m.GuildID,
				"channel_id":    m.ChannelID,
				"message_count": len(messageIDs),
			})
			sendEmbedMessage(s, m.ChannelID, "❌ Error", "Failed to delete messages. Make sure I have 'Manage Messages' permission.", 0xff0000)
			return
		}
	}

	// Delete the command message itself
	s.ChannelMessageDelete(m.ChannelID, m.ID)

	logger.Info("Message deletion completed", map[string]interface{}{
		"user_id":       m.Author.ID,
		"guild_id":      m.GuildID,
		"channel_id":    m.ChannelID,
		"deleted_count": deletedCount,
		"skipped_count": skippedCount,
	})

	// Send confirmation message (will be deleted after 5 seconds)
	embed := &discordgo.MessageEmbed{
		Title:     "🗑️ Messages Deleted",
		Color:     0x00ff00, // Green
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae",
		},
		Description: "Messages have been successfully deleted.",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Messages Deleted",
				Value:  fmt.Sprintf("%d messages", deletedCount),
				Inline: true,
			},
			{
				Name:   "Deleted By",
				Value:  m.Author.Username,
				Inline: true,
			},
		},
	}

	// Add field for skipped messages if any
	if skippedCount > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Messages Skipped",
			Value:  fmt.Sprintf("%d messages (older than 14 days)", skippedCount),
			Inline: true,
		})
	}

	// Send the confirmation message
	confirmMsg, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		return
	}

	// Delete the confirmation message after 5 seconds
	time.AfterFunc(5*time.Second, func() {
		s.ChannelMessageDelete(m.ChannelID, confirmMsg.ID)
	})
}

// hasManageMessagesPermission checks if a user has manage messages permission
func hasManageMessagesPermission(s *discordgo.Session, guildID, userID string) bool {
	// Get guild member
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return false
	}

	// Check if user is the server owner
	guild, err := s.Guild(guildID)
	if err == nil && guild.OwnerID == userID {
		return true
	}

	// Check if user has manage messages permission
	for _, roleID := range member.Roles {
		role, err := s.State.Role(guildID, roleID)
		if err != nil {
			continue
		}
		if role.Permissions&discordgo.PermissionManageMessages != 0 {
			return true
		}
	}

	return false
}
