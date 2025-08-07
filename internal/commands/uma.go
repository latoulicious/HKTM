package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/uma"
)

var umaClient = uma.NewClient()
var navigationManager = uma.GetNavigationManager()
var gametoraClient = uma.NewGametoraClient()

// UmaCommand handles Uma Musume related commands
func UmaCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "❌ Please specify a subcommand.\n\n**Usage:** `!uma char <character name>`\n**Example:** `!uma char Oguri Cap`")
		return
	}

	subcommand := strings.ToLower(args[0])

	switch subcommand {
	case "char", "character":
		CharacterCommand(s, m, args[1:])
	case "support":
		SupportCommand(s, m, args[1:])
	case "skills":
		SkillsCommand(s, m, args[1:])
	case "refresh":
		StableRefreshCommand(s, m, args[1:])
	default:
		s.ChannelMessageSend(m.ChannelID, "❌ Unknown subcommand.\n\n**Available subcommands:**\n• `char <name>` - Search for a character\n• `support <name>` - Search for a support card (list view)\n• `skills <name>` - Get skills for a support card (Gametora API)\n• `refresh` - Refresh the Gametora API build ID\n\n**Examples:**\n• `!uma char Oguri Cap`\n• `!uma support daring tact`\n• `!uma skills daring tact`\n• `!uma refresh`")
	}
}

// CharacterCommand searches for and displays character information
func CharacterCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if user provided a character name
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "❌ Please provide a character name to search for.\n\n**Usage:** `!uma char <character name>`\n**Example:** `!uma char Oguri Cap`")
		return
	}

	// Join the arguments to form the search query
	query := strings.Join(args, " ")

	// Send a loading message
	loadingMsg, _ := s.ChannelMessageSend(m.ChannelID, "🔍 Searching for character...")

	// Search for character
	result := umaClient.SearchCharacter(query)

	// Delete the loading message
	s.ChannelMessageDelete(m.ChannelID, loadingMsg.ID)

	if !result.Found {
		// Create error embed
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Character Not Found",
			Description: fmt.Sprintf("Could not find character: **%s**", query),
			Color:       0xff0000, // Red color
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Hokko Tarumae | Uma Musume Character Search",
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "💡 Tips",
					Value:  "• Try using the Japanese name\n• Check spelling and try alternative names\n• Try partial names (e.g., 'oguri' for 'Oguri Cap')",
					Inline: false,
				},
			},
		}

		if result.Error != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🔧 Error",
				Value:  result.Error.Error(),
				Inline: false,
			})
		}

		s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return
	}

	// Fetch character images
	imagesResult := umaClient.GetCharacterImages(result.Character.ID)

	// Create success embed with image navigation
	embed := navigationManager.CreateCharacterEmbed(result.Character, imagesResult, 0)

	// Send the initial embed
	msg, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Failed to send character information.")
		return
	}

	// Register navigation if there are multiple images
	totalImages := 0
	if imagesResult.Found {
		for _, category := range imagesResult.Images {
			totalImages += len(category.Images)
		}
	}

	if totalImages > 1 {
		navigationManager.RegisterNavigation(msg.ID, result.Character, imagesResult, m.ChannelID)

		// Add navigation emotes
		reactions := []string{"⬅️", "➡️", "🔄"}
		for _, reaction := range reactions {
			s.MessageReactionAdd(m.ChannelID, msg.ID, reaction)
		}
	}
}

// SupportCommand searches for and displays support card information
func SupportCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if user provided a support card name
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "❌ Please provide a support card name to search for.\n\n**Usage:** `!uma support <support card name>`\n**Example:** `!uma support daring tact`")
		return
	}

	// Join the arguments to form the search query
	query := strings.Join(args, " ")

	// Send a loading message
	loadingMsg, _ := s.ChannelMessageSend(m.ChannelID, "🔍 Searching for support card...")

	// Search for support card
	result := umaClient.SearchSupportCard(query)

	// Delete the loading message
	s.ChannelMessageDelete(m.ChannelID, loadingMsg.ID)

	if !result.Found {
		// Create error embed
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Support Card Not Found",
			Description: fmt.Sprintf("Could not find support card: **%s**", query),
			Color:       0xff0000, // Red color
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Hokko Tarumae | Uma Musume Support Card Search",
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "💡 Tips",
					Value:  "• Try using the English title\n• Try using the Japanese title\n• Try using the gametora identifier\n• Check spelling and try alternative names",
					Inline: false,
				},
			},
		}

		if result.Error != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🔧 Error",
				Value:  result.Error.Error(),
				Inline: false,
			})
		}

		s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return
	}

	// Create success embed
	embed := createSupportCardEmbed(result.SupportCard)

	// If there are multiple versions, add them to the embed
	if len(result.SupportCards) > 1 {
		embed = createMultiVersionSupportCardEmbed(result.SupportCards)
	}

	// Send the embed
	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Failed to send support card information.")
		return
	}
}

