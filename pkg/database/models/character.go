package models

import (
	"time"

	"github.com/google/uuid"
)

// Character represents a Uma Musume character in the database
type Character struct {
	ID              uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	NameEn          string    `gorm:"index;not null"`
	NameJp          string    `gorm:"index"`
	NameEnInternal  string    `gorm:"index"`
	CategoryLabel   string    `gorm:"index"`
	CategoryLabelEn string    `gorm:"index"`
	CategoryValue   string    `gorm:"index"`
	ColorMain       string    `gorm:"index"`
	ColorSub        string    `gorm:"index"`
	PreferredURL    string    `gorm:"index"`
	RowNumber       int       `gorm:"index"`
	ThumbImg        string    `gorm:"index"`
	CreatedAt       time.Time `gorm:"default:now()"`
	UpdatedAt       time.Time `gorm:"default:now()"`
	DeletedAt       time.Time `gorm:"index"`

	// Relationships
	Images []CharacterImage `gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE"`
}

// CharacterImage represents a character image in the database
type CharacterImage struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	CharacterID int       `gorm:"index;not null"`
	ImageURL    string    `gorm:"not null"`
	Uploaded    string    `gorm:"index"`
	Category    string    `gorm:"index"`
	CategoryEn  string    `gorm:"index"`
	CreatedAt   time.Time `gorm:"default:now()"`

	// Relationships
	Character Character `gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE"`
}
