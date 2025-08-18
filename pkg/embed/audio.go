package embed

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// AudioEmbeds implements AudioEmbedBuilder interface
type AudioEmbeds struct {
	baseColor int
	botName   string
}

// NewAudioEmbedBuilder creates a new AudioEmbedBuilder
func NewAudioEmbedBuilder() AudioEmbedBuilder {
	return &AudioEmbeds{
		baseColor: 0x0099ff,
		botName:   "Hokko Tarumae",
	}
}

// Success creates a success embed
func (a *AudioEmbeds) Success(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
	}
}

// Error creates an error embed
func (a *AudioEmbeds) Error(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
	}
}

// Info creates an info embed
func (a *AudioEmbeds) Info(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       a.baseColor,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
	}
}

// Warning creates a warning embed
func (a *AudioEmbeds) Warning(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       0xffa500, // Orange
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
	}
}

// NowPlaying creates a now playing embed
func (a *AudioEmbeds) NowPlaying(title, url string, duration time.Duration) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "🎵 Now Playing",
		Description: fmt.Sprintf("[%s](%s)", title, url),
		Color:       0x00ff00, // Green
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
	}

	if duration > 0 {
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "Duration", Value: formatDuration(duration), Inline: true},
		}
	}

	return embed
}

// QueueStatus creates a queue status embed
func (a *AudioEmbeds) QueueStatus(current string, queue []string, queueSize int) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:     "🎵 Music Queue",
		Color:     a.baseColor,
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
	}

	var fields []*discordgo.MessageEmbedField

	// Show currently playing
	if current != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "🎶 Now Playing",
			Value:  current,
			Inline: false,
		})
	}

	// Show queue items
	if len(queue) > 0 {
		var queueText strings.Builder
		for i, item := range queue {
			if i >= 10 { // Limit to first 10 items to avoid embed limits
				queueText.WriteString(fmt.Sprintf("... and %d more songs\n", len(queue)-10))
				break
			}
			queueText.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "📋 Up Next",
			Value:  queueText.String(),
			Inline: false,
		})
	} else {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "📋 Up Next",
			Value:  "No songs in queue.",
			Inline: false,
		})
	}

	// Add queue size info
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "📊 Queue Info",
		Value:  fmt.Sprintf("Total songs: %d", queueSize),
		Inline: true,
	})

	embed.Fields = fields
	return embed
}

// PlaybackError creates a playback error embed
func (a *AudioEmbeds) PlaybackError(url string, err error) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "❌ Playback Error",
		Description: fmt.Sprintf("Failed to play: %s", url),
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Error", Value: err.Error(), Inline: false},
		},
	}
}

// IdleTimeout creates an idle timeout embed
func (a *AudioEmbeds) IdleTimeout() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "⏰ Idle Timeout",
		Description: "Bot has been idle for 5 minutes. Disconnected from voice channel to preserve resources.\nUse `/play` to start playing again!",
		Color:       0xffa500, // Orange
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
	}
}

// QueueEnded creates a queue ended embed
func (a *AudioEmbeds) QueueEnded() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "📭 Queue Ended",
		Description: "All songs in the queue have been played. Add more songs with `/play` or `/queue add`!",
		Color:       0x808080, // Gray
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
	}
}

// SongFinished creates a song finished embed
func (a *AudioEmbeds) SongFinished(title, requestedBy string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:     "🎵 Song Finished",
		Color:     0x00ff00, // Green
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Finished Playing",
				Value:  fmt.Sprintf("**%s**\nRequested by: %s", title, requestedBy),
				Inline: false,
			},
		},
	}
}

// SongSkipped creates a song skipped embed
func (a *AudioEmbeds) SongSkipped(title, requestedBy, skippedBy string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:     "⏭️ Song Skipped",
		Color:     0xffa500, // Orange
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Skipped Song",
				Value:  fmt.Sprintf("**%s**\nRequested by: %s", title, requestedBy),
				Inline: false,
			},
			{
				Name:   "Skipped By",
				Value:  skippedBy,
				Inline: false,
			},
		},
	}
}

// PlaybackStopped creates a playback stopped embed
func (a *AudioEmbeds) PlaybackStopped(stoppedBy string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "⏹️ Playback Stopped",
		Description: "Music playback has been stopped. Use `/play` to start playing again!",
		Color:       0xff0000, // Red
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: a.botName,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Stopped By",
				Value:  stoppedBy,
				Inline: false,
			},
		},
	}
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "Unknown"
	}
	
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}