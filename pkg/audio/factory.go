package audio

import (
	"gorm.io/gorm"
)

// NewAudioPipelineWithDependencies creates a complete audio pipeline with all dependencies
// This is a placeholder factory function that will be implemented in later tasks
func NewAudioPipelineWithDependencies(db *gorm.DB, guildID string) (AudioPipeline, error) {
	// This will be implemented in task 12 - Create dependency injection factory
	// For now, return nil to satisfy the interface
	return nil, nil
}

// Default configuration values
var (
	DefaultPipelineConfig = &PipelineConfig{
		RetryCount:     3,
		TimeoutSeconds: 30,
		LogLevel:       "info",
		FFmpegOptions:  []string{"-reconnect", "1", "-reconnect_delay_max", "5"},
	}

	DefaultFFmpegConfig = &FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{"-reconnect", "1", "-reconnect_delay_max", "5"},
	}

	DefaultOpusConfig = &OpusConfig{
		Bitrate:   128000,
		FrameSize: 960,
	}

	DefaultRetryConfig = &RetryConfig{
		MaxRetries: 3,
		BaseDelay:  2000000000,  // 2 seconds in nanoseconds
		MaxDelay:   30000000000, // 30 seconds in nanoseconds
		Multiplier: 2.0,
	}

	DefaultLoggerConfig = &LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: true,
	}
)
