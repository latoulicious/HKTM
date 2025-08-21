package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"gopkg.in/yaml.v3"
)

// PipelineConfig contains general pipeline configuration
type PipelineConfig struct {
	RetryCount     int      `yaml:"retry_count" toml:"retry_count" env:"AUDIO_RETRY_COUNT"`
	TimeoutSeconds int      `yaml:"timeout_seconds" toml:"timeout_seconds" env:"AUDIO_TIMEOUT"`
	FFmpegOptions  []string `yaml:"ffmpeg_options" toml:"ffmpeg_options" env:"AUDIO_FFMPEG_OPTIONS"`
	LogLevel       string   `yaml:"log_level" toml:"log_level" env:"AUDIO_LOG_LEVEL"`
}

// FFmpegConfig contains FFmpeg-specific configuration
type FFmpegConfig struct {
	BinaryPath  string   `yaml:"binary_path" toml:"binary_path" env:"AUDIO_FFMPEG_BINARY"`
	AudioFormat string   `yaml:"audio_format" toml:"audio_format" env:"AUDIO_FFMPEG_FORMAT"`
	SampleRate  int      `yaml:"sample_rate" toml:"sample_rate" env:"AUDIO_FFMPEG_SAMPLE_RATE"`
	Channels    int      `yaml:"channels" toml:"channels" env:"AUDIO_FFMPEG_CHANNELS"`
	CustomArgs  []string `yaml:"custom_args" toml:"custom_args" env:"AUDIO_FFMPEG_CUSTOM_ARGS"`
}

// YtDlpConfig contains yt-dlp-specific configuration
type YtDlpConfig struct {
	BinaryPath string   `yaml:"binary_path" toml:"binary_path" env:"AUDIO_YTDLP_BINARY"`
	CustomArgs []string `yaml:"custom_args" toml:"custom_args" env:"AUDIO_YTDLP_CUSTOM_ARGS"`
}

// OpusConfig contains Opus encoder configuration
type OpusConfig struct {
	Bitrate   int `yaml:"bitrate" toml:"bitrate" env:"AUDIO_OPUS_BITRATE"`
	FrameSize int `yaml:"frame_size" toml:"frame_size" env:"AUDIO_OPUS_FRAME_SIZE"`
}

// RetryConfig contains retry logic configuration
type RetryConfig struct {
	MaxRetries int           `yaml:"max_retries" toml:"max_retries" env:"AUDIO_MAX_RETRIES"`
	BaseDelay  time.Duration `yaml:"base_delay" toml:"base_delay" env:"AUDIO_BASE_DELAY"`
	MaxDelay   time.Duration `yaml:"max_delay" toml:"max_delay" env:"AUDIO_MAX_DELAY"`
	Multiplier float64       `yaml:"multiplier" toml:"multiplier" env:"AUDIO_RETRY_MULTIPLIER"`
}

