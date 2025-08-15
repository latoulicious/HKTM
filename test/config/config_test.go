package config

//! Uncomment if needed for further testing and resolve the import errors

// import (
// 	"os"
// 	"testing"
// 	"time"
// )

// func TestNewConfigManager_WithYAMLFile(t *testing.T) {
// 	// Test with existing YAML file
// 	config, err := NewConfigManager()
// 	if err != nil {
// 		t.Fatalf("Failed to create config manager: %v", err)
// 	}

// 	// Verify pipeline config
// 	pipelineConfig := config.GetPipelineConfig()
// 	if pipelineConfig.RetryCount != 3 {
// 		t.Errorf("Expected retry count 3, got %d", pipelineConfig.RetryCount)
// 	}
// 	if pipelineConfig.TimeoutSeconds != 30 {
// 		t.Errorf("Expected timeout 30, got %d", pipelineConfig.TimeoutSeconds)
// 	}
// 	if pipelineConfig.LogLevel != "info" {
// 		t.Errorf("Expected log level 'info', got %s", pipelineConfig.LogLevel)
// 	}

// 	// Verify FFmpeg config
// 	ffmpegConfig := config.GetFFmpegConfig()
// 	if ffmpegConfig.BinaryPath != "ffmpeg" {
// 		t.Errorf("Expected binary path 'ffmpeg', got %s", ffmpegConfig.BinaryPath)
// 	}
// 	if ffmpegConfig.SampleRate != 48000 {
// 		t.Errorf("Expected sample rate 48000, got %d", ffmpegConfig.SampleRate)
// 	}

// 	// Verify Opus config
// 	opusConfig := config.GetOpusConfig()
// 	if opusConfig.Bitrate != 128000 {
// 		t.Errorf("Expected bitrate 128000, got %d", opusConfig.Bitrate)
// 	}
// 	if opusConfig.FrameSize != 960 {
// 		t.Errorf("Expected frame size 960, got %d", opusConfig.FrameSize)
// 	}

// 	// Verify retry config
// 	retryConfig := config.GetRetryConfig()
// 	if retryConfig.MaxRetries != 3 {
// 		t.Errorf("Expected max retries 3, got %d", retryConfig.MaxRetries)
// 	}
// 	if retryConfig.BaseDelay != 2*time.Second {
// 		t.Errorf("Expected base delay 2s, got %v", retryConfig.BaseDelay)
// 	}

// 	// Verify logger config
// 	loggerConfig := config.GetLoggerConfig()
// 	if loggerConfig.Level != "info" {
// 		t.Errorf("Expected logger level 'info', got %s", loggerConfig.Level)
// 	}
// 	if loggerConfig.Format != "json" {
// 		t.Errorf("Expected logger format 'json', got %s", loggerConfig.Format)
// 	}
// }

// func TestNewConfigManager_WithEnvironmentVariables(t *testing.T) {
// 	// Temporarily rename the YAML file to force environment loading
// 	if err := os.Rename("config/audio.yaml", "config/audio.yaml.bak"); err != nil {
// 		t.Skip("Could not rename YAML file for test")
// 	}
// 	defer os.Rename("config/audio.yaml.bak", "config/audio.yaml")

// 	// Set environment variables
// 	os.Setenv("AUDIO_RETRY_COUNT", "5")
// 	os.Setenv("AUDIO_TIMEOUT", "60")
// 	os.Setenv("AUDIO_LOG_LEVEL", "debug")
// 	os.Setenv("AUDIO_FFMPEG_BINARY", "/usr/bin/ffmpeg")
// 	os.Setenv("AUDIO_OPUS_BITRATE", "256000")
// 	defer func() {
// 		os.Unsetenv("AUDIO_RETRY_COUNT")
// 		os.Unsetenv("AUDIO_TIMEOUT")
// 		os.Unsetenv("AUDIO_LOG_LEVEL")
// 		os.Unsetenv("AUDIO_FFMPEG_BINARY")
// 		os.Unsetenv("AUDIO_OPUS_BITRATE")
// 	}()

// 	config, err := NewConfigManager()
// 	if err != nil {
// 		t.Fatalf("Failed to create config manager: %v", err)
// 	}

// 	// Verify environment variables were loaded
// 	pipelineConfig := config.GetPipelineConfig()
// 	if pipelineConfig.RetryCount != 5 {
// 		t.Errorf("Expected retry count 5, got %d", pipelineConfig.RetryCount)
// 	}
// 	if pipelineConfig.TimeoutSeconds != 60 {
// 		t.Errorf("Expected timeout 60, got %d", pipelineConfig.TimeoutSeconds)
// 	}
// 	if pipelineConfig.LogLevel != "debug" {
// 		t.Errorf("Expected log level 'debug', got %s", pipelineConfig.LogLevel)
// 	}

// 	ffmpegConfig := config.GetFFmpegConfig()
// 	if ffmpegConfig.BinaryPath != "/usr/bin/ffmpeg" {
// 		t.Errorf("Expected binary path '/usr/bin/ffmpeg', got %s", ffmpegConfig.BinaryPath)
// 	}

// 	opusConfig := config.GetOpusConfig()
// 	if opusConfig.Bitrate != 256000 {
// 		t.Errorf("Expected bitrate 256000, got %d", opusConfig.Bitrate)
// 	}
// }

