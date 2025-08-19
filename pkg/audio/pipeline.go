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
	state         PipelineState
	currentURL    string
	voiceConn     *discordgo.VoiceConnection
	startTime     time.Time
	errorCount    int
	lastError     error
	stopChan      chan struct{}
	mu            sync.RWMutex
	ctx           context.Context
	cancelFunc    context.CancelFunc
	initialized   bool
	shutdownOnce  sync.Once
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
		initialized:     false,
	}
}

// Initialize initializes the audio pipeline and validates all dependencies
// Implements the AudioPipeline interface
func (c *AudioPipelineController) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil // Already initialized
	}

	c.logger.Info("Initializing audio pipeline", CreateContextFieldsWithComponent("", "", "", "initialization"))

	// Step 1: Validate configuration
	if err := c.config.Validate(); err != nil {
		c.logger.Error("Configuration validation failed", err, CreateContextFieldsWithComponent("", "", "", "config_validation"))
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Step 2: Validate binary dependencies
	if err := c.config.ValidateDependencies(); err != nil {
		c.logger.Error("Dependency validation failed", err, CreateContextFieldsWithComponent("", "", "", "dependency_validation"))
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	// Step 3: Initialize audio encoder
	if err := c.audioEncoder.Initialize(); err != nil {
		c.logger.Error("Audio encoder initialization failed", err, CreateContextFieldsWithComponent("", "", "", "encoder_init"))
		return fmt.Errorf("audio encoder initialization failed: %w", err)
	}

	c.initialized = true
	c.logger.Info("Audio pipeline initialized successfully", CreateContextFieldsWithComponent("", "", "", "initialization"))

	return nil
}

// Shutdown gracefully shuts down the audio pipeline and cleans up all resources
// Implements the AudioPipeline interface
func (c *AudioPipelineController) Shutdown() error {
	var shutdownErr error

	c.shutdownOnce.Do(func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		if !c.initialized {
			return // Not initialized, nothing to shutdown
		}

		c.logger.Info("Shutting down audio pipeline", CreateContextFieldsWithComponent("", "", "", "shutdown"))

		// Step 1: Stop any active playback
		if c.state == StatePlaying || c.state == StateStarting {
			c.logger.Debug("Stopping active playback during shutdown", CreateContextFieldsWithComponent("", "", "", "shutdown"))
			if err := c.stopPlaybackInternal(); err != nil {
				c.logger.Error("Error stopping playback during shutdown", err, CreateContextFieldsWithComponent("", "", "", "shutdown"))
				shutdownErr = fmt.Errorf("failed to stop playback during shutdown: %w", err)
			}
		}

		// Step 2: Cancel context and close channels
		c.cancelFunc()
		if c.stopChan != nil {
			select {
			case <-c.stopChan:
				// Already closed
			default:
				close(c.stopChan)
			}
		}

		// Step 3: Shutdown components in reverse initialization order
		var shutdownErrors []error

		// Close audio encoder
		if c.audioEncoder != nil {
			if err := c.audioEncoder.Close(); err != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("audio encoder shutdown failed: %w", err))
			}
		}

		// Stop stream processor
		if c.streamProcessor != nil {
			if err := c.streamProcessor.Stop(); err != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("stream processor shutdown failed: %w", err))
			}
		}

		// Log any shutdown errors
		if len(shutdownErrors) > 0 {
			for _, err := range shutdownErrors {
				c.logger.Error("Component shutdown error", err, CreateContextFieldsWithComponent("", "", "", "shutdown"))
			}
			if shutdownErr == nil {
				shutdownErr = fmt.Errorf("component shutdown errors occurred: %d errors", len(shutdownErrors))
			}
		}

		// Step 4: Reset state
		c.state = StateStopped
		c.currentURL = ""
		c.startTime = time.Time{}
		c.voiceConn = nil
		c.errorCount = 0
		c.lastError = nil
		c.initialized = false

		if shutdownErr == nil {
			c.logger.Info("Audio pipeline shutdown completed successfully", CreateContextFieldsWithComponent("", "", "", "shutdown"))
		} else {
			c.logger.Error("Audio pipeline shutdown completed with errors", shutdownErr, CreateContextFieldsWithComponent("", "", "", "shutdown"))
		}
	})

	return shutdownErr
}

