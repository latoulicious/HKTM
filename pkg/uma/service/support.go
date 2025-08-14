package service

import (
	"fmt"

	"github.com/latoulicious/HKTM/pkg/database/models"
	"github.com/latoulicious/HKTM/pkg/uma"
	"github.com/latoulicious/HKTM/pkg/uma/shared"
)

type SupportCardService struct {
	service *uma.Service
}

var _ uma.SupportCardServiceInterface = (*SupportCardService)(nil)

func NewSupportCardService(s *uma.Service) uma.SupportCardServiceInterface {
	return &SupportCardService{service: s}
}

// SearchSupportCard searches for a support card by name, checking database first then API
func (scs *SupportCardService) SearchSupportCard(query string) (*shared.SupportCardSearchResult, error) {
	// First, try to find in database
	dbSupportCards, err := scs.service.SupportCardRepo.GetSupportCardsByTitle(query)
	if err == nil && len(dbSupportCards) > 0 {
		// Convert database models to API models
		var supportCards []shared.SupportCard
		for _, dbCard := range dbSupportCards {
			supportCard := shared.SupportCard{
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
			supportCards = append(supportCards, supportCard)
		}

		return &shared.SupportCardSearchResult{
			Found:        true,
			SupportCards: supportCards,
			Query:        query,
		}, nil
	}

	// If not found in database, search API
	result := scs.service.UmapyoiClient.SearchSupportCard(query)

	// If found in API, save to database
	if result.Found && len(result.SupportCards) > 0 {
		if err := scs.saveSupportCardsToDatabase(result.SupportCards); err != nil {
			// Log error but don't fail the search
			fmt.Printf("Failed to save support cards to database: %v\n", err)
		}
	}

	return result, nil
}

// GetSupportCardList gets all support cards, checking database first then API
func (scs *SupportCardService) GetSupportCardList() (*shared.SupportCardListResult, error) {
	// First, try to find in database
	dbSupportCards, err := scs.service.SupportCardRepo.GetAllSupportCards()
	if err == nil && len(dbSupportCards) > 0 {
		// Convert database models to API models
		var supportCards []shared.SupportCard
		for _, dbCard := range dbSupportCards {
			supportCard := shared.SupportCard{
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
			supportCards = append(supportCards, supportCard)
		}

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
			fmt.Printf("Failed to save support cards to database: %v\n", err)
		}
	}

	return result, nil
}

// saveSupportCardsToDatabase saves support cards from API to database
func (scs *SupportCardService) saveSupportCardsToDatabase(supportCards []shared.SupportCard) error {
	for _, card := range supportCards {
		dbCard := &models.SupportCard{
			CharaID:      card.CharaID,
			Gametora:     card.Gametora,
			TitleEn:      card.TitleEn,
			Title:        card.Title,
			Rarity:       card.Rarity,
			RarityString: card.RarityString,
			StartDate:    card.StartDate,
			Type:         card.Type,
			TypeIconURL:  card.TypeIconURL,
		}

		if err := scs.service.SupportCardRepo.CreateSupportCard(dbCard); err != nil {
			return err
		}
	}
	return nil
}
