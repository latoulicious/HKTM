package common

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

// AudioPipeline manages the entire audio streaming pipeline
type AudioPipeline struct {
	ctx         context.Context
	cancel      context.CancelFunc
	voiceConn   *discordgo.VoiceConnection
	ffmpegCmd   *exec.Cmd
	opusEncoder *gopus.Encoder
	isPlaying   bool
	mu          sync.RWMutex

	// Health monitoring
	lastFrameTime time.Time
	healthTicker  *time.Ticker

	// Error handling
	errorChan    chan error
	restartChan  chan struct{}
	maxRestarts  int
	restartCount int
}

// NewAudioPipeline creates a new audio pipeline
func NewAudioPipeline(vc *discordgo.VoiceConnection) *AudioPipeline {
	ctx, cancel := context.WithCancel(context.Background())

	return &AudioPipeline{
		ctx:           ctx,
		cancel:        cancel,
		voiceConn:     vc,
		maxRestarts:   3,
		errorChan:     make(chan error, 10),
		restartChan:   make(chan struct{}, 1),
		lastFrameTime: time.Now(),
	}
}

// PlayStream starts streaming audio from the given URL
func (ap *AudioPipeline) PlayStream(streamURL string) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.isPlaying {
		return fmt.Errorf("pipeline is already playing")
	}

	// Initialize Opus encoder
	encoder, err := gopus.NewEncoder(48000, 2, gopus.Audio)
	if err != nil {
		return fmt.Errorf("failed to create opus encoder: %v", err)
	}
	encoder.SetBitrate(128000) // Higher bitrate for better quality
	ap.opusEncoder = encoder

	ap.isPlaying = true

	// Start the main streaming goroutine
	go ap.streamLoop(streamURL)

	// Start health monitoring in a separate goroutine
	go ap.startHealthMonitoring()

	// Start error handler
	go ap.errorHandler(streamURL)

	return nil
}

// PlayStreamWithOriginalURL starts streaming with URL refresh capability
// This is useful for YouTube URLs that may expire
func (ap *AudioPipeline) PlayStreamWithOriginalURL(streamURL, originalURL string) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.isPlaying {
		return fmt.Errorf("pipeline is already playing")
	}

	// Initialize Opus encoder
	encoder, err := gopus.NewEncoder(48000, 2, gopus.Audio)
	if err != nil {
		return fmt.Errorf("failed to create opus encoder: %v", err)
	}
	encoder.SetBitrate(128000) // Higher bitrate for better quality
	ap.opusEncoder = encoder

	ap.isPlaying = true

	// Start the main streaming goroutine with URL refresh capability
	go ap.streamLoopWithURLRefresh(streamURL, originalURL)

	// Start health monitoring in a separate goroutine
	go ap.startHealthMonitoring()

	// Start error handler
	go ap.errorHandler(streamURL)

	return nil
}

// streamLoop is the main audio streaming loop with restart capability
func (ap *AudioPipeline) streamLoop(streamURL string) {
	defer func() {
		ap.mu.Lock()
		ap.isPlaying = false
		ap.mu.Unlock()
	}()

	// Add a restart mutex to prevent multiple simultaneous restarts
	var restartMutex sync.Mutex

	for {
		select {
		case <-ap.ctx.Done():
			log.Println("Audio pipeline context cancelled")
			return
		case <-ap.restartChan:
			restartMutex.Lock()
			if ap.restartCount >= ap.maxRestarts {
				log.Printf("Max restart attempts (%d) reached, stopping", ap.maxRestarts)
				ap.errorChan <- fmt.Errorf("max restarts exceeded")
				restartMutex.Unlock()
				return
			}
			ap.restartCount++
			log.Printf("Restarting audio pipeline (attempt %d/%d)", ap.restartCount, ap.maxRestarts)
			time.Sleep(2 * time.Second) // Brief delay before restart
			restartMutex.Unlock()
		}

		err := ap.streamAudio(streamURL)
		if err != nil {
			log.Printf("Stream error: %v", err)

			// Check if this is a normal completion error or network timeout
			errStr := err.Error()
			if strings.Contains(errStr, "stream ended normally") ||
				strings.Contains(errStr, "ffmpeg process exited") ||
				strings.Contains(errStr, "connection timed out") ||
				strings.Contains(errStr, "connection refused") ||
				strings.Contains(errStr, "invalid stream url") ||
				strings.Contains(errStr, "ffmpeg failed to start") {
				log.Println("Stream ended due to normal completion or network issue, not treating as error")
				return
			}

			ap.errorChan <- err

			// Check if we should restart
			if ap.shouldRestart(err) {
				restartMutex.Lock()
				select {
				case ap.restartChan <- struct{}{}:
				default:
					// If channel is full, don't send another restart signal
				}
				restartMutex.Unlock()
				continue
			}
			return
		}

		// Normal completion
		log.Println("Audio stream completed normally")
		return
	}
}

