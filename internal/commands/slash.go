package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// RegisterSlashCommands registers all slash commands globally
func RegisterSlashCommands(s *discordgo.Session) error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "play",
			Description: "Add a song to the queue and play it",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "url",
					Description: "YouTube URL to play",
					Required:    true,
				},
			},
		},
		{
			Name:        "queue",
			Description: "Manage the music queue",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a song to the queue",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "url",
							Description: "YouTube URL to add",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "Show the current queue",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a song from the queue",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "index",
							Description: "Position of the song to remove (1-based)",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "clear",
					Description: "Clear the entire queue",
				},
			},
		},
		{
			Name:        "skip",
			Description: "Skip the current song",
		},
		{
			Name:        "stop",
			Description: "Stop playback and clear the queue",
		},
		{
			Name:        "pause",
			Description: "Pause the current playback",
		},
		{
			Name:        "resume",
			Description: "Resume paused playback",
		},
		{
			Name:        "help",
			Description: "Show help information",
		},
		{
			Name:        "servers",
			Description: "Show server information (Bot Owner Only)",
		},
		{
			Name:        "nowplaying",
			Description: "Show what's currently playing",
		},
		{
			Name:        "version",
			Description: "Show bot version information",
		},
	}

	log.Println("Registering global slash commands...")

	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Error creating command %s: %v", cmd.Name, err)
			return err
		}
		log.Printf("Registered command: %s", cmd.Name)
	}

	log.Println("All slash commands registered successfully!")
	return nil
}

// DeleteAllSlashCommands deletes all global slash commands
func DeleteAllSlashCommands(s *discordgo.Session) error {
	log.Println("Deleting all global slash commands...")

	commands, err := s.ApplicationCommands(s.State.User.ID, "")
	if err != nil {
		log.Printf("Error fetching commands: %v", err)
		return err
	}

	for _, cmd := range commands {
		err := s.ApplicationCommandDelete(s.State.User.ID, "", cmd.ID)
		if err != nil {
			log.Printf("Error deleting command %s: %v", cmd.Name, err)
			return err
		}
		log.Printf("Deleted command: %s", cmd.Name)
	}

	log.Println("All slash commands deleted successfully!")
	return nil
}

// DeleteSpecificSlashCommand deletes a specific slash command by name
func DeleteSpecificSlashCommand(s *discordgo.Session, commandName string) error {
	log.Printf("Deleting specific command: %s", commandName)

	commands, err := s.ApplicationCommands(s.State.User.ID, "")
	if err != nil {
		log.Printf("Error fetching commands: %v", err)
		return err
	}

	for _, cmd := range commands {
		if cmd.Name == commandName {
			err := s.ApplicationCommandDelete(s.State.User.ID, "", cmd.ID)
			if err != nil {
				log.Printf("Error deleting command %s: %v", cmd.Name, err)
				return err
			}
			log.Printf("Deleted command: %s", cmd.Name)
			return nil
		}
	}

	log.Printf("Command %s not found", commandName)
	return nil
}
