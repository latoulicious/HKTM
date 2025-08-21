package handlers

import (
	"math/rand"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/internal/commands"
)

func MessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Add comprehensive nil checks
	if s == nil || m == nil || m.Author == nil {
		return
	}

	// Ignore all messages created by the bot itself
	if s.State.User != nil && m.Author.ID == s.State.User.ID {
		return
	}

	// Check if the bot is mentioned
	if s.State.User != nil && (m.MentionEveryone || len(m.Mentions) > 0) {
		for _, mention := range m.Mentions {
			if mention != nil && mention.ID == s.State.User.ID {
				// Randomly choose between two responses
				responses := []string{
					"I'm Hokko Tarumae, Tomakomai's Tourism Ambassador!★",
					"Hmm, would ah look cuter if ah was lookin' up more?",
					"A paper-winged migrating bird from the port in the north ♪ The name's Hokko Tarumae, Tomakomai's local-dol, eh! ...Yeah, maybe I should work on it more",
				}
				randomResponse := responses[rand.Intn(len(responses))]
				s.ChannelMessageSend(m.ChannelID, randomResponse)
				return
			}
		}
	}

	// Check if the message is a command
	if m.Content != "" && strings.HasPrefix(m.Content, "!") {
		args := strings.Split(m.Content, " ")
		command := strings.TrimPrefix(args[0], "!")

		switch command {
		case "help", "h":
			commands.ShowHelpCommand(s, m)
		case "play", "p":
			commands.PlayCommand(s, m, args[1:])
		case "pause":
			commands.PauseCommand(s, m)
		case "resume":
			commands.ResumeCommand(s, m)
		case "skip":
			commands.SkipCommand(s, m)
		case "stop":
			commands.StopCommand(s, m, args[1:])
		case "servers":
			commands.ServersCommand(s, m)
		case "leave":
			commands.LeaveCommand(s, m, args[1:])
		case "queue", "q":
			commands.QueueCommand(s, m, args[1:])
		case "clear":
			commands.ClearCommand(s, m, args[1:])
		case "shuffle":
			commands.ShuffleCommand(s, m, args[1:])
		case "about":
			commands.AboutCommand(s, m)
		case "nowplaying", "np":
			commands.NowPlayingCommand(s, m)
		case "gremlin":
			commands.GremlinCommand(s, m)
		case "uma":
			commands.UmaCommand(s, m, args[1:])
		case "utility":
			commands.UtilityCommand(s, m, args[1:])
		case "delete":
			commands.DeleteCommand(s, m, args[1:])
		case "version", "v":
			commands.VersionCommand(s, m)
		default:
			s.ChannelMessageSend(m.ChannelID, "Unknown command. Try `!help` to see all available commands.")
		}
	}
}
