package repository

import (
	"github.com/google/uuid"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"gorm.io/gorm"
)

// CharacterRepository handles database operations for Character model
type CharacterRepository struct {
	db *gorm.DB
}

func NewCharacterRepository(db *gorm.DB) *CharacterRepository {
	return &CharacterRepository{db: db}
}

func (r *CharacterRepository) GetAllCharacters() ([]models.Character, error) {
	var characters []models.Character
	if err := r.db.Find(&characters).Error; err != nil {
		return nil, err
	}
	return characters, nil
}

func (r *CharacterRepository) GetCharacterByID(id uuid.UUID) (*models.Character, error) {
	var character models.Character
	if err := r.db.First(&character, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &character, nil
}

func (r *CharacterRepository) GetCharacterByName(name string) (*models.Character, error) {
	var character models.Character
	if err := r.db.Where("name_en ILIKE ? OR name_jp ILIKE ?", "%"+name+"%", "%"+name+"%").First(&character).Error; err != nil {
		return nil, err
	}
	return &character, nil
}

func (r *CharacterRepository) GetCharacterImagesByCharacterID(charaID int) ([]models.CharacterImage, error) {
	var images []models.CharacterImage
	if err := r.db.Where("character_id = ?", charaID).Find(&images).Error; err != nil {
		return nil, err
	}
	return images, nil
}

func (r *CharacterRepository) CreateCharacter(character *models.Character) error {
	return r.db.Create(character).Error
}

func (r *CharacterRepository) CreateCharacterImage(image *models.CharacterImage) error {
	return r.db.Create(image).Error
}