// IsInitialized returns true if the pipeline has been successfully initialized
// Implements the AudioPipeline interface
func (c *AudioPipelineController) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// PlayURL starts playback of the given URL using the voice connection
// Implements the AudioPipeline interface
func (c *AudioPipelineController) PlayURL(url string, voiceConn *discordgo.VoiceConnection) error {
	// Check if initialized
	if !c.IsInitialized() {
		return fmt.Errorf("pipeline not initialized - call Initialize() first")
	}

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

	return c.stopPlaybackInternal()
}

// stopPlaybackInternal stops the current playback - must be called with mutex held
func (c *AudioPipelineController) stopPlaybackInternal() error {
	if c.state == StateStopped {
		return nil // Already stopped
	}

	c.logger.Info("Stopping audio pipeline", CreateContextFields("", "", c.currentURL))

	// Signal stop to all goroutines
	if c.stopChan != nil {
		select {
		case <-c.stopChan:
			// Already closed
		default:
			close(c.stopChan)
		}
	}
	c.cancelFunc()

	// Stop components in reverse order
	var stopErrors []error

	// Stop stream processor
	if c.streamProcessor != nil {
		if err := c.streamProcessor.Stop(); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("stream processor stop failed: %w", err))
		}
	}

	// Close audio encoder (but don't fully shutdown - just close current session)
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
// This method coordinates all components in the correct order:
// 1. Initialize state and logging context
// 2. Initialize audio encoder
// 3. Start stream processor
// 4. Record metrics
// 5. Start audio streaming loop
func (c *AudioPipelineController) executePlayback(url string, voiceConn *discordgo.VoiceConnection) error {
	// Set up initial state and context
	c.mu.Lock()
	c.state = StateStarting
	c.currentURL = url
	c.voiceConn = voiceConn
	c.startTime = time.Now()
	c.lastError = nil
	c.errorCount = 0 // Reset error count for new playback
	guildID := voiceConn.GuildID
	c.mu.Unlock()

	// Create enriched logging context for this playback session
	contextFields := CreateContextFieldsWithComponent(guildID, "", url, "pipeline")
	c.logger.Info("Starting playback execution", contextFields)

	// Record startup time measurement
	startTime := time.Now()

	// Step 1: Check if audio encoder needs initialization
	c.logger.Debug("Checking audio encoder status", contextFields)
	if !c.audioEncoder.IsInitialized() {
		c.logger.Debug("Initializing audio encoder", contextFields)
		if err := c.audioEncoder.Initialize(); err != nil {
			c.logger.Error("Audio encoder initialization failed", err, contextFields)
			return c.handlePlaybackError(err, "encoder_init")
		}
	} else {
		c.logger.Debug("Audio encoder already initialized, skipping", contextFields)
	}

	// Step 2: Prepare encoder for streaming
	c.logger.Debug("Preparing encoder for streaming", contextFields)
	if err := c.audioEncoder.PrepareForStreaming(); err != nil {
		c.logger.Error("Audio encoder streaming preparation failed", err, contextFields)
		return c.handlePlaybackError(err, "encoder_prepare")
	}

	// Step 3: Start the stream processor
	c.logger.Debug("Starting stream processor", contextFields)
	stream, err := c.streamProcessor.StartStream(url)
	if err != nil {
		c.logger.Error("Stream processor start failed", err, contextFields)
		return c.handlePlaybackError(err, "stream_start")
	}

	// Step 4: Record successful startup metrics
	startupDuration := time.Since(startTime)
	c.metrics.RecordStartupTime(startupDuration)
	
	contextFields["startup_duration"] = FormatDuration(startupDuration)
	c.logger.Info("Pipeline components initialized successfully", contextFields)

	// Step 5: Update state to playing
	c.mu.Lock()
	c.state = StatePlaying
	c.mu.Unlock()

	c.logger.Info("Playback started successfully, beginning audio stream", contextFields)

	// Step 6: Start streaming audio in a separate goroutine
	// This is non-blocking so the method can return immediately
	go c.streamAudio(stream)

	return nil
}

