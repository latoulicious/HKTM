package audio

import (
	"testing"

	"github.com/latoulicious/HKTM/pkg/audio"
)

// MockAudioLogger is a simple mock implementation of AudioLogger for testing
type MockAudioLogger struct{}

func (m *MockAudioLogger) Info(msg string, fields map[string]interface{})                    {}
func (m *MockAudioLogger) Error(msg string, err error, fields map[string]interface{})       {}
func (m *MockAudioLogger) Warn(msg string, fields map[string]interface{})                   {}
func (m *MockAudioLogger) Debug(msg string, fields map[string]interface{})                  {}
func (m *MockAudioLogger) WithPipeline(pipeline string) audio.AudioLogger                   { return m }
func (m *MockAudioLogger) WithContext(ctx map[string]interface{}) audio.AudioLogger         { return m }

func TestOpusProcessor_PCMToOpusConversion(t *testing.T) {
	// Create Opus configuration for Discord
	config := &audio.OpusConfig{
		Bitrate:   128000,
		FrameSize: 960, // 20ms at 48kHz
	}

	logger := &MockAudioLogger{}
	processor := audio.NewOpusProcessor(config, logger)

	// Initialize the processor
	err := processor.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize Opus processor: %v", err)
	}
	defer processor.Close()

	t.Run("ValidFrameEncoding", func(t *testing.T) {
		// Create a valid PCM frame (960 samples per channel * 2 channels = 1920 samples)
		frameSize := processor.GetFrameSize()
		pcmFrame := make([]int16, frameSize)

		// Fill with test audio data (simple sine wave pattern)
		for i := 0; i < frameSize; i += 2 {
			// Left channel
			pcmFrame[i] = int16(i % 1000)
			// Right channel
			pcmFrame[i+1] = int16((i + 500) % 1000)
		}

		// Encode the frame
		opusData, err := processor.EncodeFrame(pcmFrame)
		if err != nil {
			t.Fatalf("Failed to encode PCM frame: %v", err)
		}

		// Validate the output
		if len(opusData) == 0 {
			t.Error("Encoded Opus data is empty")
		}

		if len(opusData) > 4000 {
			t.Errorf("Encoded Opus frame too large: %d bytes", len(opusData))
		}

		t.Logf("Successfully encoded %d PCM samples to %d Opus bytes", frameSize, len(opusData))
	})

	t.Run("InvalidFrameSize", func(t *testing.T) {
		// Test with wrong frame size
		wrongSizeFrame := make([]int16, 100) // Too small

		_, err := processor.EncodeFrame(wrongSizeFrame)
		if err == nil {
			t.Error("Expected error for invalid frame size, got nil")
		}

		t.Logf("Correctly rejected invalid frame size: %v", err)
	})

	t.Run("EmptyFrame", func(t *testing.T) {
		// Test with empty frame
		emptyFrame := make([]int16, 0)

		_, err := processor.EncodeFrame(emptyFrame)
		if err == nil {
			t.Error("Expected error for empty frame, got nil")
		}

		t.Logf("Correctly rejected empty frame: %v", err)
	})

	t.Run("FrameSizeValidation", func(t *testing.T) {
		expectedSize := 1920 // 960 samples per channel * 2 channels
		actualSize := processor.GetFrameSize()

		if actualSize != expectedSize {
			t.Errorf("Expected frame size %d, got %d", expectedSize, actualSize)
		}
	})

	t.Run("FrameDuration", func(t *testing.T) {
		duration := processor.GetFrameDuration()
		expectedDuration := 20 // 20ms for Discord

		if duration.Milliseconds() != int64(expectedDuration) {
			t.Errorf("Expected frame duration %dms, got %dms", expectedDuration, duration.Milliseconds())
		}
	})

	t.Run("StreamingPreparation", func(t *testing.T) {
		err := processor.PrepareForStreaming()
		if err != nil {
			t.Errorf("Failed streaming preparation: %v", err)
		}
	})
}

func TestOpusProcessor_ErrorHandling(t *testing.T) {
	config := &audio.OpusConfig{
		Bitrate:   128000,
		FrameSize: 960,
	}

	logger := &MockAudioLogger{}
	processor := audio.NewOpusProcessor(config, logger)

	t.Run("UninitializedEncoder", func(t *testing.T) {
		// Try to encode without initializing
		pcmFrame := make([]int16, 1920)
		_, err := processor.EncodeFrame(pcmFrame)
		if err == nil {
			t.Error("Expected error for uninitialized encoder, got nil")
		}
	})

	t.Run("ClosedEncoder", func(t *testing.T) {
		// Initialize and then close
		processor.Initialize()
		processor.Close()

		pcmFrame := make([]int16, 1920)
		_, err := processor.EncodeFrame(pcmFrame)
		if err == nil {
			t.Error("Expected error for closed encoder, got nil")
		}
	})
}

func TestOpusProcessor_DiscordCompatibility(t *testing.T) {
	t.Run("InvalidFrameSizeForDiscord", func(t *testing.T) {
		// Test with invalid frame size for Discord
		invalidConfig := &audio.OpusConfig{
			Bitrate:   128000,
			FrameSize: 480, // Wrong size for Discord (should be 960)
		}

		logger := &MockAudioLogger{}
		processor := audio.NewOpusProcessor(invalidConfig, logger)
		err := processor.Initialize()
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}
		defer processor.Close()

		// This should fail during streaming preparation
		err = processor.PrepareForStreaming()
		if err == nil {
			t.Error("Expected error for invalid Discord frame size, got nil")
		}
	})

	t.Run("InvalidBitrateForDiscord", func(t *testing.T) {
		// Test with invalid bitrate
		invalidConfig := &audio.OpusConfig{
			Bitrate:   1000000, // Too high for Discord
			FrameSize: 960,
		}

		logger := &MockAudioLogger{}
		processor := audio.NewOpusProcessor(invalidConfig, logger)
		err := processor.Initialize()
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}
		defer processor.Close()

		// This should fail during streaming preparation
		err = processor.PrepareForStreaming()
		if err == nil {
			t.Error("Expected error for invalid Discord bitrate, got nil")
		}
	})
}

func TestOpusProcessor_MultipleFrames(t *testing.T) {
	config := &audio.OpusConfig{
		Bitrate:   128000,
		FrameSize: 960,
	}

	logger := &MockAudioLogger{}
	processor := audio.NewOpusProcessor(config, logger)
	err := processor.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	defer processor.Close()

	frameSize := processor.GetFrameSize()

	// Test encoding multiple consecutive frames (simulating streaming)
	for i := 0; i < 10; i++ {
		pcmFrame := make([]int16, frameSize)

		// Fill with different test data for each frame
		for j := 0; j < frameSize; j += 2 {
			pcmFrame[j] = int16((i*1000 + j) % 32767)   // Left channel
			pcmFrame[j+1] = int16((i*1500 + j) % 32767) // Right channel
		}

		opusData, err := processor.EncodeFrame(pcmFrame)
		if err != nil {
			t.Fatalf("Failed to encode frame %d: %v", i, err)
		}

		if len(opusData) == 0 {
			t.Errorf("Frame %d produced empty Opus data", i)
		}

		t.Logf("Frame %d: %d PCM samples -> %d Opus bytes", i, frameSize, len(opusData))
	}
}
