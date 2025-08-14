package service

import (
	"fmt"

	"github.com/latoulicious/HKTM/pkg/database/models"
	"github.com/latoulicious/HKTM/pkg/uma"
	"github.com/latoulicious/HKTM/pkg/uma/shared"
)

type CharacterService struct {
	service *uma.Service
}

var _ uma.CharacterServiceInterface = (*CharacterService)(nil)

func NewCharacterService(s *uma.Service) uma.CharacterServiceInterface {
	return &CharacterService{service: s}
}

// SearchCharacter searches for a character by name, checking database first then API
func (cs *CharacterService) SearchCharacter(query string) (*shared.CharacterSearchResult, error) {
	fmt.Printf("Searching for character: %s\n", query)

	// First, try to find in database
	dbCharacter, err := cs.service.CharacterRepo.GetCharacterByName(query)
	if err != nil {
		fmt.Printf("Database query error: %v\n", err)
	} else if dbCharacter != nil {
		fmt.Printf("Found character in database: %s (ID: %d)\n", dbCharacter.NameEn, dbCharacter.CharacterID)
		// Convert database model to API model
		character := cs.convertDBCharacterToShared(dbCharacter)

		return &shared.CharacterSearchResult{
			Found:     true,
			Character: character,
			Query:     query,
		}, nil
	} else {
		fmt.Printf("Character not found in database\n")
	}

	// If not found in database, search API
	fmt.Printf("Searching API for character: %s\n", query)
	result := cs.service.UmapyoiClient.SearchCharacter(query)

	// If found in API, save to database
	if result.Found && result.Character != nil {
		fmt.Printf("Found character in API, saving to database: %s (ID: %d)\n", result.Character.NameEn, result.Character.ID)
		if err := cs.saveCharacterToDatabase(result.Character); err != nil {
			// Log error but don't fail the search
			fmt.Printf("Failed to save character to database: %v\n", err)
		} else {
			fmt.Printf("Successfully saved character to database\n")
		}
	} else {
		fmt.Printf("Character not found in API\n")
	}

	return result, nil
}

// GetCharacterImages gets character images, checking database first then API
func (cs *CharacterService) GetCharacterImages(charaID int) (*shared.CharacterImagesResult, error) {
	// First, try to find in database
	dbImages, err := cs.service.CharacterRepo.GetCharacterImagesByCharacterID(charaID)
	if err == nil && len(dbImages) > 0 {
		// Convert database models to API models
		imageCategories := cs.convertDBImagesToShared(dbImages)

		return &shared.CharacterImagesResult{
			Found:   true,
			Images:  imageCategories,
			CharaID: charaID,
		}, nil
	}

	// If not found in database, search API
	result := cs.service.UmapyoiClient.GetCharacterImages(charaID)

	// If found in API, save to database
	if result.Found && len(result.Images) > 0 {
		if err := cs.saveCharacterImagesToDatabase(charaID, result.Images); err != nil {
			// Log error but don't fail the search
			fmt.Printf("Failed to save character images to database: %v\n", err)
		}
	}

	return result, nil
}

// convertDBCharacterToShared converts database character model to shared model
func (cs *CharacterService) convertDBCharacterToShared(dbCharacter *models.Character) *shared.Character {
	return &shared.Character{
		ID:              dbCharacter.CharacterID, // Use the API data ID
		NameEn:          dbCharacter.NameEn,
		NameJp:          dbCharacter.NameJp,
		NameEnInternal:  dbCharacter.NameEnInternal,
		CategoryLabel:   dbCharacter.CategoryLabel,
		CategoryLabelEn: dbCharacter.CategoryLabelEn,
		CategoryValue:   dbCharacter.CategoryValue,
		ColorMain:       dbCharacter.ColorMain,
		ColorSub:        dbCharacter.ColorSub,
		PreferredURL:    dbCharacter.PreferredURL,
		RowNumber:       dbCharacter.RowNumber,
		ThumbImg:        dbCharacter.ThumbImg,
	}
}

// convertDBImagesToShared converts database image models to shared models
func (cs *CharacterService) convertDBImagesToShared(dbImages []models.CharacterImage) []shared.CharacterImageCategory {
	var imageCategories []shared.CharacterImageCategory
	categoryMap := make(map[string]*shared.CharacterImageCategory)

	for _, img := range dbImages {
		categoryKey := img.Category
		if category, exists := categoryMap[categoryKey]; exists {
			category.Images = append(category.Images, shared.CharacterImage{
				Image:    img.ImageURL,
				Uploaded: img.Uploaded,
			})
		} else {
			categoryMap[categoryKey] = &shared.CharacterImageCategory{
				Images: []shared.CharacterImage{{
					Image:    img.ImageURL,
					Uploaded: img.Uploaded,
				}},
				Label:   img.Category,
				LabelEn: img.CategoryEn,
			}
		}
	}

	for _, category := range categoryMap {
		imageCategories = append(imageCategories, *category)
	}

	return imageCategories
}

// saveCharacterToDatabase saves a character from API to database
func (cs *CharacterService) saveCharacterToDatabase(character *shared.Character) error {
	dbCharacter := &models.Character{
		CharacterID:     character.ID, // Store the API data ID
		NameEn:          character.NameEn,
		NameJp:          character.NameJp,
		NameEnInternal:  character.NameEnInternal,
		CategoryLabel:   character.CategoryLabel,
		CategoryLabelEn: character.CategoryLabelEn,
		CategoryValue:   character.CategoryValue,
		ColorMain:       character.ColorMain,
		ColorSub:        character.ColorSub,
		PreferredURL:    character.PreferredURL,
		RowNumber:       character.RowNumber,
		ThumbImg:        character.ThumbImg,
	}

	return cs.service.CharacterRepo.CreateCharacter(dbCharacter)
}

// saveCharacterImagesToDatabase saves character images from API to database
func (cs *CharacterService) saveCharacterImagesToDatabase(charaID int, imageCategories []shared.CharacterImageCategory) error {
	// First, find the character in database to get the database ID
	dbCharacter, err := cs.service.CharacterRepo.GetCharacterByCharacterID(charaID)
	if err != nil {
		return fmt.Errorf("failed to find character for images: %v", err)
	}

	for _, category := range imageCategories {
		for _, image := range category.Images {
			dbImage := &models.CharacterImage{
				CharacterID:   charaID,        // API data ID
				CharacterDBID: dbCharacter.ID, // Database foreign key
				ImageURL:      image.Image,
				Uploaded:      image.Uploaded,
				Category:      category.Label,
				CategoryEn:    category.LabelEn,
			}

			if err := cs.service.CharacterRepo.CreateCharacterImage(dbImage); err != nil {
				return err
			}
		}
	}
	return nil
}