// streamAudio handles the audio streaming loop
// This method manages the continuous audio streaming process:
// 1. Set up streaming buffers and context
// 2. Continuously read PCM data from stream
// 3. Encode PCM data to Opus format
// 4. Send Opus frames to Discord voice connection
// 5. Handle streaming errors and cleanup
func (c *AudioPipelineController) streamAudio(stream io.ReadCloser) {
	defer func() {
		stream.Close()
		c.logger.Debug("Audio streaming loop ended", CreateContextFields("", "", c.currentURL))
	}()

	// Get current context for logging
	c.mu.RLock()
	url := c.currentURL
	guildID := ""
	if c.voiceConn != nil {
		guildID = c.voiceConn.GuildID
	}
	c.mu.RUnlock()

	contextFields := CreateContextFieldsWithComponent(guildID, "", url, "stream")
	c.logger.Debug("Starting audio streaming loop", contextFields)

	// Give FFmpeg a moment to initialize and start producing output
	// This prevents immediate "file already closed" errors
	time.Sleep(200 * time.Millisecond)

	// Get optimal frame size from encoder
	frameSize := c.audioEncoder.GetFrameSize()
	frameDuration := c.audioEncoder.GetFrameDuration()
	
	// Prepare buffers for audio processing
	pcmBuffer := make([]int16, frameSize)
	byteBuffer := make([]byte, frameSize*2) // 2 bytes per int16 sample
	
	contextFields["frame_size"] = frameSize
	contextFields["frame_duration"] = FormatDuration(frameDuration)
	c.logger.Debug("Audio streaming buffers initialized", contextFields)

	// Streaming statistics
	framesProcessed := 0
	bytesProcessed := 0
	streamStartTime := time.Now()

	// Main streaming loop
	for {
		select {
		case <-c.stopChan:
			c.logger.Debug("Stop signal received, ending stream", contextFields)
			return
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled, ending stream", contextFields)
			return
		default:
			// Read PCM data from stream processor
			n, err := stream.Read(byteBuffer)
			if err != nil {
				if err == io.EOF {
					// Normal stream completion
					streamDuration := time.Since(streamStartTime)
					endContextFields := CreateContextFieldsWithComponent(guildID, "", url, "stream_end")
					endContextFields["frames_processed"] = framesProcessed
					endContextFields["bytes_processed"] = bytesProcessed
					endContextFields["stream_duration"] = FormatDuration(streamDuration)
					
					c.logger.Info("Stream ended normally", endContextFields)
					c.handleStreamEnd()
					return
				}
				
				// Stream read error - provide extra context for first read failure
				errorContextFields := CreateContextFieldsWithComponent(guildID, "", url, "stream_read")
				errorContextFields["frames_processed"] = framesProcessed
				errorContextFields["bytes_processed"] = bytesProcessed
				errorContextFields["is_first_read"] = framesProcessed == 0 && bytesProcessed == 0
				errorContextFields["time_since_start"] = FormatDuration(time.Since(streamStartTime))
				
				// Check if FFmpeg process is still alive
				if c.streamProcessor != nil {
					errorContextFields["ffmpeg_running"] = c.streamProcessor.IsRunning()
					errorContextFields["ffmpeg_alive"] = c.streamProcessor.IsProcessAlive()
					errorContextFields["ffmpeg_info"] = c.streamProcessor.GetProcessInfo()
				}
				
				if framesProcessed == 0 && bytesProcessed == 0 {
					c.logger.Error("FFmpeg process appears to have exited immediately - check FFmpeg stderr for details", err, errorContextFields)
				} else {
					c.logger.Error("Stream read error", err, errorContextFields)
				}
				c.handlePlaybackError(err, "stream_read")
				return
			}

			// Skip empty reads
			if n == 0 {
				continue
			}

			bytesProcessed += n

			// Convert bytes to int16 PCM samples (little-endian)
			samplesRead := n / 2
			for i := 0; i < samplesRead; i++ {
				pcmBuffer[i] = int16(byteBuffer[i*2]) | int16(byteBuffer[i*2+1])<<8
			}

			// Validate frame size before encoding
			if err := c.audioEncoder.ValidateFrameSize(pcmBuffer[:samplesRead]); err != nil {
				c.logger.Warn("Invalid frame size, adjusting", CreateContextFieldsWithComponent(guildID, "", url, "frame_validation"))
				// Continue with available samples - encoder should handle partial frames
			}

			// Encode PCM data to Opus format
			opusData, err := c.audioEncoder.EncodeFrame(pcmBuffer[:samplesRead])
			if err != nil {
				errorContextFields := CreateContextFieldsWithComponent(guildID, "", url, "encoding")
				errorContextFields["samples_count"] = samplesRead
				errorContextFields["frame_number"] = framesProcessed
				
				c.logger.Error("Encoding error", err, errorContextFields)
				c.handlePlaybackError(err, "encoding")
				return
			}

			// Send encoded audio to Discord voice connection
			if c.voiceConn != nil && c.voiceConn.OpusSend != nil {
				select {
				case c.voiceConn.OpusSend <- opusData:
					// Successfully sent frame
					framesProcessed++
					
					// Log progress periodically (every 100 frames)
					if framesProcessed%100 == 0 {
						progressFields := CreateContextFieldsWithComponent(guildID, "", url, "stream_progress")
						progressFields["frames_processed"] = framesProcessed
						progressFields["bytes_processed"] = bytesProcessed
						progressFields["elapsed_time"] = FormatDuration(time.Since(streamStartTime))
						c.logger.Debug("Streaming progress", progressFields)
					}
					
				case <-c.stopChan:
					c.logger.Debug("Stop signal received while sending frame", contextFields)
					return
				case <-c.ctx.Done():
					c.logger.Debug("Context cancelled while sending frame", contextFields)
					return
				default:
					// Discord send channel is full - this indicates potential issues
					// Skip this frame but log the occurrence
					c.logger.Warn("Discord send channel full, skipping frame", CreateContextFieldsWithComponent(guildID, "", url, "discord_send"))
				}
			} else {
				// Voice connection is not available
				c.logger.Error("Voice connection unavailable", nil, CreateContextFieldsWithComponent(guildID, "", url, "voice_connection"))
				c.handlePlaybackError(fmt.Errorf("voice connection lost"), "voice_connection")
				return
			}
		}
	}
}

