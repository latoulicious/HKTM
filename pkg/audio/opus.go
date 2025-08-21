package audio

import (
	"fmt"
	"sync"
	"time"

	"layeh.com/gopus"
)

// OpusProcessor implements the AudioEncoder interface for Opus encoding
type OpusProcessor struct {
	encoder *gopus.Encoder
	config  *OpusConfig
	logger  AudioLogger
	mu      sync.RWMutex
	closed  bool
}

// NewOpusProcessor creates a new OpusProcessor with the given configuration and logger
func NewOpusProcessor(config *OpusConfig, logger AudioLogger) AudioEncoder {
	// Create pipeline-specific logger for Opus encoding operations
	pipelineLogger := logger.WithPipeline("opus")

	return &OpusProcessor{
		config: config,
		logger: pipelineLogger,
	}
}

// Initialize initializes the Opus encoder with Discord-compatible parameters
func (op *OpusProcessor) Initialize() error {
	op.mu.Lock()
	defer op.mu.Unlock()

	if op.encoder != nil {
		op.logger.Debug("Opus encoder already initialized, skipping", nil)
		return nil // Already initialized - this is not an error
	}

	// Discord requires 48kHz sample rate, stereo (2 channels)
	// These are fixed values for Discord compatibility
	const (
		sampleRate = 48000
		channels   = 2
	)

	op.logger.Info("Initializing Opus encoder", map[string]interface{}{
		"sample_rate": sampleRate,
		"channels":    channels,
		"bitrate":     op.config.Bitrate,
		"frame_size":  op.config.FrameSize,
	})

	// Create the Opus encoder
	encoder, err := gopus.NewEncoder(sampleRate, channels, gopus.Audio)
	if err != nil {
		op.logger.Error("Failed to create Opus encoder", err, map[string]interface{}{
			"sample_rate": sampleRate,
			"channels":    channels,
		})
		return fmt.Errorf("failed to create opus encoder: %w", err)
	}

	// Set bitrate from configuration
	encoder.SetBitrate(op.config.Bitrate)

	// Set additional Discord-optimized settings
	// Enable variable bitrate for better quality
	encoder.SetVbr(true)

	// Note: SetComplexity and SetPacketLossPerc are not available in this version of gopus
	// The encoder will use default values which are suitable for Discord

	op.encoder = encoder
	op.closed = false

	op.logger.Info("Opus encoder initialized successfully", map[string]interface{}{
		"bitrate":    op.config.Bitrate,
		"frame_size": op.config.FrameSize,
		"vbr":        true,
	})

	return nil
}

// Encode encodes PCM audio data to Opus format for Discord streaming
// Implements frame-based encoding with proper Discord timing and error handling
func (op *OpusProcessor) Encode(pcmData []int16) ([]byte, error) {
	op.mu.RLock()
	defer op.mu.RUnlock()

	if op.encoder == nil {
		op.logger.Error("Encode called on uninitialized encoder", nil, map[string]interface{}{
			"pcm_samples": len(pcmData),
		})
		return nil, fmt.Errorf("opus encoder not initialized")
	}

	if op.closed {
		op.logger.Error("Encode called on closed encoder", nil, map[string]interface{}{
			"pcm_samples": len(pcmData),
		})
		return nil, fmt.Errorf("opus encoder is closed")
	}

	if len(pcmData) == 0 {
		op.logger.Warn("Empty PCM data provided for encoding", nil)
		return nil, fmt.Errorf("empty PCM data provided")
	}

	// Discord requires specific frame sizes for proper timing
	// Frame size is per channel, so for stereo we need frameSize * channels samples
	const channels = 2 // Discord always uses stereo
	expectedFrameSize := op.config.FrameSize * channels

	if len(pcmData) != expectedFrameSize {
		op.logger.Error("Invalid PCM frame size", nil, map[string]interface{}{
			"expected_samples":       expectedFrameSize,
			"received_samples":       len(pcmData),
			"frame_size_per_channel": op.config.FrameSize,
			"channels":               channels,
		})
		return nil, fmt.Errorf("invalid PCM frame size: expected %d samples (%d per channel * %d channels), got %d",
			expectedFrameSize, op.config.FrameSize, channels, len(pcmData))
	}

	// Validate frame size is appropriate for Discord (20ms frames at 48kHz)
	// 48000 Hz * 0.02s = 960 samples per channel
	if op.config.FrameSize != 960 {
		op.logger.Error("Invalid frame size for Discord", nil, map[string]interface{}{
			"configured_frame_size": op.config.FrameSize,
			"expected_frame_size":   960,
		})
		return nil, fmt.Errorf("invalid frame size for Discord: expected 960 samples per channel, got %d", op.config.FrameSize)
	}

	// Skip debug logging for individual frames to improve performance

	// Encode the PCM data to Opus with proper error handling
	// The maxDataBytes parameter (4000) provides enough space for worst-case Opus frame
	opusData, err := op.encoder.Encode(pcmData, op.config.FrameSize, 4000)
	if err != nil {
		// Provide detailed error context for debugging
		op.logger.Error("Failed to encode PCM to Opus", err, map[string]interface{}{
			"frame_size":  op.config.FrameSize,
			"pcm_samples": len(pcmData),
			"bitrate":     op.config.Bitrate,
		})
		return nil, fmt.Errorf("failed to encode PCM to Opus (frame_size=%d, samples=%d, bitrate=%d): %w",
			op.config.FrameSize, len(pcmData), op.config.Bitrate, err)
	}

	// Validate that we got a reasonable Opus frame
	if len(opusData) == 0 {
		op.logger.Error("Opus encoder returned empty frame", nil, map[string]interface{}{
			"pcm_samples": len(pcmData),
			"frame_size":  op.config.FrameSize,
		})
		return nil, fmt.Errorf("opus encoder returned empty frame")
	}

	// Discord expects Opus frames to be reasonable size (typically 20-1000 bytes)
	if len(opusData) > 4000 {
		op.logger.Error("Opus frame too large", nil, map[string]interface{}{
			"opus_bytes":  len(opusData),
			"max_bytes":   4000,
			"pcm_samples": len(pcmData),
		})
		return nil, fmt.Errorf("opus frame too large: %d bytes (max 4000)", len(opusData))
	}

	// Skip individual frame logging to improve performance

	return opusData, nil
}

