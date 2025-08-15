package audio

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// FFmpegProcessor implements the StreamProcessor interface for FFmpeg operations
type FFmpegProcessor struct {
	config     *FFmpegConfig
	cmd        *exec.Cmd
	outputPipe io.ReadCloser
	errorPipe  io.ReadCloser
	isRunning  bool
	currentURL string
	mu         sync.RWMutex
	logger     AudioLogger
}

// NewFFmpegProcessor creates a new FFmpegProcessor instance
func NewFFmpegProcessor(config *FFmpegConfig, logger AudioLogger) StreamProcessor {
	return &FFmpegProcessor{
		config: config,
		logger: logger,
	}
}

// StartStream starts the yt-dlp + FFmpeg pipeline for the given URL
func (fp *FFmpegProcessor) StartStream(url string) (io.ReadCloser, error) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// Stop any existing stream first
	if fp.isRunning {
		if err := fp.stopInternal(); err != nil {
			fp.logger.Warn("Failed to stop existing stream before starting new one", map[string]interface{}{
				"error": err.Error(),
				"url":   url,
			})
		}
	}

	fp.currentURL = url

	// First, get the direct stream URL using yt-dlp
	streamURL, err := fp.getStreamURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream URL: %w", err)
	}

	fp.logger.Debug("Got stream URL from yt-dlp", map[string]interface{}{
		"original_url": url,
		"stream_url":   streamURL,
	})

	// Build the FFmpeg command arguments with the direct stream URL
	args := fp.buildFFmpegArgsWithStreamURL(streamURL)

	fp.logger.Debug("Starting FFmpeg process", map[string]interface{}{
		"url":  url,
		"args": args,
	})

	// Create the command
	fp.cmd = exec.Command(fp.config.BinaryPath, args...)

	// Set up process group for proper cleanup
	fp.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Create pipes for stdout (audio data) and stderr (error output)
	stdout, err := fp.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	fp.outputPipe = stdout

	stderr, err := fp.cmd.StderrPipe()
	if err != nil {
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	fp.errorPipe = stderr

	// Start the command
	if err := fp.cmd.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start FFmpeg process: %w", err)
	}

	fp.isRunning = true

	// Start monitoring stderr in a separate goroutine
	go fp.monitorStderr()

	// Start monitoring the process in a separate goroutine
	go fp.monitorProcess()

	fp.logger.Info("FFmpeg stream started successfully", map[string]interface{}{
		"url": url,
		"pid": fp.cmd.Process.Pid,
	})

	return stdout, nil
}

// Stop stops the current FFmpeg process
func (fp *FFmpegProcessor) Stop() error {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	return fp.stopInternal()
}

// stopInternal stops the process without acquiring the lock (internal use)
func (fp *FFmpegProcessor) stopInternal() error {
	if !fp.isRunning || fp.cmd == nil {
		return nil
	}

	fp.logger.Debug("Stopping FFmpeg process", map[string]interface{}{
		"pid": fp.cmd.Process.Pid,
		"url": fp.currentURL,
	})

	// Close pipes first
	if fp.outputPipe != nil {
		fp.outputPipe.Close()
		fp.outputPipe = nil
	}
	if fp.errorPipe != nil {
		fp.errorPipe.Close()
		fp.errorPipe = nil
	}

	// Try graceful termination first
	if fp.cmd.Process != nil {
		// Send SIGTERM to the process group
		if err := syscall.Kill(-fp.cmd.Process.Pid, syscall.SIGTERM); err != nil {
			fp.logger.Warn("Failed to send SIGTERM to process group", map[string]interface{}{
				"error": err.Error(),
				"pid":   fp.cmd.Process.Pid,
			})
		}

		// Wait for graceful shutdown with timeout
		done := make(chan error, 1)
		go func() {
			done <- fp.cmd.Wait()
		}()

		select {
		case <-done:
			// Process terminated gracefully
		case <-time.After(5 * time.Second):
			// Force kill if graceful shutdown takes too long
			fp.logger.Warn("FFmpeg process did not terminate gracefully, force killing", map[string]interface{}{
				"pid": fp.cmd.Process.Pid,
			})
			if err := syscall.Kill(-fp.cmd.Process.Pid, syscall.SIGKILL); err != nil {
				fp.logger.Error("Failed to force kill process group", err, map[string]interface{}{
					"pid": fp.cmd.Process.Pid,
				})
			}
			<-done // Wait for the process to actually exit
		}
	}

	fp.isRunning = false
	fp.cmd = nil
	fp.currentURL = ""

	fp.logger.Info("FFmpeg process stopped", nil)
	return nil
}