// createSupportCardEmbed creates an embed for a support card
func createSupportCardEmbed(supportCard *uma.SupportCard) *discordgo.MessageEmbed {
	// Determine embed color based on rarity
	var color int
	switch supportCard.RarityString {
	case "SSR":
		color = 0xFFD700 // Gold
	case "SR":
		color = 0xC0C0C0 // Silver
	case "R":
		color = 0xCD7F32 // Bronze
	default:
		color = 0x7289DA // Default Discord blue
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       supportCard.TitleEn,
		Description: supportCard.Title,
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Data from umapyoi.net",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🎴 Rarity",
				Value:  supportCard.RarityString,
				Inline: true,
			},
			{
				Name:   "🎯 Type",
				Value:  supportCard.Type,
				Inline: true,
			},
			{
				Name:   "🆔 Card ID",
				Value:  fmt.Sprintf("%d", supportCard.ID),
				Inline: true,
			},
			{
				Name:   "👤 Character ID",
				Value:  fmt.Sprintf("%d", supportCard.CharaID),
				Inline: true,
			},
			{
				Name:   "🔗 Gametora",
				Value:  supportCard.Gametora,
				Inline: true,
			},
		},
	}

	// Add type icon if available
	if supportCard.TypeIconURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: supportCard.TypeIconURL,
		}
	}

	return embed
}

// SkillsCommand retrieves skills for a support card using the Gametora API
func SkillsCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if user provided a support card name
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "❌ Please provide a support card name to get skills for.\n\n**Usage:** `!uma skills <support card name>`\n**Example:** `!uma skills daring tact`")
		return
	}

	// Join the arguments to form the search query
	query := strings.Join(args, " ")

	// Send a loading message
	loadingMsg, _ := s.ChannelMessageSend(m.ChannelID, "🔍 Searching for support card skills using Gametora API...")

	// Search for support card using Gametora API
	result := gametoraClient.SearchSimplifiedSupportCard(query)

	// Delete the loading message
	s.ChannelMessageDelete(m.ChannelID, loadingMsg.ID)

	if !result.Found {
		// Create error embed
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Support Card Not Found",
			Description: fmt.Sprintf("Could not find support card: **%s**", query),
			Color:       0xff0000, // Red color
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Hokko Tarumae | Uma Musume Support Card Skills (Gametora API)",
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "💡 Tips",
					Value:  "• Try using the English title\n• Try using the Japanese title\n• Try using the gametora identifier\n• Check spelling and try alternative names\n• Try partial names",
					Inline: false,
				},
			},
		}

		if result.Error != nil {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🔧 Error",
				Value:  result.Error.Error(),
				Inline: false,
			})
		}

		s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return
	}

	// Create success embed
	embed := createSimplifiedSkillsEmbed(result.SupportCard)

	// Send the embed
	_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Failed to send support card skills.")
		return
	}
}

// createSimplifiedSkillsEmbed creates a simplified embed showing only skills for a support card
func createSimplifiedSkillsEmbed(supportCard *uma.SimplifiedSupportCard) *discordgo.MessageEmbed {
	// Determine embed color based on rarity
	var color int
	switch supportCard.Rarity {
	case 3: // SSR
		color = 0xFFD700 // Gold
	case 2: // SR
		color = 0xC0C0C0 // Silver
	case 1: // R
		color = 0xCD7F32 // Bronze
	default:
		color = 0x7289DA // Default Discord blue
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       supportCard.NameJp,
		Description: fmt.Sprintf("**Character:** %s", supportCard.CharName),
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Data from Gametora API | Hokko Tarumae",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🎴 Rarity",
				Value:  fmt.Sprintf("%d", supportCard.Rarity),
				Inline: true,
			},
			{
				Name:   "🎯 Type",
				Value:  supportCard.Type,
				Inline: true,
			},
			{
				Name:   "🆔 Support ID",
				Value:  fmt.Sprintf("%d", supportCard.SupportID),
				Inline: true,
			},
		},
	}

	// Add obtained info if available
	if supportCard.Obtained != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📦 Obtained",
			Value:  supportCard.Obtained,
			Inline: true,
		})
	}

	// Add card image if available
	if supportCard.URLName != "" {
		imageURL := gametoraClient.GetSupportCardImageURL(supportCard.URLName)
		if imageURL != "" {
			embed.Image = &discordgo.MessageEmbedImage{
				URL: imageURL,
			}
		}
	}

	// Add support hints if available
	if len(supportCard.Hints.HintSkills) > 0 {
		var hintsText strings.Builder
		for i, hint := range supportCard.Hints.HintSkills {
			hintsText.WriteString(fmt.Sprintf("• %s", hint.NameEn))
			if i < len(supportCard.Hints.HintSkills)-1 {
				hintsText.WriteString("\n")
			}
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("💡 Support Hints (%d)", len(supportCard.Hints.HintSkills)),
			Value:  hintsText.String(),
			Inline: false,
		})
	}

	// Add event skills if available
	if len(supportCard.EventSkills) > 0 {
		var eventsText strings.Builder
		for i, event := range supportCard.EventSkills {
			eventsText.WriteString(fmt.Sprintf("• %s", event.NameEn))
			if i < len(supportCard.EventSkills)-1 {
				eventsText.WriteString("\n")
			}
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("🎉 Event Skills (%d)", len(supportCard.EventSkills)),
			Value:  eventsText.String(),
			Inline: false,
		})
	}

	return embed
}

