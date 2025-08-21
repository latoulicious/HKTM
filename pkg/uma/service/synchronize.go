package service

import (
	"fmt"

	"github.com/latoulicious/HKTM/pkg/logging"
	"github.com/latoulicious/HKTM/pkg/uma"
)

type SyncService struct {
	service *uma.Service
	logger  logging.Logger
}

var _ uma.SyncServiceInterface = (*SyncService)(nil)

func NewSyncService(s *uma.Service) uma.SyncServiceInterface {
	return &SyncService{
		service: s,
		logger:  logging.GetGlobalLoggerFactory().CreateLogger("uma_sync"),
	}
}

// SyncAllCharacters syncs all characters from API to database
func (ss *SyncService) SyncAllCharacters() error {
	ss.logger.Info("Starting character synchronization", map[string]interface{}{
		"sync_type": "characters",
	})

	// Get all characters from API using the client's method
	result := ss.service.UmapyoiClient.GetAllCharacters()
	if result.Error != nil {
		ss.logger.Error("Failed to fetch character data from API", result.Error, map[string]interface{}{
			"sync_type": "characters",
			"stage":     "fetch_all",
		})
		return fmt.Errorf("failed to fetch character data: %v", result.Error)
	}

	if !result.Found {
		ss.logger.Error("No characters found in API response", nil, map[string]interface{}{
			"sync_type": "characters",
			"stage":     "fetch_all",
		})
		return fmt.Errorf("no characters found in API response")
	}

	ss.logger.Info("Fetched characters from API, starting database sync", map[string]interface{}{
		"sync_type":        "characters",
		"characters_count": len(result.Characters),
		"stage":            "database_sync",
	})

	// Save all characters to database
	characterService := NewCharacterService(ss.service)
	successCount := 0
	errorCount := 0

	for _, character := range result.Characters {
		if err := characterService.(*CharacterService).saveCharacterToDatabase(&character); err != nil {
			// Log error but continue with other characters
			ss.logger.Error("Failed to save character to database", err, map[string]interface{}{
				"sync_type":      "characters",
				"character_name": character.NameEn,
				"character_id":   character.ID,
				"stage":          "individual_save",
			})
			errorCount++
		} else {
			successCount++
		}
	}

	ss.logger.Info("Character synchronization completed", map[string]interface{}{
		"sync_type":        "characters",
		"total_characters": len(result.Characters),
		"success_count":    successCount,
		"error_count":      errorCount,
		"stage":            "completed",
	})

	return nil
}

// SyncAllSupportCards syncs all support cards from API to database
func (ss *SyncService) SyncAllSupportCards() error {
	ss.logger.Info("Starting support card synchronization", map[string]interface{}{
		"sync_type": "support_cards",
	})

	// Get all support cards from API
	result := ss.service.UmapyoiClient.GetSupportCardList()
	if !result.Found {
		ss.logger.Error("Failed to get support card list from API", result.Error, map[string]interface{}{
			"sync_type": "support_cards",
			"stage":     "fetch_all",
		})
		return fmt.Errorf("failed to get support card list from API")
	}

	ss.logger.Info("Fetched support cards from API, starting database sync", map[string]interface{}{
		"sync_type":   "support_cards",
		"cards_count": len(result.SupportCards),
		"stage":       "database_sync",
	})

	// Save all support cards to database
	supportCardService := NewSupportCardService(ss.service)
	if err := supportCardService.(*SupportCardService).saveSupportCardsToDatabase(result.SupportCards); err != nil {
		ss.logger.Error("Failed to save support cards to database", err, map[string]interface{}{
			"sync_type":   "support_cards",
			"cards_count": len(result.SupportCards),
			"stage":       "database_save_failed",
		})
		return err
	}

	ss.logger.Info("Support card synchronization completed", map[string]interface{}{
		"sync_type":   "support_cards",
		"cards_count": len(result.SupportCards),
		"stage":       "completed",
	})

	return nil
}
