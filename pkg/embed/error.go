package embed

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ErrorEmbeds implements EmbedBuilder interface for error-specific embeds
type ErrorEmbeds struct{}

// NewErrorEmbedBuilder creates a new ErrorEmbeds instance
func NewErrorEmbedBuilder() EmbedBuilder {
	return &ErrorEmbeds{}
}

// Success creates a success embed (not typically used in error context)
func (e *ErrorEmbeds) Success(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Error creates a standard error embed
func (e *ErrorEmbeds) Error(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Info creates an info embed
func (e *ErrorEmbeds) Info(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x7289da, // Discord blurple
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Warning creates a warning embed
func (e *ErrorEmbeds) Warning(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xffaa00, // Orange
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// CommandError creates an embed for command execution errors
func (e *ErrorEmbeds) CommandError(command string, err error) *discordgo.MessageEmbed {
	title := fmt.Sprintf("❌ Command Error: %s", command)
	description := "An error occurred while executing the command."

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if err != nil {
		errorMsg := err.Error()
		// Truncate very long error messages
		if len(errorMsg) > 1000 {
			errorMsg = errorMsg[:997] + "..."
		}

		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Error Details",
				Value:  errorMsg,
				Inline: false,
			},
		}
	}

	return embed
}

// ValidationError creates an embed for validation errors
func (e *ErrorEmbeds) ValidationError(field string, message string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "⚠️ Validation Error",
		Description: fmt.Sprintf("Invalid %s: %s", field, message),
		Color:       0xffaa00, // Orange
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// PermissionError creates an embed for permission errors
func (e *ErrorEmbeds) PermissionError(action string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "🔒 Permission Error",
		Description: fmt.Sprintf("You don't have permission to %s.", action),
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// NetworkError creates an embed for network-related errors
func (e *ErrorEmbeds) NetworkError(operation string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "🌐 Network Error",
		Description: fmt.Sprintf("Network error occurred during %s. Please try again later.", operation),
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}
