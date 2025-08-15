package audio

import (
	"io"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/database/models"
)

// AudioPipeline is the main interface for the audio pipeline controller
type AudioPipeline interface {
	PlayURL(url string, voiceConn *discordgo.VoiceConnection) error
	Stop() error
	IsPlaying() bool
	GetStatus() PipelineStatus
}

// StreamProcessor handles the FFmpeg process and audio stream generation
type StreamProcessor interface {
	StartStream(url string) (io.ReadCloser, error)
	Stop() error
	IsRunning() bool
	Restart(url string) error
}

// AudioEncoder handles Opus encoding for Discord compatibility
type AudioEncoder interface {
	Initialize() error
	Encode(pcmData []int16) ([]byte, error)
	Close() error
}

// ErrorHandler manages error handling and retry logic
type ErrorHandler interface {
	HandleError(err error, context string) (shouldRetry bool, delay time.Duration)
	LogError(err error, context string)
	IsRetryableError(err error) bool
}

// AudioLogger provides centralized logging with database persistence
type AudioLogger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})
}

// ConfigProvider manages configuration loading from multiple sources
type ConfigProvider interface {
	GetPipelineConfig() *PipelineConfig
	GetFFmpegConfig() *FFmpegConfig
	GetOpusConfig() *OpusConfig
	GetRetryConfig() *RetryConfig
	GetLoggerConfig() *LoggerConfig
	Validate() error
}

// MetricsCollector handles performance metrics collection
type MetricsCollector interface {
	RecordStartupTime(duration time.Duration)
	RecordError(errorType string)
	RecordPlaybackDuration(duration time.Duration)
	GetStats() MetricsStats
}

// AudioRepository handles database operations for audio-related data
type AudioRepository interface {
	SaveError(error *models.AudioError) error
	SaveMetric(metric *models.AudioMetric) error
	SaveLog(log *models.AudioLog) error
	GetErrorStats(guildID string) (*ErrorStats, error)
	GetMetricsStats(guildID string) (*MetricsStats, error)
}
