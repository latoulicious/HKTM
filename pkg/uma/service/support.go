package service

import (
	"github.com/google/uuid"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"github.com/latoulicious/HKTM/pkg/logging"
	"github.com/latoulicious/HKTM/pkg/uma"
	"github.com/latoulicious/HKTM/pkg/uma/shared"
)

type SupportCardService struct {
	service *uma.Service
	logger  logging.Logger
}

var _ uma.SupportCardServiceInterface = (*SupportCardService)(nil)

func NewSupportCardService(s *uma.Service) uma.SupportCardServiceInterface {
	return &SupportCardService{
		service: s,
		logger:  logging.GetGlobalLoggerFactory().CreateLogger("uma_service"),
	}
}

// SearchSupportCard searches for a support card by name, checking database first then API
func (scs *SupportCardService) SearchSupportCard(query string) (*shared.SupportCardSearchResult, error) {
	scs.logger.Info("Starting support card search", map[string]interface{}{
		"query": query,
		"stage": "database_lookup",
	})

	// First, try to find in database
	dbSupportCards, err := scs.service.SupportCardRepo.GetSupportCardsByTitle(query)
	if err != nil {
		scs.logger.Error("Database query error during support card search", err, map[string]interface{}{
			"query": query,
			"stage": "database_lookup",
		})
	} else if len(dbSupportCards) > 0 {
		scs.logger.Info("Found support cards in database", map[string]interface{}{
			"query":       query,
			"cards_count": len(dbSupportCards),
			"stage":       "database_found",
		})
		// Convert database models to API models
		supportCards := scs.convertDBSupportCardsToShared(dbSupportCards)

		return &shared.SupportCardSearchResult{
			Found:        true,
			SupportCards: supportCards,
			Query:        query,
		}, nil
	} else {
		scs.logger.Info("Support cards not found in database", map[string]interface{}{
			"query": query,
			"stage": "database_not_found",
		})
	}

	// If not found in database, search API
	scs.logger.Info("Searching API for support card", map[string]interface{}{
		"query": query,
		"stage": "api_lookup",
	})
	result := scs.service.UmapyoiClient.SearchSupportCard(query)

	// If found in API, save to database
	if result.Found && len(result.SupportCards) > 0 {
		scs.logger.Info("Found support cards in API, saving to database", map[string]interface{}{
			"query":       query,
			"cards_count": len(result.SupportCards),
			"stage":       "api_found_saving",
		})
		if err := scs.saveSupportCardsToDatabase(result.SupportCards); err != nil {
			// Log error but don't fail the search
			scs.logger.Error("Failed to save support cards to database", err, map[string]interface{}{
				"query":       query,
				"cards_count": len(result.SupportCards),
				"stage":       "database_save_failed",
			})
		} else {
			scs.logger.Info("Successfully saved support cards to database", map[string]interface{}{
				"query":       query,
				"cards_count": len(result.SupportCards),
				"stage":       "database_save_success",
			})
		}
	} else {
		scs.logger.Info("Support cards not found in API", map[string]interface{}{
			"query": query,
			"stage": "api_not_found",
		})
	}

	return result, nil
}

// GetSupportCardList gets all support cards, checking database first then API
func (scs *SupportCardService) GetSupportCardList() (*shared.SupportCardListResult, error) {
	// First, try to find in database
	dbSupportCards, err := scs.service.SupportCardRepo.GetAllSupportCards()
	if err == nil && len(dbSupportCards) > 0 {
		// Convert database models to API models
		supportCards := scs.convertDBSupportCardsToShared(dbSupportCards)

		return &shared.SupportCardListResult{
			Found:        true,
			SupportCards: supportCards,
		}, nil
	}

	// If not found in database, search API
	result := scs.service.UmapyoiClient.GetSupportCardList()

	// If found in API, save to database
	if result.Found && len(result.SupportCards) > 0 {
		if err := scs.saveSupportCardsToDatabase(result.SupportCards); err != nil {
			// Log error but don't fail the search
			scs.logger.Error("Failed to save support cards to database", err, map[string]interface{}{
				"cards_count": len(result.SupportCards),
				"stage":       "database_save_failed",
			})
		}
	}

	return result, nil
}

// convertDBSupportCardsToShared converts database support card models to shared models
func (scs *SupportCardService) convertDBSupportCardsToShared(dbSupportCards []models.SupportCard) []shared.SupportCard {
	var supportCards []shared.SupportCard
	for _, dbCard := range dbSupportCards {
		supportCard := shared.SupportCard{
			CharaID:      dbCard.CharaID,
			Gametora:     dbCard.Gametora,
			ID:           dbCard.SupportID, // Use the API data ID
			TitleEn:      dbCard.TitleEn,
			Title:        dbCard.Title,
			Rarity:       dbCard.Rarity,
			RarityString: dbCard.RarityString,
			StartDate:    dbCard.StartDate,
			Type:         dbCard.Type,
			TypeIconURL:  dbCard.TypeIconURL,
		}
		supportCards = append(supportCards, supportCard)
	}
	return supportCards
}

// saveSupportCardsToDatabase saves support cards from API to database
func (scs *SupportCardService) saveSupportCardsToDatabase(supportCards []shared.SupportCard) error {
	for _, card := range supportCards {
		// Try to find the character in database to get the database ID
		var characterDBID *uuid.UUID
		dbCharacter, err := scs.service.CharacterRepo.GetCharacterByCharacterID(card.CharaID)
		if err == nil && dbCharacter != nil {
			characterDBID = &dbCharacter.ID
		}

		dbCard := &models.SupportCard{
			SupportID:     card.ID, // Store the API data ID
			CharaID:       card.CharaID,
			CharacterDBID: characterDBID, // Database foreign key (optional, can be nil)
			Gametora:      card.Gametora,
			TitleEn:       card.TitleEn,
			Title:         card.Title,
			Rarity:        card.Rarity,
			RarityString:  card.RarityString,
			StartDate:     card.StartDate,
			Type:          card.Type,
			TypeIconURL:   card.TypeIconURL,
		}

		if err := scs.service.SupportCardRepo.CreateSupportCard(dbCard); err != nil {
			return err
		}
	}
	return nil
}
