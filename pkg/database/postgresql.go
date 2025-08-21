package database

import (
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewGormDBFromConfig creates a new GORM database connection from config
func NewGormDBFromConfig(databaseURL string) (*gorm.DB, error) {
	return NewGormDB(databaseURL)
}

// NewGormDB creates a new GORM database connection using the provided DSN
func NewGormDB(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, errors.New("database DSN is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
