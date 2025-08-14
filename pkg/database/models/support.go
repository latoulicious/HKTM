package models

import (
	"time"

	"github.com/google/uuid"
)

// SupportCard represents a support card in the database
type SupportCard struct {
	ID            uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4()"`
	SupportID     int        `gorm:"index;not null"` // API data ID
	CharaID       int        `gorm:"index;not null"` // API character ID
	CharacterDBID *uuid.UUID `gorm:"type:uuid"`      // Database foreign key (optional, nullable)
	Gametora      string     `gorm:"index"`
	TitleEn       string     `gorm:"index;not null"`
	Title         string     `gorm:"index;not null"`
	Rarity        int        `gorm:"index"`
	RarityString  string     `gorm:"index"`
	StartDate     int64      `gorm:"index"`
	Type          string     `gorm:"index"`
	TypeIconURL   string     `gorm:"index"`
	CreatedAt     time.Time  `gorm:"default:now()"`
	UpdatedAt     time.Time  `gorm:"default:now()"`
	DeletedAt     time.Time  `gorm:"index"`

	// Relationships
	// Note: CharacterDBID is optional, so we don't enforce foreign key constraint
	// Character Character `gorm:"foreignKey:CharacterDBID;constraint:OnDelete:CASCADE"`
}
