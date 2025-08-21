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

// AudioEmbedBuilder provides audio-specific embed creation functionality
type AudioEmbedBuilder interface {
	EmbedBuilder
	NowPlaying(title, url string, duration time.Duration) *discordgo.MessageEmbed
	QueueStatus(current string, queue []string, queueSize int) *discordgo.MessageEmbed
	PlaybackError(url string, err error) *discordgo.MessageEmbed
	IdleTimeout() *discordgo.MessageEmbed
	QueueEnded() *discordgo.MessageEmbed
	SongFinished(title, requestedBy string) *discordgo.MessageEmbed
	SongSkipped(title, requestedBy, skippedBy string) *discordgo.MessageEmbed
	PlaybackStopped(stoppedBy string) *discordgo.MessageEmbed
}