// streamAudio handles the actual audio streaming
func (ap *AudioPipeline) streamAudio(streamURL string) error {
	// Create FFmpeg command with improved buffering and performance
	cmd := exec.CommandContext(ap.ctx, "ffmpeg",
		"-reconnect", "1",
		"-reconnect_streamed", "1",
		"-reconnect_delay_max", "5",
		"-reconnect_at_eof", "1",
		"-reconnect_on_network_error", "1",
		"-timeout", "30", // Add timeout for initial connection
		"-rw_timeout", "30000000", // 30 second read/write timeout
		"-i", streamURL,
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-ar", "48000",
		"-ac", "2",
		// Improved buffering for smoother streaming
		"-bufsize", "256k", // Increased buffer size
		"-max_muxing_queue_size", "2048", // Larger queue
		"-probesize", "128k", // Larger probe size
		"-analyzeduration", "10000000", // 10 seconds analysis
		// Reduce output noise
		"-hide_banner",
		"-loglevel", "error",
		"-")

	ap.ffmpegCmd = cmd

	// Capture stderr for debugging
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start stderr consumer to prevent blocking
	go ap.consumeStderr(stderrPipe)

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	// Start FFmpeg
	log.Println("Starting FFmpeg process...")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	// Wait a moment for FFmpeg to start and check if it's still running
	time.Sleep(1 * time.Second)
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		exitCode := cmd.ProcessState.ExitCode()
		if exitCode == 1 {
			return fmt.Errorf("ffmpeg failed to start - likely invalid stream URL or network issue")
		}
		return fmt.Errorf("ffmpeg process exited immediately after start with code %d", exitCode)
	}

	// Ensure process cleanup
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cmd.Wait()

		// Check exit status for network-related errors
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			if cmd.ProcessState.ExitCode() != 0 {
				log.Printf("FFmpeg exited with code %d", cmd.ProcessState.ExitCode())
			}
		}
	}()

	// Wait for voice connection readiness
	if err := ap.waitForVoiceReady(); err != nil {
		return err
	}

	// Start speaking
	ap.voiceConn.Speaking(true)
	defer ap.voiceConn.Speaking(false)

	// Initialize frame time to now to prevent premature health check failures
	ap.lastFrameTime = time.Now()

	log.Println("Starting audio stream to Discord...")

	// Stream audio with proper buffering and error handling
	return ap.streamPCMToDiscord(stdout)
}

// Audio frame constants
const (
	audioFrameSize = 1920 // 960 samples * 2 channels for 20ms
	audioChunkSize = 3840 // 20ms of audio data in bytes
)

