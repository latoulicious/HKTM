package service

import (
	"fmt"

	"github.com/latoulicious/HKTM/pkg/database/models"
	"github.com/latoulicious/HKTM/pkg/logging"
	"github.com/latoulicious/HKTM/pkg/uma"
	"github.com/latoulicious/HKTM/pkg/uma/shared"
)

type CharacterService struct {
	service *uma.Service
	logger  logging.Logger
}

var _ uma.CharacterServiceInterface = (*CharacterService)(nil)

func NewCharacterService(s *uma.Service) uma.CharacterServiceInterface {
	return &CharacterService{
		service: s,
		logger:  logging.GetGlobalLoggerFactory().CreateLogger("uma_service"),
	}
}

// SearchCharacter searches for a character by name, checking database first then API
func (cs *CharacterService) SearchCharacter(query string) (*shared.CharacterSearchResult, error) {
	cs.logger.Info("Starting character search", map[string]interface{}{
		"query": query,
		"stage": "database_lookup",
	})

	// First, try to find in database
	dbCharacter, err := cs.service.CharacterRepo.GetCharacterByName(query)
	if err != nil {
		cs.logger.Error("Database query error during character search", err, map[string]interface{}{
			"query": query,
			"stage": "database_lookup",
		})
	} else if dbCharacter != nil {
		cs.logger.Info("Found character in database", map[string]interface{}{
			"query":          query,
			"character_name": dbCharacter.NameEn,
			"character_id":   dbCharacter.CharacterID,
			"stage":          "database_found",
		})
		// Convert database model to API model
		character := cs.convertDBCharacterToShared(dbCharacter)

		return &shared.CharacterSearchResult{
			Found:     true,
			Character: character,
			Query:     query,
		}, nil
	} else {
		cs.logger.Info("Character not found in database", map[string]interface{}{
			"query": query,
			"stage": "database_not_found",
		})
	}

	// If not found in database, search API
	cs.logger.Info("Searching API for character", map[string]interface{}{
		"query": query,
		"stage": "api_lookup",
	})
	result := cs.service.UmapyoiClient.SearchCharacter(query)

	// If found in API, save to database
	if result.Found && result.Character != nil {
		cs.logger.Info("Found character in API, saving to database", map[string]interface{}{
			"query":          query,
			"character_name": result.Character.NameEn,
			"character_id":   result.Character.ID,
			"stage":          "api_found_saving",
		})
		if err := cs.saveCharacterToDatabase(result.Character); err != nil {
			// Log error but don't fail the search
			cs.logger.Error("Failed to save character to database", err, map[string]interface{}{
				"query":          query,
				"character_name": result.Character.NameEn,
				"character_id":   result.Character.ID,
				"stage":          "database_save_failed",
			})
		} else {
			cs.logger.Info("Successfully saved character to database", map[string]interface{}{
				"query":          query,
				"character_name": result.Character.NameEn,
				"character_id":   result.Character.ID,
				"stage":          "database_save_success",
			})
		}
	} else {
		cs.logger.Info("Character not found in API", map[string]interface{}{
			"query": query,
			"stage": "api_not_found",
		})
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
			cs.logger.Error("Failed to save character images to database", err, map[string]interface{}{
				"character_id": charaID,
				"images_count": len(result.Images),
				"stage":        "image_save_failed",
			})
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
