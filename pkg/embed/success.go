package embed

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// SuccessEmbeds implements EmbedBuilder interface for success-specific embeds
type SuccessEmbeds struct{}

// NewSuccessEmbedBuilder creates a new SuccessEmbeds instance
func NewSuccessEmbedBuilder() EmbedBuilder {
	return &SuccessEmbeds{}
}

// Success creates a standard success embed
func (s *SuccessEmbeds) Success(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Error creates an error embed (not typically used in success context)
func (s *SuccessEmbeds) Error(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Info creates an info embed
func (s *SuccessEmbeds) Info(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x7289da, // Discord blurple
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Warning creates a warning embed
func (s *SuccessEmbeds) Warning(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xffaa00, // Orange
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// CommandSuccess creates an embed for successful command execution
func (s *SuccessEmbeds) CommandSuccess(command string, message string) *discordgo.MessageEmbed {
	title := fmt.Sprintf("✅ %s", command)

	return &discordgo.MessageEmbed{
		Title:       title,
		Description: message,
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// QueueAdded creates an embed for when a song is added to the queue
func (s *SuccessEmbeds) QueueAdded(title string, position int) *discordgo.MessageEmbed {
	description := fmt.Sprintf("**%s** has been added to the queue", title)

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Added to Queue",
		Description: description,
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if position > 0 {
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Position in Queue",
				Value:  fmt.Sprintf("#%d", position),
				Inline: true,
			},
		}
	}

	return embed
}

// ConfigurationUpdated creates an embed for configuration updates
func (s *SuccessEmbeds) ConfigurationUpdated(setting string, value string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "⚙️ Configuration Updated",
		Description: fmt.Sprintf("**%s** has been set to: `%s`", setting, value),
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// OperationComplete creates an embed for completed operations
func (s *SuccessEmbeds) OperationComplete(operation string, details string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("✅ %s Complete", operation),
		Color:     0x00ff00, // Green
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if details != "" {
		embed.Description = details
	}

	return embed
}
