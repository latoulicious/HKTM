package main

import (
	"flag"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/latoulicious/HKTM/internal/commands"
	"github.com/latoulicious/HKTM/internal/config"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Parse command line flags
	action := flag.String("action", "", "Action to perform: register, delete-all, delete-specific, check")
	commandName := flag.String("command", "", "Command name for delete-specific action")
	flag.Parse()

	// Create Discord session
	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}

	// Open connection
	err = dg.Open()
	if err != nil {
		log.Fatalf("Failed to open Discord session: %v", err)
	}
	defer dg.Close()

	// Perform the requested action
	switch *action {
	case "register":
		log.Println("Registering slash commands...")
		err = commands.RegisterSlashCommands(dg)
		if err != nil {
			log.Fatalf("Failed to register slash commands: %v", err)
		}
		log.Println("✅ Slash commands registered successfully!")

	case "delete-all":
		log.Println("Deleting all slash commands...")
		err = commands.DeleteAllSlashCommands(dg)
		if err != nil {
			log.Fatalf("Failed to delete slash commands: %v", err)
		}
		log.Println("✅ All slash commands deleted successfully!")

	case "delete-specific":
		if *commandName == "" {
			log.Fatal("Please provide a command name with -command flag")
		}
		log.Printf("Deleting specific command: %s", *commandName)
		err = commands.DeleteSpecificSlashCommand(dg, *commandName)
		if err != nil {
			log.Fatalf("Failed to delete command: %v", err)
		}
		log.Printf("✅ Command '%s' deleted successfully!", *commandName)

	case "check":
		log.Println("Checking all registered commands...")
		checkCommands(dg)

	default:
		log.Println("Usage:")
		log.Println("  go run tools/slash-manager.go -action register")
		log.Println("  go run tools/slash-manager.go -action delete-all")
		log.Println("  go run tools/slash-manager.go -action delete-specific -command play")
		log.Println("  go run tools/slash-manager.go -action check")
		os.Exit(1)
	}
}

// checkCommands lists all currently registered commands
func checkCommands(s *discordgo.Session) {
	// Get all global commands
	log.Println("=== GLOBAL COMMANDS ===")
	globalCommands, err := s.ApplicationCommands(s.State.User.ID, "")
	if err != nil {
		log.Printf("Error fetching global commands: %v", err)
	} else {
		if len(globalCommands) == 0 {
			log.Println("No global commands found.")
		} else {
			for _, cmd := range globalCommands {
				log.Printf("Global: %s (ID: %s) - %s", cmd.Name, cmd.ID, cmd.Description)
			}
		}
	}

	// Get all guild commands (if any)
	log.Println("\n=== GUILD COMMANDS ===")
	guilds := s.State.Guilds
	for _, guild := range guilds {
		guildCommands, err := s.ApplicationCommands(s.State.User.ID, guild.ID)
		if err != nil {
			log.Printf("Error fetching commands for guild %s: %v", guild.Name, err)
			continue
		}

		if len(guildCommands) > 0 {
			log.Printf("Guild: %s (ID: %s)", guild.Name, guild.ID)
			for _, cmd := range guildCommands {
				log.Printf("  - %s (ID: %s) - %s", cmd.Name, cmd.ID, cmd.Description)
			}
		}
	}

	log.Println("\n=== SUMMARY ===")
	log.Printf("Total global commands: %d", len(globalCommands))

	totalGuildCommands := 0
	for _, guild := range guilds {
		guildCommands, _ := s.ApplicationCommands(s.State.User.ID, guild.ID)
		totalGuildCommands += len(guildCommands)
	}
	log.Printf("Total guild commands: %d", totalGuildCommands)
}
