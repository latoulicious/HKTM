package navigation

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/uma/handler"
	"github.com/latoulicious/HKTM/pkg/uma/shared"
)

// SupportCardNavigationState tracks the current state of support card version navigation
type SupportCardNavigationState struct {
	SupportCards []*shared.SimplifiedSupportCard
	CurrentIndex int
	MessageID    string
	ChannelID    string
	Query        string
}

// SupportCardNavigationManager manages version navigation for support card embeds
type SupportCardNavigationManager struct {
	activeNavigations map[string]*SupportCardNavigationState
	mutex             sync.RWMutex
}

var supportCardNavigationManager = &SupportCardNavigationManager{
	activeNavigations: make(map[string]*SupportCardNavigationState),
}

// HandleSupportCardReaction handles reaction events for support card version navigation
func (scnm *SupportCardNavigationManager) HandleSupportCardReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Only handle reactions from the bot's own messages
	if r.UserID == s.State.User.ID {
		return
	}

	// Check if this is a navigation reaction
	reaction := r.Emoji.Name

	// Define all possible number reactions (1️⃣ through 9️⃣)
	validReactions := []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔄"}

	isValidReaction := false
	for _, validReaction := range validReactions {
		if reaction == validReaction {
			isValidReaction = true
			break
		}
	}

	if !isValidReaction {
		return
	}

	// Get navigation state
	scnm.mutex.RLock()
	state, exists := scnm.activeNavigations[r.MessageID]
	scnm.mutex.RUnlock()

	if !exists {
		return
	}

	// Handle navigation
	switch reaction {
	case "1️⃣":
		// First version
		state.CurrentIndex = 0
	case "2️⃣":
		// Second version
		if len(state.SupportCards) > 1 {
			state.CurrentIndex = 1
		}
	case "3️⃣":
		// Third version
		if len(state.SupportCards) > 2 {
			state.CurrentIndex = 2
		}
	case "4️⃣":
		// Fourth version
		if len(state.SupportCards) > 3 {
			state.CurrentIndex = 3
		}
	case "5️⃣":
		// Fifth version
		if len(state.SupportCards) > 4 {
			state.CurrentIndex = 4
		}
	case "6️⃣":
		// Sixth version
		if len(state.SupportCards) > 5 {
			state.CurrentIndex = 5
		}
	case "7️⃣":
		// Seventh version
		if len(state.SupportCards) > 6 {
			state.CurrentIndex = 6
		}
	case "8️⃣":
		// Eighth version
		if len(state.SupportCards) > 7 {
			state.CurrentIndex = 7
		}
	case "9️⃣":
		// Ninth version
		if len(state.SupportCards) > 8 {
			state.CurrentIndex = 8
		}
	case "🔄":
		// Refresh (re-search)
		client := handler.GetGametoraClient()
		if client != nil {
			result := client.SearchSimplifiedSupportCard(state.Query)
			if result.Found && len(result.SupportCards) > 0 {
				state.SupportCards = result.SupportCards
				state.CurrentIndex = 0
			}
		}
	}

	// Update the embed
	embed := scnm.createSupportCardEmbed(state.SupportCards[state.CurrentIndex], state.SupportCards, state.CurrentIndex)

	// Update the message
	_, err := s.ChannelMessageEditEmbed(state.ChannelID, state.MessageID, embed)
	if err != nil {
		fmt.Printf("Error updating support card navigation embed: %v\n", err)
		return
	}

	// Remove the user's reaction
	s.MessageReactionRemove(state.ChannelID, state.MessageID, reaction, r.UserID)
}

// RegisterSupportCardNavigation registers a new support card version navigation
func (scnm *SupportCardNavigationManager) RegisterSupportCardNavigation(messageID string, supportCards []*shared.SimplifiedSupportCard, channelID string, query string) {
	scnm.mutex.Lock()
	defer scnm.mutex.Unlock()

	scnm.activeNavigations[messageID] = &SupportCardNavigationState{
		SupportCards: supportCards,
		CurrentIndex: 0,
		MessageID:    messageID,
		ChannelID:    channelID,
		Query:        query,
	}
}

// CleanupSupportCardNavigation removes a support card navigation state
func (scnm *SupportCardNavigationManager) CleanupSupportCardNavigation(messageID string) {
	scnm.mutex.Lock()
	defer scnm.mutex.Unlock()

	delete(scnm.activeNavigations, messageID)
}

// CreateSupportCardEmbed creates an embed for support card display with version navigation
func (scnm *SupportCardNavigationManager) CreateSupportCardEmbed(supportCard *shared.SimplifiedSupportCard, allCards []*shared.SimplifiedSupportCard, currentIndex int) *discordgo.MessageEmbed {
	return scnm.createSupportCardEmbed(supportCard, allCards, currentIndex)
}

// createSupportCardEmbed creates an embed for support card display with version navigation
func (scnm *SupportCardNavigationManager) createSupportCardEmbed(supportCard *shared.SimplifiedSupportCard, allCards []*shared.SimplifiedSupportCard, currentIndex int) *discordgo.MessageEmbed {
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

	// Build footer text
	footerText := "Data from Gametora API"
	if len(allCards) > 1 {
		footerText = fmt.Sprintf("Data from Gametora API | %s Version (%d of %d)", shared.GetRarityText(supportCard.Rarity), currentIndex+1, len(allCards))
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       supportCard.NameJp,
		Description: fmt.Sprintf("**Character:** %s", supportCard.CharName),
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footerText,
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🎴 Rarity",
				Value:  shared.GetRarityText(supportCard.Rarity),
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
		client := handler.GetGametoraClient()
		if client != nil {
			imageURL := client.GetSupportCardImageURL(supportCard.URLName)
			if imageURL != "" {
				embed.Image = &discordgo.MessageEmbedImage{
					URL: imageURL,
				}
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

	// Add version info if multiple versions exist
	if len(allCards) > 1 {
		var versionsText strings.Builder
		for i, card := range allCards {
			indicator := "○"
			if i == currentIndex {
				indicator = "●"
			}

			versionsText.WriteString(fmt.Sprintf("%s %s (%s) - ID: %d", indicator, card.NameJp, shared.GetRarityText(card.Rarity), card.SupportID))
			if i < len(allCards)-1 {
				versionsText.WriteString("\n")
			}
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("📋 Available Versions (%d)", len(allCards)),
			Value:  versionsText.String(),
			Inline: false,
		})
	}

	return embed
}

// GetSupportCardNavigationManager returns the global support card navigation manager instance
func GetSupportCardNavigationManager() *SupportCardNavigationManager {
	return supportCardNavigationManager
}
