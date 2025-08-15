package audio

import (
	"time"

	"github.com/latoulicious/HKTM/pkg/database/models"
)

// PipelineConfig contains general pipeline configuration
type PipelineConfig struct {
	RetryCount     int      `yaml:"retry_count" env:"AUDIO_RETRY_COUNT"`
	TimeoutSeconds int      `yaml:"timeout_seconds" env:"AUDIO_TIMEOUT"`
	FFmpegOptions  []string `yaml:"ffmpeg_options" env:"AUDIO_FFMPEG_OPTIONS"`
	LogLevel       string   `yaml:"log_level" env:"AUDIO_LOG_LEVEL"`
}

// FFmpegConfig contains FFmpeg-specific configuration
type FFmpegConfig struct {
	BinaryPath  string   `yaml:"binary_path" env:"AUDIO_FFMPEG_BINARY"`
	AudioFormat string   `yaml:"audio_format" env:"AUDIO_FFMPEG_FORMAT"`
	SampleRate  int      `yaml:"sample_rate" env:"AUDIO_FFMPEG_SAMPLE_RATE"`
	Channels    int      `yaml:"channels" env:"AUDIO_FFMPEG_CHANNELS"`
	CustomArgs  []string `yaml:"custom_args" env:"AUDIO_FFMPEG_CUSTOM_ARGS"`
}

// OpusConfig contains Opus encoder configuration
type OpusConfig struct {
	Bitrate   int `yaml:"bitrate" env:"AUDIO_OPUS_BITRATE"`
	FrameSize int `yaml:"frame_size" env:"AUDIO_OPUS_FRAME_SIZE"`
}

// RetryConfig contains retry logic configuration
type RetryConfig struct {
	MaxRetries int           `yaml:"max_retries" env:"AUDIO_MAX_RETRIES"`
	BaseDelay  time.Duration `yaml:"base_delay" env:"AUDIO_BASE_DELAY"`
	MaxDelay   time.Duration `yaml:"max_delay" env:"AUDIO_MAX_DELAY"`
	Multiplier float64       `yaml:"multiplier" env:"AUDIO_RETRY_MULTIPLIER"`
}

// LoggerConfig contains logging configuration
type LoggerConfig struct {
	Level    string `yaml:"level" env:"AUDIO_LOG_LEVEL"`
	Format   string `yaml:"format" env:"AUDIO_LOG_FORMAT"`
	SaveToDB bool   `yaml:"save_to_db" env:"AUDIO_LOG_SAVE_DB"`
}

// PipelineStatus represents the current status of the audio pipeline
type PipelineStatus struct {
	IsPlaying  bool      `json:"is_playing"`
	CurrentURL string    `json:"current_url"`
	StartTime  time.Time `json:"start_time"`
	ErrorCount int       `json:"error_count"`
	LastError  string    `json:"last_error"`
}

// MetricsStats contains aggregated metrics data
type MetricsStats struct {
	TotalPlaybackTime  time.Duration `json:"total_playback_time"`
	AverageStartupTime time.Duration `json:"average_startup_time"`
	ErrorCount         int           `json:"error_count"`
	SuccessfulPlays    int           `json:"successful_plays"`
}

// ErrorStats contains aggregated error statistics
type ErrorStats struct {
	TotalErrors   int                 `json:"total_errors"`
	ErrorsByType  map[string]int      `json:"errors_by_type"`
	RecentErrors  []models.AudioError `json:"recent_errors"`
	LastErrorTime time.Time           `json:"last_error_time"`
}
