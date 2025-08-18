package embed

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// AudioEmbeds implements AudioEmbedBuilder interface
type AudioEmbeds struct {
	baseColor int
}

// NewAudioEmbedBuilder creates a new AudioEmbeds instance
func NewAudioEmbedBuilder() AudioEmbedBuilder {
	return &AudioEmbeds{
		baseColor: 0x7289da, // Discord blurple
	}
}

// Success creates a success embed
func (a *AudioEmbeds) Success(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Error creates an error embed
func (a *AudioEmbeds) Error(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Info creates an info embed
func (a *AudioEmbeds) Info(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       a.baseColor, // Discord blurple
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Warning creates a warning embed
func (a *AudioEmbeds) Warning(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xffaa00, // Orange
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// NowPlaying creates an embed for currently playing audio
func (a *AudioEmbeds) NowPlaying(title, url string, duration time.Duration) *discordgo.MessageEmbed {
	description := fmt.Sprintf("[%s](%s)", title, url)

	embed := &discordgo.MessageEmbed{
		Title:       "🎵 Now Playing",
		Description: description,
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	if duration > 0 {
		embed.Fields = []*discordgo.MessageEmbedField{
			{
				Name:   "Duration",
				Value:  formatDuration(duration),
				Inline: true,
			},
		}
	}

	return embed
}

// QueueStatus creates an embed showing the current queue status
func (a *AudioEmbeds) QueueStatus(current string, queue []string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:     "📋 Queue Status",
		Color:     a.baseColor,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if current != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🎵 Currently Playing",
			Value:  current,
			Inline: false,
		})
	}

	if len(queue) > 0 {
		queueText := ""
		maxItems := 10 // Limit to prevent embed from being too long

		for i, item := range queue {
			if i >= maxItems {
				queueText += fmt.Sprintf("... and %d more", len(queue)-maxItems)
				break
			}
			queueText += fmt.Sprintf("%d. %s\n", i+1, item)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📝 Up Next",
			Value:  queueText,
			Inline: false,
		})
	} else if current == "" {
		embed.Description = "Queue is empty"
	}

	return embed
}

// PlaybackError creates an embed for playback errors
func (a *AudioEmbeds) PlaybackError(url string, err error) *discordgo.MessageEmbed {
	description := fmt.Sprintf("Failed to play: %s", url)

	embed := &discordgo.MessageEmbed{
		Title:       "❌ Playback Error",
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

// QueueEmpty creates an embed for when the queue becomes empty
func (a *AudioEmbeds) QueueEmpty() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "📭 Queue Empty",
		Description: "The music queue is now empty. Add more songs to continue playing!",
		Color:       0xffaa00, // Orange
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// AudioStopped creates an embed for when audio playback is stopped
func (a *AudioEmbeds) AudioStopped() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "⏹️ Playback Stopped",
		Description: "Audio playback has been stopped.",
		Color:       0x808080, // Gray
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	if minutes < 60 {
		return fmt.Sprintf("%d:%02d", minutes, seconds)
	}

	hours := minutes / 60
	minutes = minutes % 60
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}
