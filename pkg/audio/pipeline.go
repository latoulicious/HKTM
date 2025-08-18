package audio

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// PipelineState represents the current state of the audio pipeline
type PipelineState int

const (
	StateStopped PipelineState = iota
	StateStarting
	StatePlaying
	StatePaused
	StateError
)

// String returns the string representation of the pipeline state
func (s PipelineState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// AudioPipelineController coordinates all audio pipeline components
// It acts as a coordinator only - delegates all business logic to injected dependencies
type AudioPipelineController struct {
	// Injected dependencies (interfaces only)
	streamProcessor StreamProcessor
	audioEncoder    AudioEncoder
	errorHandler    ErrorHandler
	metrics         MetricsCollector
	logger          AudioLogger
	config          ConfigProvider

	// State management only
	state       PipelineState
	currentURL  string
	voiceConn   *discordgo.VoiceConnection
	startTime   time.Time
	errorCount  int
	lastError   error
	stopChan    chan struct{}
	mu          sync.RWMutex
	ctx         context.Context
	cancelFunc  context.CancelFunc
}

// NewAudioPipelineController creates a new AudioPipelineController with injected dependencies
func NewAudioPipelineController(
	streamProcessor StreamProcessor,
	audioEncoder AudioEncoder,
	errorHandler ErrorHandler,
	metrics MetricsCollector,
	logger AudioLogger,
	config ConfigProvider,
) *AudioPipelineController {
	ctx, cancel := context.WithCancel(context.Background())

	return &AudioPipelineController{
		streamProcessor: streamProcessor,
		audioEncoder:    audioEncoder,
		errorHandler:    errorHandler,
		metrics:         metrics,
		logger:          logger,
		config:          config,
		state:           StateStopped,
		stopChan:        make(chan struct{}),
		ctx:             ctx,
		cancelFunc:      cancel,
	}
}

// PlayURL starts playback of the given URL using the voice connection
// Implements the AudioPipeline interface
func (c *AudioPipelineController) PlayURL(url string, voiceConn *discordgo.VoiceConnection) error {
	// Validate using shared utility
	if err := ValidateURL(url); err != nil {
		c.logger.Error("URL validation failed", err, CreateContextFields("", "", url))
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check if already playing
	c.mu.Lock()
	if c.state == StatePlaying || c.state == StateStarting {
		currentURL := c.currentURL
		c.mu.Unlock()
		return fmt.Errorf("pipeline is already playing: %s", currentURL)
	}
	c.mu.Unlock()

	// Delegate to state manager
	return c.executePlayback(url, voiceConn)
}

// Stop stops the current playback and cleans up resources
// Implements the AudioPipeline interface
func (c *AudioPipelineController) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateStopped {
		return nil // Already stopped
	}

	c.logger.Info("Stopping audio pipeline", CreateContextFields("", "", c.currentURL))

	// Signal stop to all goroutines
	close(c.stopChan)
	c.cancelFunc()

	// Stop components in reverse order
	var stopErrors []error

	// Stop stream processor
	if c.streamProcessor != nil {
		if err := c.streamProcessor.Stop(); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("stream processor stop failed: %w", err))
		}
	}

	// Close audio encoder
	if c.audioEncoder != nil {
		if err := c.audioEncoder.Close(); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("audio encoder close failed: %w", err))
		}
	}

	// Update state
	c.state = StateStopped
	c.currentURL = ""
	c.startTime = time.Time{}
	c.voiceConn = nil

	// Create new context for next playback
	c.ctx, c.cancelFunc = context.WithCancel(context.Background())
	c.stopChan = make(chan struct{})

	// Log any stop errors but don't fail the stop operation
	if len(stopErrors) > 0 {
		for _, err := range stopErrors {
			c.logger.Error("Error during stop", err, CreateContextFields("", "", ""))
		}
	}

	c.logger.Info("Audio pipeline stopped successfully", CreateContextFields("", "", ""))
	return nil
}

// IsPlaying returns true if the pipeline is currently playing audio
// Implements the AudioPipeline interface
func (c *AudioPipelineController) IsPlaying() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == StatePlaying
}

// GetStatus returns the current status of the pipeline
// Implements the AudioPipeline interface
func (c *AudioPipelineController) GetStatus() PipelineStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var lastErrorStr string
	if c.lastError != nil {
		lastErrorStr = c.lastError.Error()
	}

	return PipelineStatus{
		IsPlaying:  c.state == StatePlaying,
		CurrentURL: c.currentURL,
		StartTime:  c.startTime,
		ErrorCount: c.errorCount,
		LastError:  lastErrorStr,
	}
}