// LoggerConfig contains logging configuration
type LoggerConfig struct {
	Level    string `yaml:"level" toml:"level" env:"AUDIO_LOG_LEVEL"`
	Format   string `yaml:"format" toml:"format" env:"AUDIO_LOG_FORMAT"`
	SaveToDB bool   `yaml:"save_to_db" toml:"save_to_db" env:"AUDIO_LOG_SAVE_DB"`
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

// ConfigManager implements the ConfigProvider interface
type ConfigManager struct {
	pipeline *PipelineConfig
	ffmpeg   *FFmpegConfig
	ytdlp    *YtDlpConfig
	opus     *OpusConfig
	retry    *RetryConfig
	logger   *LoggerConfig
}

// AudioConfig represents the complete configuration structure for YAML/TOML files
type AudioConfig struct {
	Pipeline PipelineConfig `yaml:"pipeline" toml:"pipeline"`
	FFmpeg   FFmpegConfig   `yaml:"ffmpeg" toml:"ffmpeg"`
	YtDlp    YtDlpConfig    `yaml:"ytdlp" toml:"ytdlp"`
	Opus     OpusConfig     `yaml:"opus" toml:"opus"`
	Retry    RetryConfig    `yaml:"retry" toml:"retry"`
	Logger   LoggerConfig   `yaml:"logger" toml:"logger"`
}

// NewConfigManager creates a new ConfigManager with configuration loaded from multiple sources
func NewConfigManager() (ConfigProvider, error) {
	manager := &ConfigManager{}

	// Try to load configuration in order of preference:
	// 1. YAML file (config/audio.yaml)
	// 2. TOML file (config/audio.toml)
	// 3. Environment variables (.env file)
	// 4. Default values

	config := &AudioConfig{}

	// Try loading YAML first
	if err := manager.loadYAMLConfig(config); err != nil {
		// Try loading TOML if YAML fails
		if err := manager.loadTOMLConfig(config); err != nil {
			// Fall back to environment variables
			if err := manager.loadEnvConfig(config); err != nil {
				// Use default values
				manager.setDefaults(config)
			}
		}
	}

	// Set the configuration in the manager
	manager.pipeline = &config.Pipeline
	manager.ffmpeg = &config.FFmpeg
	manager.ytdlp = &config.YtDlp
	manager.opus = &config.Opus
	manager.retry = &config.Retry
	manager.logger = &config.Logger

	// Validate the configuration
	if err := manager.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return manager, nil
}

// loadYAMLConfig attempts to load configuration from YAML file
func (cm *ConfigManager) loadYAMLConfig(config *AudioConfig) error {
	yamlPath := filepath.Join("config", "audio.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		return fmt.Errorf("YAML config file not found: %s", yamlPath)
	}

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to read YAML config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return nil
}

// loadTOMLConfig attempts to load configuration from TOML file
func (cm *ConfigManager) loadTOMLConfig(config *AudioConfig) error {
	tomlPath := filepath.Join("config", "audio.toml")
	if _, err := os.Stat(tomlPath); os.IsNotExist(err) {
		return fmt.Errorf("TOML config file not found: %s", tomlPath)
	}

	if _, err := toml.DecodeFile(tomlPath, config); err != nil {
		return fmt.Errorf("failed to parse TOML config: %w", err)
	}

	return nil
}

// loadEnvConfig loads configuration from environment variables
func (cm *ConfigManager) loadEnvConfig(config *AudioConfig) error {
	// Load .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			return fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	// Load pipeline config from environment
	config.Pipeline = PipelineConfig{
		RetryCount:     getEnvInt("AUDIO_RETRY_COUNT", 3),
		TimeoutSeconds: getEnvInt("AUDIO_TIMEOUT", 30),
		FFmpegOptions:  getEnvStringSlice("AUDIO_FFMPEG_OPTIONS", []string{"-reconnect", "1", "-reconnect_delay_max", "5"}),
		LogLevel:       getEnvString("AUDIO_LOG_LEVEL", "info"),
	}

	// Load FFmpeg config from environment
	config.FFmpeg = FFmpegConfig{
		BinaryPath:  getEnvString("AUDIO_FFMPEG_BINARY", "ffmpeg"),
		AudioFormat: getEnvString("AUDIO_FFMPEG_FORMAT", "s16le"),
		SampleRate:  getEnvInt("AUDIO_FFMPEG_SAMPLE_RATE", 48000),
		Channels:    getEnvInt("AUDIO_FFMPEG_CHANNELS", 2),
		CustomArgs:  getEnvStringSlice("AUDIO_FFMPEG_CUSTOM_ARGS", []string{"-reconnect", "1", "-reconnect_delay_max", "5"}),
	}

	// Load yt-dlp config from environment
	config.YtDlp = YtDlpConfig{
		BinaryPath: getEnvString("AUDIO_YTDLP_BINARY", "yt-dlp"),
		CustomArgs: getEnvStringSlice("AUDIO_YTDLP_CUSTOM_ARGS", []string{"--no-playlist", "--extract-flat"}),
	}

	// Load Opus config from environment
	config.Opus = OpusConfig{
		Bitrate:   getEnvInt("AUDIO_OPUS_BITRATE", 128000),
		FrameSize: getEnvInt("AUDIO_OPUS_FRAME_SIZE", 960),
	}

	// Load retry config from environment
	config.Retry = RetryConfig{
		MaxRetries: getEnvInt("AUDIO_MAX_RETRIES", 3),
		BaseDelay:  getEnvDuration("AUDIO_BASE_DELAY", 2*time.Second),
		MaxDelay:   getEnvDuration("AUDIO_MAX_DELAY", 30*time.Second),
		Multiplier: getEnvFloat("AUDIO_RETRY_MULTIPLIER", 2.0),
	}

	// Load logger config from environment
	config.Logger = LoggerConfig{
		Level:    getEnvString("AUDIO_LOG_LEVEL", "info"),
		Format:   getEnvString("AUDIO_LOG_FORMAT", "json"),
		SaveToDB: getEnvBool("AUDIO_LOG_SAVE_DB", true),
	}

	return nil
}

// setDefaults sets default configuration values
func (cm *ConfigManager) setDefaults(config *AudioConfig) {
	config.Pipeline = PipelineConfig{
		RetryCount:     3,
		TimeoutSeconds: 30,
		FFmpegOptions:  []string{"-reconnect", "1", "-reconnect_delay_max", "5"},
		LogLevel:       "info",
	}

	config.FFmpeg = FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{"-reconnect", "1", "-reconnect_delay_max", "5"},
	}

	config.YtDlp = YtDlpConfig{
		BinaryPath: "yt-dlp",
		CustomArgs: []string{"--no-playlist", "--extract-flat"},
	}

	config.Opus = OpusConfig{
		Bitrate:   128000,
		FrameSize: 960,
	}

	config.Retry = RetryConfig{
		MaxRetries: 3,
		BaseDelay:  2 * time.Second,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
	}

	config.Logger = LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: true,
	}
}