// streamPCMToDiscord handles the PCM to Opus conversion and Discord streaming
func (ap *AudioPipeline) streamPCMToDiscord(reader io.Reader) error {
	// Use larger buffer for better performance and reduced read frequency
	const bufferSize = audioChunkSize * 10 // 10 frames = 200ms of audio

	buffer := make([]byte, bufferSize)
	frameCount := 0

	// Create a buffered reader to handle variable data rates
	bufReader := io.Reader(reader)

	for {
		select {
		case <-ap.ctx.Done():
			return nil
		default:
		}

		// Read PCM data with improved buffering strategy
		readDone := make(chan int, 1)
		readErr := make(chan error, 1)

		go func() {
			// Try to read full buffer first, fall back to partial reads
			n, err := bufReader.Read(buffer)
			if err != nil {
				readErr <- err
				return
			}
			readDone <- n
		}()

		var n int
		var err error

		select {
		case n = <-readDone:
			if n > 0 {
				// Process audio in 20ms chunks
				for offset := 0; offset < n; offset += audioChunkSize {
					end := offset + audioChunkSize
					if end > n {
						end = n
					}

					chunkSize := end - offset
					if chunkSize < audioChunkSize {
						// Pad the last chunk if it's smaller than frame size
						padded := make([]byte, audioChunkSize)
						copy(padded, buffer[offset:end])
						ap.processAudioChunk(padded[:audioChunkSize], &frameCount)
					} else {
						ap.processAudioChunk(buffer[offset:end], &frameCount)
					}
				}
			}
		case err = <-readErr:
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				log.Println("FFmpeg stream ended normally")
				return nil
			}
			// Check if FFmpeg process has exited
			if ap.ffmpegCmd != nil && ap.ffmpegCmd.Process != nil {
				if ap.ffmpegCmd.ProcessState != nil && ap.ffmpegCmd.ProcessState.Exited() {
					log.Println("FFmpeg process exited, stream ended")
					return nil
				}
			}
			return fmt.Errorf("error reading PCM data: %v", err)
		case <-time.After(15 * time.Second): // Increased timeout for larger buffers
			// Check if FFmpeg process is still running
			if ap.ffmpegCmd != nil && ap.ffmpegCmd.Process != nil {
				if ap.ffmpegCmd.ProcessState != nil && ap.ffmpegCmd.ProcessState.Exited() {
					log.Println("FFmpeg process exited during timeout")
					return nil
				}
			}
			return fmt.Errorf("timeout reading PCM data")
		}
	}
}

// processAudioChunk handles encoding and sending a single audio chunk to Discord
func (ap *AudioPipeline) processAudioChunk(chunk []byte, frameCount *int) {
	// Convert bytes to int16 samples
	samples := bytesToInt16(chunk)

	// Ensure we have exactly 960 samples per channel for 20ms frames
	if len(samples) != audioFrameSize {
		// Pad or truncate to correct size
		if len(samples) < audioFrameSize {
			padded := make([]int16, audioFrameSize)
			copy(padded, samples)
			samples = padded
		} else {
			samples = samples[:audioFrameSize]
		}
	}

	// Encode to Opus
	opusData, err := ap.opusEncoder.Encode(samples, 960, len(chunk))
	if err != nil {
		log.Printf("Opus encoding error: %v", err)
		return
	}

	// Send to Discord with non-blocking send
	select {
	case ap.voiceConn.OpusSend <- opusData:
		*frameCount++
		ap.lastFrameTime = time.Now()

		// Log progress every 100 frames (2 seconds)
		if *frameCount%100 == 0 {
			log.Printf("Streamed %d frames (%.1fs of audio)", *frameCount, float64(*frameCount)*0.02)
		}
	case <-time.After(100 * time.Millisecond):
		log.Println("Warning: OpusSend channel blocked, skipping frame")
	}
}

// Error handling
func (ap *AudioPipeline) errorHandler(_ string) {
	for {
		select {
		case <-ap.ctx.Done():
			return
		case err := <-ap.errorChan:
			log.Printf("Pipeline error: %v", err)

			if ap.shouldRestart(err) {
				log.Println("Attempting to restart pipeline...")
				select {
				case ap.restartChan <- struct{}{}:
				default:
				}
			} else {
				log.Println("Error is not recoverable, stopping pipeline")
				ap.Stop()
				return
			}
		}
	}
}

