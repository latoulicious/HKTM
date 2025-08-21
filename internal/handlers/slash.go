package handlers

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/internal/commands"
)

// SlashCommandHandler handles slash command interactions
func SlashCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Ignore interactions from bots
	if i.Member.User.Bot {
		return
	}

	// Handle different command types
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleApplicationCommand(s, i)
	case discordgo.InteractionApplicationCommandAutocomplete:
		handleAutocomplete(s, i)
	default:
		log.Printf("Unknown interaction type: %d", i.Type)
	}
}

// handleApplicationCommand handles application command interactions
func handleApplicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	// Acknowledge the interaction immediately
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error acknowledging interaction: %v", err)
		return
	}

	var response string

	switch data.Name {
	case "play":
		response = handlePlaySlash(s, i, data)
	case "queue":
		response = handleQueueSlash(s, i, data)
	case "skip":
		response = handleSkipSlash(s, i)
	case "stop":
		response = handleStopSlash(s, i)
	case "pause":
		response = handlePauseSlash(s, i)
	case "resume":
		response = handleResumeSlash(s, i)
	case "servers":
		response = handleServersSlash(s, i)
	case "help":
		response = handleHelpSlash(s, i)
	case "nowplaying":
		response = handleNowPlayingSlash(s, i)
	case "version":
		response = handleVersionSlash(s, i)
	default:
		response = "❌ Unknown command."
	}

	// Send the response
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &response,
	})
	if err != nil {
		log.Printf("Error sending interaction response: %v", err)
	}
}

// handleAutocomplete handles autocomplete interactions
func handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()

	var choices []*discordgo.ApplicationCommandOptionChoice

	switch data.Name {
	case "play":
		choices = handlePlayAutocomplete(data)
	case "queue":
		choices = handleQueueAutocomplete(data)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	if err != nil {
		log.Printf("Error sending autocomplete response: %v", err)
	}
}

// Slash command handlers
func handlePlaySlash(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) string {
	// Get URL from options
	var url string
	for _, option := range data.Options {
		if option.Name == "url" {
			url = option.StringValue()
			break
		}
	}

	if url == "" {
		return "❌ Please provide a YouTube URL."
	}

	// Create a mock message for compatibility with existing commands
	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    i.Member.User,
		},
	}

	// Call the existing play command logic
	commands.PlayCommand(s, mockMessage, []string{url})

	return "✅ Song added to queue!"
}

func handleQueueSlash(s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) string {
	// Get subcommand from options
	var subcommand string
	var args []string

	for _, option := range data.Options {
		switch option.Name {
		case "action":
			subcommand = option.StringValue()
		case "url":
			args = append(args, option.StringValue())
		case "index":
			args = append(args, option.StringValue())
		}
	}

	// Create a mock message for compatibility
	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    i.Member.User,
		},
	}

	// Call the existing queue command logic
	commands.QueueCommand(s, mockMessage, append([]string{subcommand}, args...))

	return "✅ Queue command executed!"
}

func handleSkipSlash(s *discordgo.Session, i *discordgo.InteractionCreate) string {
	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    i.Member.User,
		},
	}

	commands.SkipCommand(s, mockMessage)
	return "⏭️ Skipped current song!"
}

func handleStopSlash(s *discordgo.Session, i *discordgo.InteractionCreate) string {
	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    i.Member.User,
		},
	}

	commands.StopCommand(s, mockMessage, []string{})
	return "⏹️ Stopped playback and cleared queue!"
}

func handlePauseSlash(s *discordgo.Session, i *discordgo.InteractionCreate) string {
	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    i.Member.User,
		},
	}

	commands.PauseCommand(s, mockMessage)
	return "⏸️ Paused playback!"
}

func handleResumeSlash(s *discordgo.Session, i *discordgo.InteractionCreate) string {
	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    i.Member.User,
		},
	}

	commands.ResumeCommand(s, mockMessage)
	return "▶️ Resumed playback!"
}

func handleServersSlash(s *discordgo.Session, i *discordgo.InteractionCreate) string {
	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    i.Member.User,
		},
	}

	commands.ServersCommand(s, mockMessage)
	return "📊 Server information displayed!"
}

func handleHelpSlash(s *discordgo.Session, i *discordgo.InteractionCreate) string {
	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    i.Member.User,
		},
	}

	commands.ShowHelpCommand(s, mockMessage)
	return "📖 Help information displayed!"
}

func handleNowPlayingSlash(s *discordgo.Session, i *discordgo.InteractionCreate) string {
	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    i.Member.User,
		},
	}

	commands.NowPlayingCommand(s, mockMessage)
	return "🎵 Now playing information displayed!"
}

func handleVersionSlash(s *discordgo.Session, i *discordgo.InteractionCreate) string {
	mockMessage := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			GuildID:   i.GuildID,
			ChannelID: i.ChannelID,
			Author:    i.Member.User,
		},
	}

	commands.VersionCommand(s, mockMessage)
	return "ℹ️ Version information displayed!"
}

// Autocomplete handlers
func handlePlayAutocomplete(_ discordgo.ApplicationCommandInteractionData) []*discordgo.ApplicationCommandOptionChoice {
	// Return some common YouTube URLs or search suggestions
	return []*discordgo.ApplicationCommandOptionChoice{
		{
			Name:  "YouTube URL",
			Value: "https://youtube.com/watch?v=",
		},
	}
}

func handleQueueAutocomplete(_ discordgo.ApplicationCommandInteractionData) []*discordgo.ApplicationCommandOptionChoice {
	return []*discordgo.ApplicationCommandOptionChoice{
		{
			Name:  "add",
			Value: "add",
		},
		{
			Name:  "list",
			Value: "list",
		},
		{
			Name:  "remove",
			Value: "remove",
		},
		{
			Name:  "clear",
			Value: "clear",
		},
	}
}
