package audio

import (
	"fmt"
	"sync"

	"layeh.com/gopus"
)

// OpusProcessor implements the AudioEncoder interface for Opus encoding
type OpusProcessor struct {
	encoder *gopus.Encoder
	config  *OpusConfig
	mu      sync.RWMutex
	closed  bool
}

// NewOpusProcessor creates a new OpusProcessor with the given configuration
func NewOpusProcessor(config *OpusConfig) AudioEncoder {
	return &OpusProcessor{
		config: config,
	}
}

// Initialize initializes the Opus encoder with Discord-compatible parameters
func (op *OpusProcessor) Initialize() error {
	op.mu.Lock()
	defer op.mu.Unlock()

	if op.encoder != nil {
		return fmt.Errorf("opus encoder already initialized")
	}

	// Discord requires 48kHz sample rate, stereo (2 channels)
	// These are fixed values for Discord compatibility
	const (
		sampleRate = 48000
		channels   = 2
	)

	// Create the Opus encoder
	encoder, err := gopus.NewEncoder(sampleRate, channels, gopus.Audio)
	if err != nil {
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

	return nil
}

// Encode encodes PCM audio data to Opus format for Discord streaming
func (op *OpusProcessor) Encode(pcmData []int16) ([]byte, error) {
	op.mu.RLock()
	defer op.mu.RUnlock()

	if op.encoder == nil {
		return nil, fmt.Errorf("opus encoder not initialized")
	}

	if op.closed {
		return nil, fmt.Errorf("opus encoder is closed")
	}

	if len(pcmData) == 0 {
		return nil, fmt.Errorf("empty PCM data provided")
	}

	// Validate frame size matches configuration
	// Discord expects specific frame sizes (960 samples for 20ms at 48kHz)
	expectedFrameSize := op.config.FrameSize * 2 // stereo, so 2 channels
	if len(pcmData) != expectedFrameSize {
		return nil, fmt.Errorf("invalid PCM frame size: expected %d samples, got %d", expectedFrameSize, len(pcmData))
	}

	// Encode the PCM data to Opus
	opusData, err := op.encoder.Encode(pcmData, op.config.FrameSize, 4000)
	if err != nil {
		return nil, fmt.Errorf("failed to encode PCM to opus: %w", err)
	}

	return opusData, nil
}

// Close closes the Opus encoder and releases resources
func (op *OpusProcessor) Close() error {
	op.mu.Lock()
	defer op.mu.Unlock()

	if op.encoder == nil {
		return nil // Already closed or never initialized
	}

	if op.closed {
		return nil // Already closed
	}

	// The gopus encoder doesn't have an explicit Close method
	// We just need to set it to nil and mark as closed
	op.encoder = nil
	op.closed = true

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
