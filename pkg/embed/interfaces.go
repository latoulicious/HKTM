package embed

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// EmbedBuilder provides basic embed creation functionality
type EmbedBuilder interface {
	Success(title, description string) *discordgo.MessageEmbed
	Error(title, description string) *discordgo.MessageEmbed
	Info(title, description string) *discordgo.MessageEmbed
	Warning(title, description string) *discordgo.MessageEmbed
}

// AudioEmbedBuilder extends EmbedBuilder with audio-specific embed functionality
type AudioEmbedBuilder interface {
	EmbedBuilder
	NowPlaying(title, url string, duration time.Duration) *discordgo.MessageEmbed
	QueueStatus(current string, queue []string) *discordgo.MessageEmbed
	PlaybackError(url string, err error) *discordgo.MessageEmbed
	QueueEmpty() *discordgo.MessageEmbed
	AudioStopped() *discordgo.MessageEmbed
}

// EmbedFactory creates different types of embed builders
type EmbedFactory interface {
	CreateAudioEmbedBuilder() AudioEmbedBuilder
	CreateBasicEmbedBuilder() EmbedBuilder
}
