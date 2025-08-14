package database

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// DatabaseManager provides the same interface as the old SQLite database
// but uses GORM/PostgreSQL internally
type DatabaseManager struct {
	db *gorm.DB
}

// NewDatabaseManager creates a new database manager with GORM
func NewDatabaseManager(gormDB *gorm.DB) *DatabaseManager {
	// Auto-migrate the cache entries table
	err := gormDB.AutoMigrate(&CacheEntry{})
	if err != nil {
		panic("Failed to migrate cache_entries table: " + err.Error())
	}

	return &DatabaseManager{
		db: gormDB,
	}
}

// Close closes the database connection
func (dm *DatabaseManager) Close() error {
	sqlDB, err := dm.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// StartCacheCleanup starts the cache cleanup goroutine
func (dm *DatabaseManager) StartCacheCleanup(interval time.Duration) {
	// TODO: Implement cache cleanup for PostgreSQL
	// This would involve deleting expired cache entries
}

// CacheCharacterSearch caches a character search result
func (dm *DatabaseManager) CacheCharacterSearch(query string, result interface{}, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	cacheEntry := &CacheEntry{
		Key:       "character_search:" + query,
		Data:      string(data),
		Type:      "character_search",
		ExpiresAt: time.Now().Add(ttl),
	}

	return dm.db.Save(cacheEntry).Error
}

// GetCachedCharacterSearch retrieves a cached character search result
func (dm *DatabaseManager) GetCachedCharacterSearch(query string) (interface{}, error) {
	var cacheEntry CacheEntry
	err := dm.db.Where("cache_key = ? AND expires_at > ?", "character_search:"+query, time.Now()).First(&cacheEntry).Error
	if err != nil {
		return nil, err
	}

	var result interface{}
	err = json.Unmarshal([]byte(cacheEntry.Data), &result)
	return result, err
}

// CacheCharacterImages caches character images
func (dm *DatabaseManager) CacheCharacterImages(characterID int, result interface{}, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	cacheEntry := &CacheEntry{
		Key:       "character_images:" + string(rune(characterID)),
		Data:      string(data),
		Type:      "character_images",
		ExpiresAt: time.Now().Add(ttl),
	}

	return dm.db.Save(cacheEntry).Error
}

// GetCachedCharacterImages retrieves cached character images
func (dm *DatabaseManager) GetCachedCharacterImages(characterID int) (interface{}, error) {
	var cacheEntry CacheEntry
	err := dm.db.Where("cache_key = ? AND expires_at > ?", "character_images:"+string(rune(characterID)), time.Now()).First(&cacheEntry).Error
	if err != nil {
		return nil, err
	}

	var result interface{}
	err = json.Unmarshal([]byte(cacheEntry.Data), &result)
	return result, err
}

// CacheSupportCardSearch caches a support card search result
func (dm *DatabaseManager) CacheSupportCardSearch(query string, result interface{}, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	cacheEntry := &CacheEntry{
		Key:       "support_search:" + query,
		Data:      string(data),
		Type:      "support_search",
		ExpiresAt: time.Now().Add(ttl),
	}

	return dm.db.Save(cacheEntry).Error
}

// GetCachedSupportCardSearch retrieves a cached support card search result
func (dm *DatabaseManager) GetCachedSupportCardSearch(query string) (interface{}, error) {
	var cacheEntry CacheEntry
	err := dm.db.Where("cache_key = ? AND expires_at > ?", "support_search:"+query, time.Now()).First(&cacheEntry).Error
	if err != nil {
		return nil, err
	}

	var result interface{}
	err = json.Unmarshal([]byte(cacheEntry.Data), &result)
	return result, err
}

// CacheGametoraSkills caches Gametora skills data
func (dm *DatabaseManager) CacheGametoraSkills(query string, result interface{}, ttl time.Duration) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	cacheEntry := &CacheEntry{
		Key:       "gametora_skills:" + query,
		Data:      string(data),
		Type:      "gametora_skills",
		ExpiresAt: time.Now().Add(ttl),
	}

	return dm.db.Save(cacheEntry).Error
}

// GetCachedGametoraSkills retrieves cached Gametora skills data
func (dm *DatabaseManager) GetCachedGametoraSkills(query string) (interface{}, error) {
	var cacheEntry CacheEntry
	err := dm.db.Where("cache_key = ? AND expires_at > ?", "gametora_skills:"+query, time.Now()).First(&cacheEntry).Error
	if err != nil {
		return nil, err
	}

	var result interface{}
	err = json.Unmarshal([]byte(cacheEntry.Data), &result)
	return result, err
}

// GetCacheStats returns cache statistics
func (dm *DatabaseManager) GetCacheStats() (map[string]interface{}, error) {
	var totalCount int64
	var expiredCount int64

	dm.db.Model(&CacheEntry{}).Count(&totalCount)
	dm.db.Model(&CacheEntry{}).Where("expires_at <= ?", time.Now()).Count(&expiredCount)

	return map[string]interface{}{
		"total_entries":   totalCount,
		"expired_entries": expiredCount,
		"active_entries":  totalCount - expiredCount,
	}, nil
}

// CacheEntry represents a cache entry in the database
type CacheEntry struct {
	ID        uint      `gorm:"primaryKey"`
	Key       string    `gorm:"uniqueIndex;not null"`
	Data      string    `gorm:"type:text;not null"`
	Type      string    `gorm:"index;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName specifies the table name for CacheEntry
func (CacheEntry) TableName() string {
	return "cache_entries"
}