// func TestConfigManager_Validate(t *testing.T) {
// 	tests := []struct {
// 		name        string
// 		modifyFunc  func(*ConfigManager)
// 		expectError bool
// 	}{
// 		{
// 			name: "valid config",
// 			modifyFunc: func(cm *ConfigManager) {
// 				// No modifications - should be valid
// 			},
// 			expectError: false,
// 		},
// 		{
// 			name: "negative retry count",
// 			modifyFunc: func(cm *ConfigManager) {
// 				cm.pipeline.RetryCount = -1
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name: "zero timeout",
// 			modifyFunc: func(cm *ConfigManager) {
// 				cm.pipeline.TimeoutSeconds = 0
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name: "invalid log level",
// 			modifyFunc: func(cm *ConfigManager) {
// 				cm.pipeline.LogLevel = "invalid"
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name: "empty ffmpeg binary path",
// 			modifyFunc: func(cm *ConfigManager) {
// 				cm.ffmpeg.BinaryPath = ""
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name: "zero sample rate",
// 			modifyFunc: func(cm *ConfigManager) {
// 				cm.ffmpeg.SampleRate = 0
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name: "zero opus bitrate",
// 			modifyFunc: func(cm *ConfigManager) {
// 				cm.opus.Bitrate = 0
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name: "invalid retry multiplier",
// 			modifyFunc: func(cm *ConfigManager) {
// 				cm.retry.Multiplier = 0.5
// 			},
// 			expectError: true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			config, err := NewConfigManager()
// 			if err != nil {
// 				t.Fatalf("Failed to create config manager: %v", err)
// 			}

// 			// Apply modifications
// 			tt.modifyFunc(config.(*ConfigManager))

// 			// Test validation
// 			err = config.Validate()
// 			if tt.expectError && err == nil {
// 				t.Errorf("Expected validation error, but got none")
// 			}
// 			if !tt.expectError && err != nil {
// 				t.Errorf("Expected no validation error, but got: %v", err)
// 			}
// 		})
// 	}
// }

// func TestHelperFunctions(t *testing.T) {
// 	// Test getEnvString
// 	os.Setenv("TEST_STRING", "test_value")
// 	defer os.Unsetenv("TEST_STRING")

// 	if result := getEnvString("TEST_STRING", "default"); result != "test_value" {
// 		t.Errorf("Expected 'test_value', got %s", result)
// 	}
// 	if result := getEnvString("NONEXISTENT", "default"); result != "default" {
// 		t.Errorf("Expected 'default', got %s", result)
// 	}

// 	// Test getEnvInt
// 	os.Setenv("TEST_INT", "42")
// 	defer os.Unsetenv("TEST_INT")

// 	if result := getEnvInt("TEST_INT", 0); result != 42 {
// 		t.Errorf("Expected 42, got %d", result)
// 	}
// 	if result := getEnvInt("NONEXISTENT", 10); result != 10 {
// 		t.Errorf("Expected 10, got %d", result)
// 	}

// 	// Test getEnvBool
// 	os.Setenv("TEST_BOOL", "true")
// 	defer os.Unsetenv("TEST_BOOL")

// 	if result := getEnvBool("TEST_BOOL", false); result != true {
// 		t.Errorf("Expected true, got %t", result)
// 	}
// 	if result := getEnvBool("NONEXISTENT", false); result != false {
// 		t.Errorf("Expected false, got %t", result)
// 	}

// 	// Test getEnvDuration
// 	os.Setenv("TEST_DURATION", "5s")
// 	defer os.Unsetenv("TEST_DURATION")

// 	if result := getEnvDuration("TEST_DURATION", time.Second); result != 5*time.Second {
// 		t.Errorf("Expected 5s, got %v", result)
// 	}
// 	if result := getEnvDuration("NONEXISTENT", time.Second); result != time.Second {
// 		t.Errorf("Expected 1s, got %v", result)
// 	}
// }

// func TestValidationHelpers(t *testing.T) {
// 	// Test isValidLogLevel
// 	validLevels := []string{"debug", "info", "warn", "error", "DEBUG", "INFO"}
// 	for _, level := range validLevels {
// 		if !isValidLogLevel(level) {
// 			t.Errorf("Expected %s to be valid log level", level)
// 		}
// 	}

// 	invalidLevels := []string{"invalid", "trace", "fatal"}
// 	for _, level := range invalidLevels {
// 		if isValidLogLevel(level) {
// 			t.Errorf("Expected %s to be invalid log level", level)
// 		}
// 	}

// 	// Test isValidLogFormat
// 	validFormats := []string{"json", "text", "JSON", "TEXT"}
// 	for _, format := range validFormats {
// 		if !isValidLogFormat(format) {
// 			t.Errorf("Expected %s to be valid log format", format)
// 		}
// 	}

// 	invalidFormats := []string{"xml", "yaml", "invalid"}
// 	for _, format := range invalidFormats {
// 		if isValidLogFormat(format) {
// 			t.Errorf("Expected %s to be invalid log format", format)
// 		}
// 	}

// 	// Test isValidAudioFormat
// 	validAudioFormats := []string{"s16le", "s16be", "s32le", "f32le"}
// 	for _, format := range validAudioFormats {
// 		if !isValidAudioFormat(format) {
// 			t.Errorf("Expected %s to be valid audio format", format)
// 		}
// 	}

// 	invalidAudioFormats := []string{"mp3", "wav", "invalid"}
// 	for _, format := range invalidAudioFormats {
// 		if isValidAudioFormat(format) {
// 			t.Errorf("Expected %s to be invalid audio format", format)
// 		}
// 	}
// }