// Close closes the Opus encoder and releases resources
func (op *OpusProcessor) Close() error {
	op.mu.Lock()
	defer op.mu.Unlock()

	if op.encoder == nil {
		op.logger.Debug("Close called on uninitialized encoder", nil)
		return nil // Already closed or never initialized
	}

	if op.closed {
		op.logger.Debug("Close called on already closed encoder", nil)
		return nil // Already closed
	}

	op.logger.Info("Closing Opus encoder", nil)

	// The gopus encoder doesn't have an explicit Close method
	// We just need to set it to nil and mark as closed
	op.encoder = nil
	op.closed = true

	op.logger.Info("Opus encoder closed successfully", nil)

	return nil
}

// IsInitialized returns whether the encoder is initialized and ready to use
func (op *OpusProcessor) IsInitialized() bool {
	op.mu.RLock()
	defer op.mu.RUnlock()
	return op.encoder != nil && !op.closed
}

// GetConfig returns the current Opus configuration
func (op *OpusProcessor) GetConfig() *OpusConfig {
	return op.config
}

// EncodeFrame encodes a single PCM frame with Discord-specific validation
// This method ensures proper frame timing for Discord streaming (20ms frames)
func (op *OpusProcessor) EncodeFrame(pcmFrame []int16) ([]byte, error) {
	// Use the main Encode method with additional frame validation
	// Note: Debug logging removed to prevent database bottleneck
	return op.Encode(pcmFrame)
}

// GetFrameSize returns the expected PCM frame size in samples (total for all channels)
func (op *OpusProcessor) GetFrameSize() int {
	const channels = 2 // Discord always uses stereo
	return op.config.FrameSize * channels
}

// GetFrameDuration returns the duration of each frame for timing calculations
func (op *OpusProcessor) GetFrameDuration() time.Duration {
	// Discord uses 20ms frames (960 samples at 48kHz)
	return 20 * time.Millisecond
}

// ValidateFrameSize validates that the provided PCM data matches Discord requirements
func (op *OpusProcessor) ValidateFrameSize(pcmData []int16) error {
	expectedSize := op.GetFrameSize()
	if len(pcmData) != expectedSize {
		return fmt.Errorf("invalid frame size: expected %d samples, got %d", expectedSize, len(pcmData))
	}
	return nil
}

// PrepareForStreaming validates the encoder configuration for Discord streaming
func (op *OpusProcessor) PrepareForStreaming() error {
	op.mu.RLock()
	defer op.mu.RUnlock()

	op.logger.Info("Preparing Opus encoder for streaming", map[string]interface{}{
		"frame_size": op.config.FrameSize,
		"bitrate":    op.config.Bitrate,
	})

	if op.encoder == nil {
		op.logger.Error("Cannot prepare uninitialized encoder for streaming", nil, nil)
		return fmt.Errorf("opus encoder not initialized - call Initialize() first")
	}

	if op.closed {
		op.logger.Error("Cannot prepare closed encoder for streaming", nil, nil)
		return fmt.Errorf("opus encoder is closed")
	}

	// Validate configuration for Discord compatibility
	if op.config.FrameSize != 960 {
		op.logger.Error("Invalid frame size for Discord streaming", nil, map[string]interface{}{
			"configured_frame_size": op.config.FrameSize,
			"expected_frame_size":   960,
		})
		return fmt.Errorf("invalid frame size for Discord streaming: expected 960, got %d", op.config.FrameSize)
	}

	// Validate bitrate is reasonable for Discord
	if op.config.Bitrate < 8000 || op.config.Bitrate > 512000 {
		op.logger.Error("Invalid bitrate for Discord streaming", nil, map[string]interface{}{
			"configured_bitrate": op.config.Bitrate,
			"min_bitrate":        8000,
			"max_bitrate":        512000,
		})
		return fmt.Errorf("invalid bitrate for Discord: %d (should be between 8000-512000)", op.config.Bitrate)
	}

	op.logger.Info("Opus encoder prepared for streaming successfully", map[string]interface{}{
		"frame_size": op.config.FrameSize,
		"bitrate":    op.config.Bitrate,
		"status":     "ready",
	})

	return nil
}
