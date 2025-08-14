package models

import (
	"time"

	"github.com/google/uuid"
)

// SupportCard represents a support card in the database
type SupportCard struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	CharaID      int       `gorm:"index;not null"`
	Gametora     string    `gorm:"index"`
	TitleEn      string    `gorm:"index;not null"`
	Title        string    `gorm:"index;not null"`
	Rarity       int       `gorm:"index"`
	RarityString string    `gorm:"index"`
	StartDate    int64     `gorm:"index"`
	Type         string    `gorm:"index"`
	TypeIconURL  string    `gorm:"index"`
	CreatedAt    time.Time `gorm:"default:now()"`
	UpdatedAt    time.Time `gorm:"default:now()"`
	DeletedAt    time.Time `gorm:"index"`

	// Relationships
	Character Character `gorm:"foreignKey:CharaID;constraint:OnDelete:CASCADE"`
}