// IsRunning returns whether the FFmpeg process is currently running
func (fp *FFmpegProcessor) IsRunning() bool {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	return fp.isRunning
}

// Restart stops the current stream and starts a new one with the given URL
func (fp *FFmpegProcessor) Restart(url string) error {
	fp.logger.Info("Restarting FFmpeg stream", map[string]interface{}{
		"old_url": fp.currentURL,
		"new_url": url,
	})

	// Stop current stream
	if err := fp.Stop(); err != nil {
		fp.logger.Error("Failed to stop current stream during restart", err, map[string]interface{}{
			"url": url,
		})
		return fmt.Errorf("failed to stop current stream: %w", err)
	}

	// Start new stream
	_, err := fp.StartStream(url)
	return err
}

// getStreamURL uses yt-dlp to extract the direct stream URL
func (fp *FFmpegProcessor) getStreamURL(url string) (string, error) {
	fp.logger.Debug("Getting stream URL with yt-dlp", map[string]interface{}{
		"url": url,
	})

	// Run yt-dlp to get the direct stream URL
	cmd := exec.Command("yt-dlp", "-f", "bestaudio", "--get-url", url)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp failed to get stream URL: %w", err)
	}

	streamURL := strings.TrimSpace(string(output))
	if streamURL == "" {
		return "", fmt.Errorf("yt-dlp returned empty stream URL")
	}

	return streamURL, nil
}

// buildFFmpegArgsWithStreamURL constructs the FFmpeg command arguments with a direct stream URL
func (fp *FFmpegProcessor) buildFFmpegArgsWithStreamURL(streamURL string) []string {
	args := []string{
		"-i", streamURL,
		"-f", fp.config.AudioFormat,
		"-ar", fmt.Sprintf("%d", fp.config.SampleRate),
		"-ac", fmt.Sprintf("%d", fp.config.Channels),
	}

	// Add custom arguments from configuration
	args = append(args, fp.config.CustomArgs...)

	// Add output to stdout
	args = append(args, "pipe:1")

	return args
}

// monitorStderr monitors the stderr output from FFmpeg for errors and warnings
func (fp *FFmpegProcessor) monitorStderr() {
	if fp.errorPipe == nil {
		return
	}

	scanner := bufio.NewScanner(fp.errorPipe)
	for scanner.Scan() {
		line := scanner.Text()

		// Log FFmpeg output for debugging
		fp.logger.Debug("FFmpeg stderr", map[string]interface{}{
			"output": line,
			"url":    fp.currentURL,
		})

		// Check for specific error patterns
		if fp.isErrorLine(line) {
			fp.logger.Warn("FFmpeg error detected", map[string]interface{}{
				"error": line,
				"url":   fp.currentURL,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		fp.logger.Error("Error reading FFmpeg stderr", err, map[string]interface{}{
			"url": fp.currentURL,
		})
	}
}

// monitorProcess monitors the FFmpeg process and handles unexpected exits
func (fp *FFmpegProcessor) monitorProcess() {
	if fp.cmd == nil {
		return
	}

	// Wait for the process to exit
	err := fp.cmd.Wait()

	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.isRunning {
		// Process exited unexpectedly
		fp.isRunning = false

		if err != nil {
			fp.logger.Error("FFmpeg process exited unexpectedly", err, map[string]interface{}{
				"url":       fp.currentURL,
				"exit_code": fp.cmd.ProcessState.ExitCode(),
			})
		} else {
			fp.logger.Info("FFmpeg process completed", map[string]interface{}{
				"url": fp.currentURL,
			})
		}

		// Clean up
		fp.cmd = nil
		fp.currentURL = ""
		if fp.outputPipe != nil {
			fp.outputPipe.Close()
			fp.outputPipe = nil
		}
		if fp.errorPipe != nil {
			fp.errorPipe.Close()
			fp.errorPipe = nil
		}
	}
}

// isErrorLine checks if a stderr line indicates an error condition
func (fp *FFmpegProcessor) isErrorLine(line string) bool {
	errorPatterns := []string{
		"Error",
		"error",
		"Failed",
		"failed",
		"Cannot",
		"cannot",
		"Invalid",
		"invalid",
		"No such file",
		"Permission denied",
		"Connection refused",
		"Timeout",
		"timeout",
	}

	for _, pattern := range errorPatterns {
		if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
