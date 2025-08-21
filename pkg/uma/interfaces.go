package uma

import (
	"github.com/latoulicious/HKTM/pkg/database/repository"
	"github.com/latoulicious/HKTM/pkg/uma/handler"
	"github.com/latoulicious/HKTM/pkg/uma/shared"
)

// Service represents the main service that holds all dependencies
type Service struct {
	CharacterRepo   *repository.CharacterRepository
	SupportCardRepo *repository.SupportCardRepository
	UmapyoiClient   *handler.Client
	GametoraClient  *handler.GametoraClient
}

// CharacterServiceInterface defines the interface for character-related operations
type CharacterServiceInterface interface {
	SearchCharacter(query string) (*shared.CharacterSearchResult, error)
	GetCharacterImages(charaID int) (*shared.CharacterImagesResult, error)
}

// SupportCardServiceInterface defines the interface for support card-related operations
type SupportCardServiceInterface interface {
	SearchSupportCard(query string) (*shared.SupportCardSearchResult, error)
	GetSupportCardList() (*shared.SupportCardListResult, error)
}

// SyncServiceInterface defines the interface for synchronization operations
type SyncServiceInterface interface {
	SyncAllCharacters() error
	SyncAllSupportCards() error
}

// UmaServiceInterface defines the main service interface that combines all sub-services
type UmaServiceInterface interface {
	CharacterServiceInterface
	SupportCardServiceInterface
	SyncServiceInterface
}

// ServiceFactoryInterface defines the interface for creating service instances
type ServiceFactoryInterface interface {
	NewCharacterService() CharacterServiceInterface
	NewSupportCardService() SupportCardServiceInterface
	NewSyncService() SyncServiceInterface
}

// NewService creates a new Service instance with all dependencies
func NewService(
	characterRepo *repository.CharacterRepository,
	supportCardRepo *repository.SupportCardRepository,
	umapyoiClient *handler.Client,
	gametoraClient *handler.GametoraClient,
) *Service {
	return &Service{
		CharacterRepo:   characterRepo,
		SupportCardRepo: supportCardRepo,
		UmapyoiClient:   umapyoiClient,
		GametoraClient:  gametoraClient,
	}
}