// handlePlaybackError handles errors during playback with retry logic
// This method implements comprehensive error recovery:
// 1. Update pipeline state and record error details
// 2. Log error with full context for debugging
// 3. Record error metrics for monitoring
// 4. Determine retry strategy using error handler
// 5. Execute retry with proper delays and limits
// 6. Handle permanent failures with cleanup
func (c *AudioPipelineController) handlePlaybackError(err error, context string) error {
	// Capture current state for error handling
	c.mu.Lock()
	c.lastError = err
	c.errorCount++
	errorCount := c.errorCount
	url := c.currentURL
	guildID := ""
	if c.voiceConn != nil {
		guildID = c.voiceConn.GuildID
	}
	c.state = StateError
	c.mu.Unlock()

	// Create comprehensive error context for logging
	errorContextFields := CreateContextFieldsWithComponent(guildID, "", url, context)
	errorContextFields["error_count"] = errorCount
	errorContextFields["error_type"] = fmt.Sprintf("%T", err)
	errorContextFields["error_message"] = err.Error()

	c.logger.Error("Playback error occurred", err, errorContextFields)

	// Record error in metrics system
	c.metrics.RecordError(context)

	// Use error handler to determine retry strategy
	shouldRetry, delay := c.errorHandler.HandleError(err, context)
	maxRetries := c.config.GetRetryConfig().MaxRetries

	// Log retry decision
	retryContextFields := CreateContextFieldsWithComponent(guildID, "", url, "retry_decision")
	retryContextFields["should_retry"] = shouldRetry
	retryContextFields["retry_delay"] = FormatDuration(delay)
	retryContextFields["attempt"] = errorCount
	retryContextFields["max_retries"] = maxRetries

	if shouldRetry && errorCount <= maxRetries {
		c.logger.Info("Attempting retry after error", retryContextFields)

		// Notify about retry attempt (if notifier is configured)
		c.errorHandler.NotifyRetryAttempt(errorCount, err, delay)

		// Wait for retry delay with cancellation support
		retryTimer := time.NewTimer(delay)
		defer retryTimer.Stop()

		select {
		case <-retryTimer.C:
			// Delay completed, proceed with retry
			c.logger.Debug("Retry delay completed, attempting recovery", retryContextFields)
		case <-c.stopChan:
			c.logger.Info("Stop requested during retry delay", retryContextFields)
			return fmt.Errorf("stop requested during retry delay for error: %w", err)
		case <-c.ctx.Done():
			c.logger.Info("Context cancelled during retry delay", retryContextFields)
			return fmt.Errorf("context cancelled during retry delay for error: %w", err)
		}

		// Clean up current state before retry
		c.cleanupForRetry()

		// Reset state for retry attempt
		c.mu.Lock()
		c.state = StateStarting
		c.mu.Unlock()

		// Log retry attempt
		retryAttemptFields := CreateContextFieldsWithComponent(guildID, "", url, "retry_attempt")
		retryAttemptFields["attempt"] = errorCount
		c.logger.Info("Executing retry attempt", retryAttemptFields)

		// Retry the playback execution
		return c.executePlayback(url, c.voiceConn)
	}

	// Max retries exceeded or non-retryable error - permanent failure
	failureContextFields := CreateContextFieldsWithComponent(guildID, "", url, "permanent_failure")
	failureContextFields["final_error_count"] = errorCount
	failureContextFields["max_retries"] = maxRetries
	failureContextFields["retry_exhausted"] = errorCount > maxRetries
	failureContextFields["non_retryable"] = !shouldRetry

	c.logger.Error("Playback failed permanently", err, failureContextFields)

	// Notify about permanent failure
	c.errorHandler.NotifyMaxRetriesExceeded(err, errorCount)

	// Clean up all resources
	c.Stop()

	// Return comprehensive error information
	if errorCount > maxRetries {
		return fmt.Errorf("playback failed after %d attempts (max %d): %w", errorCount, maxRetries, err)
	} else {
		return fmt.Errorf("playback failed with non-retryable error: %w", err)
	}
}

// cleanupForRetry performs cleanup operations before attempting a retry
func (c *AudioPipelineController) cleanupForRetry() {
	c.logger.Debug("Performing cleanup before retry", CreateContextFields("", "", c.currentURL))

	// Stop stream processor if running
	if c.streamProcessor != nil && c.streamProcessor.IsRunning() {
		if err := c.streamProcessor.Stop(); err != nil {
			c.logger.Warn("Error stopping stream processor during cleanup", CreateContextFields("", "", c.currentURL))
		}
	}

	// Close audio encoder (but don't reinitialize - let executePlayback handle that)
	if c.audioEncoder != nil && c.audioEncoder.IsInitialized() {
		if err := c.audioEncoder.Close(); err != nil {
			c.logger.Warn("Error closing audio encoder during cleanup", CreateContextFields("", "", c.currentURL))
		}
	}
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