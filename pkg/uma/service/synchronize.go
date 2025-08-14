package service

import (
	"fmt"

	"github.com/latoulicious/HKTM/pkg/uma"
)

type SyncService struct {
	service *uma.Service
}

var _ uma.SyncServiceInterface = (*SyncService)(nil)

func NewSyncService(s *uma.Service) uma.SyncServiceInterface {
	return &SyncService{service: s}
}

// SyncAllCharacters syncs all characters from API to database
func (ss *SyncService) SyncAllCharacters() error {
	// Get all characters from API using the client's method
	result := ss.service.UmapyoiClient.GetAllCharacters()
	if result.Error != nil {
		return fmt.Errorf("failed to fetch character data: %v", result.Error)
	}

	if !result.Found {
		return fmt.Errorf("no characters found in API response")
	}

	// Save all characters to database
	characterService := NewCharacterService(ss.service)
	for _, character := range result.Characters {
		if err := characterService.(*CharacterService).saveCharacterToDatabase(&character); err != nil {
			// Log error but continue with other characters
			fmt.Printf("Failed to save character %s: %v\n", character.NameEn, err)
		}
	}

	return nil
}

// SyncAllSupportCards syncs all support cards from API to database
func (ss *SyncService) SyncAllSupportCards() error {
	// Get all support cards from API
	result := ss.service.UmapyoiClient.GetSupportCardList()
	if !result.Found {
		return fmt.Errorf("failed to get support card list from API")
	}

	// Save all support cards to database
	supportCardService := NewSupportCardService(ss.service)
	if err := supportCardService.(*SupportCardService).saveSupportCardsToDatabase(result.SupportCards); err != nil {
		return err
	}

	return nil
}