// GetPipelineConfig returns the pipeline configuration
func (cm *ConfigManager) GetPipelineConfig() *PipelineConfig {
	return cm.pipeline
}

// GetFFmpegConfig returns the FFmpeg configuration
func (cm *ConfigManager) GetFFmpegConfig() *FFmpegConfig {
	return cm.ffmpeg
}

// GetYtDlpConfig returns the yt-dlp configuration
func (cm *ConfigManager) GetYtDlpConfig() *YtDlpConfig {
	return cm.ytdlp
}

// GetOpusConfig returns the Opus configuration
func (cm *ConfigManager) GetOpusConfig() *OpusConfig {
	return cm.opus
}

// GetRetryConfig returns the retry configuration
func (cm *ConfigManager) GetRetryConfig() *RetryConfig {
	return cm.retry
}

// GetLoggerConfig returns the logger configuration
func (cm *ConfigManager) GetLoggerConfig() *LoggerConfig {
	return cm.logger
}

// Validate validates the configuration values
func (cm *ConfigManager) Validate() error {
	// Validate pipeline config
	if cm.pipeline.RetryCount < 0 {
		return fmt.Errorf("pipeline retry_count must be non-negative, got %d", cm.pipeline.RetryCount)
	}
	if cm.pipeline.TimeoutSeconds <= 0 {
		return fmt.Errorf("pipeline timeout_seconds must be positive, got %d", cm.pipeline.TimeoutSeconds)
	}
	if !isValidLogLevel(cm.pipeline.LogLevel) {
		return fmt.Errorf("invalid pipeline log_level: %s (must be debug, info, warn, or error)", cm.pipeline.LogLevel)
	}

	// Validate FFmpeg config
	if cm.ffmpeg.BinaryPath == "" {
		return fmt.Errorf("ffmpeg binary_path cannot be empty")
	}
	if cm.ffmpeg.SampleRate <= 0 {
		return fmt.Errorf("ffmpeg sample_rate must be positive, got %d", cm.ffmpeg.SampleRate)
	}
	if cm.ffmpeg.Channels <= 0 {
		return fmt.Errorf("ffmpeg channels must be positive, got %d", cm.ffmpeg.Channels)
	}
	if !isValidAudioFormat(cm.ffmpeg.AudioFormat) {
		return fmt.Errorf("invalid ffmpeg audio_format: %s", cm.ffmpeg.AudioFormat)
	}

	// Validate yt-dlp config
	if cm.ytdlp.BinaryPath == "" {
		return fmt.Errorf("yt-dlp binary_path cannot be empty")
	}

	// Validate Opus config
	if cm.opus.Bitrate <= 0 {
		return fmt.Errorf("opus bitrate must be positive, got %d", cm.opus.Bitrate)
	}
	if cm.opus.FrameSize <= 0 {
		return fmt.Errorf("opus frame_size must be positive, got %d", cm.opus.FrameSize)
	}

	// Validate retry config
	if cm.retry.MaxRetries < 0 {
		return fmt.Errorf("retry max_retries must be non-negative, got %d", cm.retry.MaxRetries)
	}
	if cm.retry.BaseDelay <= 0 {
		return fmt.Errorf("retry base_delay must be positive, got %v", cm.retry.BaseDelay)
	}
	if cm.retry.MaxDelay <= 0 {
		return fmt.Errorf("retry max_delay must be positive, got %v", cm.retry.MaxDelay)
	}
	if cm.retry.Multiplier <= 1.0 {
		return fmt.Errorf("retry multiplier must be greater than 1.0, got %f", cm.retry.Multiplier)
	}

	// Validate logger config
	if !isValidLogLevel(cm.logger.Level) {
		return fmt.Errorf("invalid logger level: %s (must be debug, info, warn, or error)", cm.logger.Level)
	}
	if !isValidLogFormat(cm.logger.Format) {
		return fmt.Errorf("invalid logger format: %s (must be json or text)", cm.logger.Format)
	}

	return nil
}

// ValidateDependencies validates that all required binary dependencies are available
func (cm *ConfigManager) ValidateDependencies() error {
	// Validate all binary dependencies using shared utilities
	return ValidateAllBinaryDependencies(cm.ffmpeg.BinaryPath, cm.ytdlp.BinaryPath)
}

// Helper functions for environment variable parsing
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// Validation helper functions
func isValidLogLevel(level string) bool {
	validLevels := []string{"debug", "info", "warn", "error"}
	for _, valid := range validLevels {
		if strings.ToLower(level) == valid {
			return true
		}
	}
	return false
}

func isValidLogFormat(format string) bool {
	validFormats := []string{"json", "text"}
	for _, valid := range validFormats {
		if strings.ToLower(format) == valid {
			return true
		}
	}
	return false
}

func isValidAudioFormat(format string) bool {
	validFormats := []string{"s16le", "s16be", "s32le", "s32be", "f32le", "f32be"}
	for _, valid := range validFormats {
		if strings.ToLower(format) == valid {
			return true
		}
	}
	return false
}
