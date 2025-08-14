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
	// First, try to find in database
	dbCharacter, err := cs.service.CharacterRepo.GetCharacterByName(query)
	if err == nil && dbCharacter != nil {
		// Convert database model to API model
		character := &shared.Character{
			ID:              int(dbCharacter.ID.String()[0]), // Simplified conversion
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

		return &shared.CharacterSearchResult{
			Found:     true,
			Character: character,
			Query:     query,
		}, nil
	}

	// If not found in database, search API
	result := cs.service.UmapyoiClient.SearchCharacter(query)

	// If found in API, save to database
	if result.Found && result.Character != nil {
		if err := cs.saveCharacterToDatabase(result.Character); err != nil {
			// Log error but don't fail the search
			fmt.Printf("Failed to save character to database: %v\n", err)
		}
	}

	return result, nil
}

// GetCharacterImages gets character images, checking database first then API
func (cs *CharacterService) GetCharacterImages(charaID int) (*shared.CharacterImagesResult, error) {
	// First, try to find in database
	dbImages, err := cs.service.CharacterRepo.GetCharacterImagesByCharacterID(charaID)
	if err == nil && len(dbImages) > 0 {
		// Convert database models to API models
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

// saveCharacterToDatabase saves a character from API to database
func (cs *CharacterService) saveCharacterToDatabase(character *shared.Character) error {
	dbCharacter := &models.Character{
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
	for _, category := range imageCategories {
		for _, image := range category.Images {
			dbImage := &models.CharacterImage{
				CharacterID: charaID,
				ImageURL:    image.Image,
				Uploaded:    image.Uploaded,
				Category:    category.Label,
				CategoryEn:  category.LabelEn,
			}

			if err := cs.service.CharacterRepo.CreateCharacterImage(dbImage); err != nil {
				return err
			}
		}
	}
	return nil
}
