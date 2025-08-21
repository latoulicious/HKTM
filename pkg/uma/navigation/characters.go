package navigation

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/uma/handler"
	"github.com/latoulicious/HKTM/pkg/uma/shared"
)

// NavigationState tracks the current state of image navigation
type NavigationState struct {
	Character    *shared.Character
	ImagesResult *shared.CharacterImagesResult
	CurrentIndex int
	MessageID    string
	ChannelID    string
}

// NavigationManager manages image navigation for Uma character embeds
type NavigationManager struct {
	activeNavigations map[string]*NavigationState
	mutex             sync.RWMutex
}

var navigationManager = &NavigationManager{
	activeNavigations: make(map[string]*NavigationState),
}

// GetNavigationManager returns the global navigation manager instance
func GetNavigationManager() *NavigationManager {
	return navigationManager
}

// HandleReaction handles reaction events for Uma character image navigation
func (nm *NavigationManager) HandleReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Only handle reactions from the bot's own messages
	if r.UserID == s.State.User.ID {
		return
	}

	// Check if this is a navigation reaction
	reaction := r.Emoji.Name
	if reaction != "⬅️" && reaction != "➡️" && reaction != "🔄" {
		return
	}

	// Get navigation state
	nm.mutex.RLock()
	state, exists := nm.activeNavigations[r.MessageID]
	nm.mutex.RUnlock()

	if !exists {
		return
	}

	// Handle navigation
	switch reaction {
	case "⬅️":
		// Previous image
		if state.CurrentIndex > 0 {
			state.CurrentIndex--
		} else {
			// Get total number of images
			totalImages := nm.getTotalImages(state.ImagesResult)
			state.CurrentIndex = totalImages - 1
		}
	case "➡️":
		// Next image
		totalImages := nm.getTotalImages(state.ImagesResult)
		if state.CurrentIndex < totalImages-1 {
			state.CurrentIndex++
		} else {
			state.CurrentIndex = 0
		}
	case "🔄":
		// Refresh images (re-fetch from API)
		client := handler.NewClient()
		imagesResult := client.GetCharacterImages(state.Character.ID)
		if imagesResult.Found {
			state.ImagesResult = imagesResult
			state.CurrentIndex = 0
		}
	}

	// Update the embed
	embed := nm.createCharacterEmbed(state.Character, state.ImagesResult, state.CurrentIndex)

	// Update the message
	_, err := s.ChannelMessageEditEmbed(state.ChannelID, state.MessageID, embed)
	if err != nil {
		fmt.Printf("Error updating Uma navigation embed: %v\n", err)
		return
	}

	// Remove the user's reaction
	s.MessageReactionRemove(state.ChannelID, state.MessageID, reaction, r.UserID)
}

// RegisterNavigation registers a new Uma character navigation
func (nm *NavigationManager) RegisterNavigation(messageID string, character *shared.Character, imagesResult *shared.CharacterImagesResult, channelID string) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	nm.activeNavigations[messageID] = &NavigationState{
		Character:    character,
		ImagesResult: imagesResult,
		CurrentIndex: 0,
		MessageID:    messageID,
		ChannelID:    channelID,
	}
}

// CleanupNavigation removes a navigation state when no longer needed
func (nm *NavigationManager) CleanupNavigation(messageID string) {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	delete(nm.activeNavigations, messageID)
}

// getTotalImages calculates the total number of images across all categories
func (nm *NavigationManager) getTotalImages(imagesResult *shared.CharacterImagesResult) int {
	totalImages := 0
	if imagesResult.Found {
		for _, category := range imagesResult.Images {
			totalImages += len(category.Images)
		}
	}
	return totalImages
}

// CreateCharacterEmbed creates an embed for character display with image navigation
func (nm *NavigationManager) CreateCharacterEmbed(character *shared.Character, imagesResult *shared.CharacterImagesResult, imageIndex int) *discordgo.MessageEmbed {
	return nm.createCharacterEmbed(character, imagesResult, imageIndex)
}

// createCharacterEmbed creates an embed for character display with image navigation
func (nm *NavigationManager) createCharacterEmbed(character *shared.Character, imagesResult *shared.CharacterImagesResult, imageIndex int) *discordgo.MessageEmbed {
	// Build footer text
	footerText := "Data from umapyoi.net"
	if imagesResult.Found && len(imagesResult.Images) > 0 {
		totalImages := 0
		for _, category := range imagesResult.Images {
			totalImages += len(category.Images)
		}
		if totalImages > 1 {
			footerText = fmt.Sprintf("Data from umapyoi.net | Image %d of %d", imageIndex+1, totalImages)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏇 Character Found",
		Description: fmt.Sprintf("**%s** (%s)", character.NameEn, character.NameJp),
		Color:       0x00ff00, // Green color
		Footer: &discordgo.MessageEmbedFooter{
			Text: footerText,
		},
		Fields: []*discordgo.MessageEmbedField{},
	}

	// Add character metadata fields
	if character.CategoryLabelEn != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Category",
			Value:  character.CategoryLabelEn,
			Inline: true,
		})
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Character ID",
		Value:  fmt.Sprintf("%d", character.ID),
		Inline: true,
	})

	// Add character image if available
	if imagesResult.Found && len(imagesResult.Images) > 0 {
		// Flatten all images from all categories for navigation
		var allImages []shared.CharacterImage
		var allCategories []string
		for _, category := range imagesResult.Images {
			for _, image := range category.Images {
				allImages = append(allImages, image)
				allCategories = append(allCategories, category.LabelEn)
			}
		}

		if len(allImages) > 0 {
			if imageIndex >= len(allImages) {
				imageIndex = 0
			}
			image := allImages[imageIndex]
			category := allCategories[imageIndex]

			// Add image details
			if len(allImages) > 1 {
				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Value:  fmt.Sprintf("**Type:** %s", category),
					Inline: false,
				})
			}

			// Add the image
			embed.Image = &discordgo.MessageEmbedImage{
				URL: image.Image,
			}
		}
	} else if character.ThumbImg != "" {
		// Fallback to thumbnail if no images available
		embed.Image = &discordgo.MessageEmbedImage{
			URL: character.ThumbImg,
		}
	}

	return embed
}
