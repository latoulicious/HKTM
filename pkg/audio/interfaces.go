package audio

import (
	"io"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/database/models"
)

// AudioPipeline is the main interface for the audio pipeline controller
type AudioPipeline interface {
	// Core playback operations
	PlayURL(url string, voiceConn *discordgo.VoiceConnection) error
	Stop() error
	IsPlaying() bool
	GetStatus() PipelineStatus

	// Lifecycle management
	Initialize() error
	Shutdown() error
	IsInitialized() bool
}

// StreamProcessor handles the FFmpeg process and audio stream generation
type StreamProcessor interface {
	StartStream(url string) (io.ReadCloser, error)
	Stop() error
	IsRunning() bool
	IsProcessAlive() bool
	Restart(url string) error
	WaitForExit(timeout time.Duration) error
	GetProcessInfo() map[string]interface{}

	// URL refresh detection methods (Requirement 8.1, 8.2)
	DetectStreamFailure(err error) bool
	HandleStreamFailureWithRefresh(originalURL string) error
}

// AudioEncoder handles Opus encoding for Discord compatibility
type AudioEncoder interface {
	Initialize() error
	Encode(pcmData []int16) ([]byte, error)
	Close() error
	IsInitialized() bool
	EncodeFrame(pcmFrame []int16) ([]byte, error)
	GetFrameSize() int
	GetFrameDuration() time.Duration
	ValidateFrameSize(pcmData []int16) error
	PrepareForStreaming() error
}

// ErrorHandler manages error handling and retry logic
type ErrorHandler interface {
	HandleError(err error, context string) (shouldRetry bool, delay time.Duration)
	LogError(err error, context string)
	IsRetryableError(err error) bool
	GetRetryDelay(attempt int) time.Duration
	GetRetryDelayForError(err error, attempt int) time.Duration
	GetMaxRetries() int
	ShouldRetryAfterAttempts(attempts int, err error) bool

	// User notification methods
	SetNotifier(notifier UserNotifier, channelID string)
	DisableNotifications()
	NotifyRetryAttempt(attempt int, err error, delay time.Duration)
	NotifyMaxRetriesExceeded(finalErr error, attempts int)
}

// UserNotifier defines the interface for sending notifications to Discord users
type UserNotifier interface {
	NotifyError(channelID string, errorType string, message string) error
	NotifyRetry(channelID string, attempt int, maxAttempts int, nextDelay time.Duration) error
	NotifyFatalError(channelID string, errorType string, message string) error
}

// AudioLogger provides centralized logging with database persistence
type AudioLogger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Debug(msg string, fields map[string]interface{})

	// Pipeline context methods for centralized logging integration
	WithPipeline(pipeline string) AudioLogger
	WithContext(ctx map[string]interface{}) AudioLogger
}

// ConfigProvider manages configuration loading from multiple sources
type ConfigProvider interface {
	GetPipelineConfig() *PipelineConfig
	GetFFmpegConfig() *FFmpegConfig
	GetYtDlpConfig() *YtDlpConfig
	GetOpusConfig() *OpusConfig
	GetStreamingConfig() *StreamingConfig
	GetRetryConfig() *RetryConfig
	GetLoggerConfig() *LoggerConfig
	Validate() error
	ValidateDependencies() error
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
