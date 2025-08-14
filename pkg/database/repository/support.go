package repository

import (
	"github.com/google/uuid"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"gorm.io/gorm"
)

type SupportCardRepository struct {
	db *gorm.DB
}

func NewSupportCardRepository(db *gorm.DB) *SupportCardRepository {
	return &SupportCardRepository{db: db}
}

func (r *SupportCardRepository) GetAllSupportCards() ([]models.SupportCard, error) {
	var supportCards []models.SupportCard
	if err := r.db.Find(&supportCards).Error; err != nil {
		return nil, err
	}
	return supportCards, nil
}

func (r *SupportCardRepository) GetSupportCardByID(id uuid.UUID) (*models.SupportCard, error) {
	var supportCard models.SupportCard
	if err := r.db.First(&supportCard, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &supportCard, nil
}

func (r *SupportCardRepository) CreateSupportCard(supportCard *models.SupportCard) error {
	return r.db.Create(supportCard).Error
}