// shouldRestart determines if an error is recoverable
func (ap *AudioPipeline) shouldRestart(err error) bool {
	if ap.restartCount >= ap.maxRestarts {
		return false
	}

	// Add logic to determine which errors are recoverable
	errStr := err.Error()
	recoverableErrors := []string{
		"stream health check failed",
		"timeout reading PCM data",
		"voice connection health check failed",
		"error reading PCM data",
	}

	// Don't restart for certain errors that indicate normal completion or network issues
	nonRecoverableErrors := []string{
		"ffmpeg stream ended normally",
		"ffmpeg process exited",
		"stream completed normally",
		"connection timed out",
		"connection refused",
		"network is unreachable",
		"no route to host",
		"temporary failure in name resolution",
		"invalid stream url",
		"ffmpeg failed to start",
	}

	for _, nonRecoverable := range nonRecoverableErrors {
		if contains(errStr, nonRecoverable) {
			return false
		}
	}

	for _, recoverable := range recoverableErrors {
		if contains(errStr, recoverable) {
			return true
		}
	}

	return false
}

// Utility functions
func (ap *AudioPipeline) waitForVoiceReady() error {
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for voice connection")
		case <-ticker.C:
			if ap.voiceConn.Ready {
				return nil
			}
		}
	}
}

func (ap *AudioPipeline) consumeStderr(stderr io.ReadCloser) {
	defer stderr.Close()
	buffer := make([]byte, 1024)
	for {
		n, err := stderr.Read(buffer)
		if err != nil {
			return
		}
		if n > 0 {
			// Log FFmpeg stderr for debugging, but only if it contains errors
			output := string(buffer[:n])
			if strings.Contains(strings.ToLower(output), "error") ||
				strings.Contains(strings.ToLower(output), "failed") ||
				strings.Contains(strings.ToLower(output), "timeout") {
				log.Printf("FFmpeg stderr: %s", output)

				// If we detect a network timeout, mark it as a network issue
				if strings.Contains(strings.ToLower(output), "connection timed out") ||
					strings.Contains(strings.ToLower(output), "connection refused") {
					log.Println("Detected network timeout in FFmpeg output")
				}
			}
		}
	}
}

// Stop gracefully stops the audio pipeline
func (ap *AudioPipeline) Stop() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	log.Println("Stopping audio pipeline...")
	ap.cancel()

	if ap.ffmpegCmd != nil && ap.ffmpegCmd.Process != nil {
		ap.ffmpegCmd.Process.Kill()
	}

	if ap.voiceConn != nil {
		ap.voiceConn.Speaking(false)
	}

	ap.isPlaying = false
}

// IsPlaying returns whether the pipeline is currently playing
func (ap *AudioPipeline) IsPlaying() bool {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.isPlaying
}

