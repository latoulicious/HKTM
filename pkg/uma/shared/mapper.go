package shared

import (
	"github.com/latoulicious/HKTM/pkg/database/models"
)

// CharacterMapper handles conversion between database and shared types
type CharacterMapper struct{}

// NewCharacterMapper creates a new character mapper
func NewCharacterMapper() *CharacterMapper {
	return &CharacterMapper{}
}

// ToShared converts database character to shared character
func (m *CharacterMapper) ToShared(dbChar *models.Character) *Character {
	if dbChar == nil {
		return nil
	}

	return &Character{
		ID:              int(dbChar.ID.String()[0]), // Simplified conversion
		NameEn:          dbChar.NameEn,
		NameJp:          dbChar.NameJp,
		NameEnInternal:  dbChar.NameEnInternal,
		CategoryLabel:   dbChar.CategoryLabel,
		CategoryLabelEn: dbChar.CategoryLabelEn,
		CategoryValue:   dbChar.CategoryValue,
		ColorMain:       dbChar.ColorMain,
		ColorSub:        dbChar.ColorSub,
		PreferredURL:    dbChar.PreferredURL,
		RowNumber:       dbChar.RowNumber,
		ThumbImg:        dbChar.ThumbImg,
	}
}

// ToDatabase converts shared character to database character
func (m *CharacterMapper) ToDatabase(sharedChar *Character) *models.Character {
	if sharedChar == nil {
		return nil
	}

	return &models.Character{
		NameEn:          sharedChar.NameEn,
		NameJp:          sharedChar.NameJp,
		NameEnInternal:  sharedChar.NameEnInternal,
		CategoryLabel:   sharedChar.CategoryLabel,
		CategoryLabelEn: sharedChar.CategoryLabelEn,
		CategoryValue:   sharedChar.CategoryValue,
		ColorMain:       sharedChar.ColorMain,
		ColorSub:        sharedChar.ColorSub,
		PreferredURL:    sharedChar.PreferredURL,
		RowNumber:       sharedChar.RowNumber,
		ThumbImg:        sharedChar.ThumbImg,
	}
}

// CharacterImageMapper handles conversion between database and shared character image types
type CharacterImageMapper struct{}

// NewCharacterImageMapper creates a new character image mapper
func NewCharacterImageMapper() *CharacterImageMapper {
	return &CharacterImageMapper{}
}

// ToShared converts database character images to shared character image categories
func (m *CharacterImageMapper) ToShared(dbImages []models.CharacterImage) []CharacterImageCategory {
	if len(dbImages) == 0 {
		return nil
	}

	categoryMap := make(map[string]*CharacterImageCategory)

	for _, img := range dbImages {
		categoryKey := img.Category
		if category, exists := categoryMap[categoryKey]; exists {
			category.Images = append(category.Images, CharacterImage{
				Image:    img.ImageURL,
				Uploaded: img.Uploaded,
			})
		} else {
			categoryMap[categoryKey] = &CharacterImageCategory{
				Images: []CharacterImage{{
					Image:    img.ImageURL,
					Uploaded: img.Uploaded,
				}},
				Label:   img.Category,
				LabelEn: img.CategoryEn,
			}
		}
	}

	var result []CharacterImageCategory
	for _, category := range categoryMap {
		result = append(result, *category)
	}

	return result
}

// ToDatabase converts shared character image categories to database character images
func (m *CharacterImageMapper) ToDatabase(charaID int, imageCategories []CharacterImageCategory) []models.CharacterImage {
	var result []models.CharacterImage

	for _, category := range imageCategories {
		for _, image := range category.Images {
			dbImage := &models.CharacterImage{
				CharacterID: charaID,
				ImageURL:    image.Image,
				Uploaded:    image.Uploaded,
				Category:    category.Label,
				CategoryEn:  category.LabelEn,
			}
			result = append(result, *dbImage)
		}
	}

	return result
}

// SupportCardMapper handles conversion between database and shared support card types
type SupportCardMapper struct{}

// NewSupportCardMapper creates a new support card mapper
func NewSupportCardMapper() *SupportCardMapper {
	return &SupportCardMapper{}
}

// ToShared converts database support cards to shared support cards
func (m *SupportCardMapper) ToShared(dbCards []models.SupportCard) []SupportCard {
	if len(dbCards) == 0 {
		return nil
	}

	var result []SupportCard
	for _, dbCard := range dbCards {
		sharedCard := SupportCard{
			CharaID:      dbCard.CharaID,
			Gametora:     dbCard.Gametora,
			ID:           int(dbCard.ID.String()[0]), // Simplified conversion
			TitleEn:      dbCard.TitleEn,
			Title:        dbCard.Title,
			Rarity:       dbCard.Rarity,
			RarityString: dbCard.RarityString,
			StartDate:    dbCard.StartDate,
			Type:         dbCard.Type,
			TypeIconURL:  dbCard.TypeIconURL,
		}
		result = append(result, sharedCard)
	}

	return result
}

// ToDatabase converts shared support cards to database support cards
func (m *SupportCardMapper) ToDatabase(sharedCards []SupportCard) []models.SupportCard {
	if len(sharedCards) == 0 {
		return nil
	}

	var result []models.SupportCard
	for _, sharedCard := range sharedCards {
		dbCard := &models.SupportCard{
			CharaID:      sharedCard.CharaID,
			Gametora:     sharedCard.Gametora,
			TitleEn:      sharedCard.TitleEn,
			Title:        sharedCard.Title,
			Rarity:       sharedCard.Rarity,
			RarityString: sharedCard.RarityString,
			StartDate:    sharedCard.StartDate,
			Type:         sharedCard.Type,
			TypeIconURL:  sharedCard.TypeIconURL,
		}
		result = append(result, *dbCard)
	}

	return result
}
