package uma

import (
	"time"

	"github.com/google/uuid"
	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"github.com/latoulicious/HKTM/pkg/logging"
)

// LogRepositoryAdapter adapts AudioRepository to implement logging.LogRepository
type LogRepositoryAdapter struct {
	audioRepo audio.AudioRepository
}

// NewLogRepositoryAdapter creates a new LogRepositoryAdapter
func NewLogRepositoryAdapter(audioRepo audio.AudioRepository) logging.LogRepository {
	return &LogRepositoryAdapter{
		audioRepo: audioRepo,
	}
}

// SaveLog implements logging.LogRepository interface
func (l *LogRepositoryAdapter) SaveLog(entry logging.LogEntry) error {
	audioLog := &models.AudioLog{
		ID:        uuid.New(),
		GuildID:   entry.GuildID,
		Component: entry.Component,
		Level:     entry.Level,
		Message:   entry.Message,
		Error:     entry.Error,
		Fields:    entry.Fields,
		UserID:    entry.UserID,
		ChannelID: entry.ChannelID,
		Timestamp: time.Now(),
	}

	return l.audioRepo.SaveLog(audioLog)
}