// Helper functions
func bytesToInt16(data []byte) []int16 {
	samples := make([]int16, len(data)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(data[i*2]) | int16(data[i*2+1])<<8
	}
	return samples
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// startHealthMonitoring starts health monitoring for the audio pipeline
func (ap *AudioPipeline) startHealthMonitoring() {
	// Start health monitoring after a delay to allow stream to establish
	time.Sleep(3 * time.Second)

	// Check if we're still playing before starting health monitoring
	ap.mu.RLock()
	if !ap.isPlaying {
		ap.mu.RUnlock()
		return
	}
	ap.mu.RUnlock()

	ap.healthTicker = time.NewTicker(5 * time.Second)
	go func() {
		defer func() {
			if ap.healthTicker != nil {
				ap.healthTicker.Stop()
			}
		}()
		for {
			select {
			case <-ap.ctx.Done():
				return
			case <-ap.healthTicker.C:
				// Check if we're still playing before running health check
				ap.mu.RLock()
				if !ap.isPlaying {
					ap.mu.RUnlock()
					return
				}
				ap.mu.RUnlock()
				ap.checkHealth()
			}
		}
	}()
}

// checkHealth performs health checks on the audio pipeline
func (ap *AudioPipeline) checkHealth() {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	if !ap.isPlaying {
		return
	}

	// Check if we haven't received frames in a while
	// Only trigger health check if we've been playing for more than 5 seconds
	// and haven't received frames in the last 15 seconds
	if time.Since(ap.lastFrameTime) > 15*time.Second {
		// Additional check: only trigger if FFmpeg process is still running
		if ap.ffmpegCmd != nil && ap.ffmpegCmd.Process != nil {
			// Check if process is still alive
			if ap.ffmpegCmd.ProcessState != nil && ap.ffmpegCmd.ProcessState.Exited() {
				log.Println("Health check: FFmpeg process has exited")
				return
			}
		}

		// Check if we've been playing for at least 20 seconds before triggering health check
		// This prevents premature health check failures during network issues
		if time.Since(ap.lastFrameTime) > 20*time.Second {
			log.Println("Health check failed: no frames received in 20 seconds")
			ap.errorChan <- fmt.Errorf("stream health check failed: no recent frames")
		}
	}

	// Check voice connection state
	if ap.voiceConn == nil || !ap.voiceConn.Ready {
		log.Println("Health check failed: voice connection not ready")
		ap.errorChan <- fmt.Errorf("voice connection health check failed")
	}
}

// streamLoopWithURLRefresh is the main audio streaming loop with URL refresh capability
func (ap *AudioPipeline) streamLoopWithURLRefresh(streamURL, originalURL string) {
	defer func() {
		ap.mu.Lock()
		ap.isPlaying = false
		ap.mu.Unlock()
	}()

	// Add a restart mutex to prevent multiple simultaneous restarts
	var restartMutex sync.Mutex

	for {
		select {
		case <-ap.ctx.Done():
			log.Println("Audio pipeline context cancelled")
			return
		case <-ap.restartChan:
			restartMutex.Lock()
			if ap.restartCount >= ap.maxRestarts {
				log.Printf("Max restart attempts (%d) reached, stopping", ap.maxRestarts)
				ap.errorChan <- fmt.Errorf("max restarts exceeded")
				restartMutex.Unlock()
				return
			}
			ap.restartCount++
			log.Printf("Restarting audio pipeline (attempt %d/%d)", ap.restartCount, ap.maxRestarts)
			time.Sleep(2 * time.Second) // Brief delay before restart
			restartMutex.Unlock()
		}

		// Try to get a fresh URL if we have an original URL (YouTube)
		currentURL := streamURL
		if originalURL != "" {
			log.Println("Getting fresh stream URL to avoid expiration...")
			freshURL, err := GetFreshYouTubeStreamURL(originalURL)
			if err != nil {
				log.Printf("Failed to get fresh URL, using original: %v", err)
			} else {
				currentURL = freshURL
				log.Println("Successfully got fresh stream URL")
			}
		}

		err := ap.streamAudio(currentURL)
		if err != nil {
			log.Printf("Stream error: %v", err)

			// Check if this is a normal completion error or network timeout
			errStr := err.Error()
			if strings.Contains(errStr, "stream ended normally") ||
				strings.Contains(errStr, "ffmpeg process exited") ||
				strings.Contains(errStr, "connection timed out") ||
				strings.Contains(errStr, "connection refused") ||
				strings.Contains(errStr, "invalid stream url") ||
				strings.Contains(errStr, "ffmpeg failed to start") {
				log.Println("Stream ended due to normal completion or network issue, not treating as error")
				return
			}

			ap.errorChan <- err

			// Check if we should restart
			if ap.shouldRestart(err) {
				restartMutex.Lock()
				select {
				case ap.restartChan <- struct{}{}:
				default:
					// If channel is full, don't send another restart signal
				}
				restartMutex.Unlock()
				continue
			}
			return
		}

		// Normal completion
		log.Println("Audio stream completed normally")
		return
	}
}