// executePlayback handles the actual playback execution
func (c *AudioPipelineController) executePlayback(url string, voiceConn *discordgo.VoiceConnection) error {
	c.mu.Lock()
	c.state = StateStarting
	c.currentURL = url
	c.voiceConn = voiceConn
	c.startTime = time.Now()
	c.lastError = nil
	c.mu.Unlock()

	c.logger.Info("Starting playback", CreateContextFields("", "", url))

	// Record startup time
	startTime := time.Now()

	// Initialize audio encoder
	if err := c.audioEncoder.Initialize(); err != nil {
		return c.handlePlaybackError(err, "encoder_init")
	}

	// Start the stream processor
	stream, err := c.streamProcessor.StartStream(url)
	if err != nil {
		return c.handlePlaybackError(err, "stream_start")
	}

	// Record successful startup
	c.metrics.RecordStartupTime(time.Since(startTime))

	// Update state to playing
	c.mu.Lock()
	c.state = StatePlaying
	c.mu.Unlock()

	c.logger.Info("Playback started successfully", CreateContextFields("", "", url))

	// Start streaming audio in a separate goroutine
	go c.streamAudio(stream)

	return nil
}

// streamAudio handles the audio streaming loop
func (c *AudioPipelineController) streamAudio(stream io.ReadCloser) {
	defer stream.Close()

	c.logger.Debug("Starting audio streaming loop", CreateContextFields("", "", c.currentURL))

	// Get frame size from encoder
	frameSize := c.audioEncoder.GetFrameSize()
	pcmBuffer := make([]int16, frameSize)
	byteBuffer := make([]byte, frameSize*2) // 2 bytes per int16

	for {
		select {
		case <-c.stopChan:
			c.logger.Debug("Stop signal received, ending stream", CreateContextFields("", "", c.currentURL))
			return
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled, ending stream", CreateContextFields("", "", c.currentURL))
			return
		default:
			// Read PCM data from stream
			n, err := stream.Read(byteBuffer)
			if err != nil {
				if err == io.EOF {
					c.logger.Info("Stream ended normally", CreateContextFields("", "", c.currentURL))
					c.handleStreamEnd()
					return
				}
				c.logger.Error("Stream read error", err, CreateContextFields("", "", c.currentURL))
				c.handlePlaybackError(err, "stream_read")
				return
			}

			if n == 0 {
				continue
			}

			// Convert bytes to int16 PCM samples
			samplesRead := n / 2
			for i := 0; i < samplesRead; i++ {
				pcmBuffer[i] = int16(byteBuffer[i*2]) | int16(byteBuffer[i*2+1])<<8
			}

			// Encode to Opus
			opusData, err := c.audioEncoder.Encode(pcmBuffer[:samplesRead])
			if err != nil {
				c.logger.Error("Encoding error", err, CreateContextFields("", "", c.currentURL))
				c.handlePlaybackError(err, "encoding")
				return
			}

			// Send to Discord
			if c.voiceConn != nil && c.voiceConn.OpusSend != nil {
				select {
				case c.voiceConn.OpusSend <- opusData:
					// Successfully sent
				case <-c.stopChan:
					return
				case <-c.ctx.Done():
					return
				default:
					// Channel is full, skip this frame
					c.logger.Debug("Discord send channel full, skipping frame", CreateContextFields("", "", c.currentURL))
				}
			}
		}
	}
}

// handlePlaybackError handles errors during playback with retry logic
func (c *AudioPipelineController) handlePlaybackError(err error, context string) error {
	c.mu.Lock()
	c.lastError = err
	c.errorCount++
	errorCount := c.errorCount
	url := c.currentURL
	c.state = StateError
	c.mu.Unlock()

	c.logger.Error("Playback error occurred", err, CreateContextFieldsWithComponent("", "", url, context))

	// Record error in metrics
	c.metrics.RecordError(context)

	// Use error handler to determine if we should retry
	shouldRetry, delay := c.errorHandler.HandleError(err, context)

	if shouldRetry && errorCount <= c.config.GetRetryConfig().MaxRetries {
		c.logger.Info("Retrying playback after error", CreateContextFields("", "", url))

		// Wait for retry delay
		select {
		case <-time.After(delay):
			// Continue with retry
		case <-c.stopChan:
			return fmt.Errorf("stop requested during retry delay")
		case <-c.ctx.Done():
			return fmt.Errorf("context cancelled during retry delay")
		}

		// Reset state for retry
		c.mu.Lock()
		c.state = StateStarting
		c.mu.Unlock()

		// Retry the playback
		return c.executePlayback(url, c.voiceConn)
	}

	// Max retries exceeded or non-retryable error
	c.logger.Error("Playback failed permanently", err, CreateContextFields("", "", url))
	c.Stop() // Clean up resources
	return fmt.Errorf("playback failed after %d attempts: %w", errorCount, err)
}

// handleStreamEnd handles normal stream completion
func (c *AudioPipelineController) handleStreamEnd() {
	c.mu.Lock()
	playTime := time.Since(c.startTime)
	url := c.currentURL
	c.mu.Unlock()

	c.logger.Info("Stream completed successfully", CreateContextFieldsWithComponent("", "", url, "stream_end"))

	// Record playback duration
	c.metrics.RecordPlaybackDuration(playTime)

	// Clean stop
	c.Stop()
}