// StableRefreshCommand refreshes the build ID for the Gametora API
func StableRefreshCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Send a loading message
	loadingMsg, _ := s.ChannelMessageSend(m.ChannelID, "🔄 Refreshing Gametora API build ID...")

	// Refresh the build ID
	buildID, err := gametoraClient.GetBuildID()

	// Delete the loading message
	s.ChannelMessageDelete(m.ChannelID, loadingMsg.ID)

	if err != nil {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Build ID Refresh Failed",
			Description: fmt.Sprintf("Failed to refresh build ID: **%v**", err),
			Color:       0xff0000, // Red color
			Timestamp:   time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Hokko Tarumae | Gametora API Build ID Refresh",
			},
		}
		s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return
	}

	// Success embed
	embed := &discordgo.MessageEmbed{
		Title:       "✅ Build ID Refreshed",
		Description: fmt.Sprintf("Successfully refreshed the build ID for the Gametora API.\n\n**Build ID:** `%s`", buildID),
		Color:       0x00ff00, // Green color
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Hokko Tarumae | Gametora API Build ID Refresh",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💡 Tip",
				Value:  "The Gametora API should now work with the latest data. Try using `!uma stats <card name>` to test.",
				Inline: false,
			},
		},
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// createMultiVersionSupportCardEmbed creates an embed showing all versions of a support card
func createMultiVersionSupportCardEmbed(supportCards []uma.SupportCard) *discordgo.MessageEmbed {
	// Use the highest rarity card for the main embed info
	mainCard := supportCards[0]

	// Determine embed color based on highest rarity
	var color int
	switch mainCard.RarityString {
	case "SSR":
		color = 0xFFD700 // Gold
	case "SR":
		color = 0xC0C0C0 // Silver
	case "R":
		color = 0xCD7F32 // Bronze
	default:
		color = 0x7289DA // Default Discord blue
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       mainCard.TitleEn,
		Description: mainCard.Title,
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Data from umapyoi.net",
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🎯 Type",
				Value:  mainCard.Type,
				Inline: true,
			},
			{
				Name:   "👤 Character ID",
				Value:  fmt.Sprintf("%d", mainCard.CharaID),
				Inline: true,
			},
		},
	}

	// Add type icon if available
	if mainCard.TypeIconURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: mainCard.TypeIconURL,
		}
	}

	// Add all versions as fields
	var versionsText strings.Builder
	for i, card := range supportCards {
		rarityEmoji := "🎴"
		switch card.RarityString {
		case "SSR":
			rarityEmoji = "⭐"
		case "SR":
			rarityEmoji = "✨"
		case "R":
			rarityEmoji = "🎴"
		}

		versionsText.WriteString(fmt.Sprintf("%s **%s**\n", rarityEmoji, card.RarityString))
		versionsText.WriteString(fmt.Sprintf("• ID: %d\n", card.ID))
		versionsText.WriteString(fmt.Sprintf("• Title: %s\n", card.TitleEn))
		if card.Title != card.TitleEn {
			versionsText.WriteString(fmt.Sprintf("• JP: %s\n", card.Title))
		}
		versionsText.WriteString(fmt.Sprintf("• Gametora: %s\n", card.Gametora))

		if i < len(supportCards)-1 {
			versionsText.WriteString("\n")
		}
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   fmt.Sprintf("📋 All Versions (%d)", len(supportCards)),
		Value:  versionsText.String(),
		Inline: false,
	})

	return embed
}
