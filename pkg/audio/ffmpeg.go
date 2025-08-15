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
	config         *FFmpegConfig
	cmd            *exec.Cmd
	outputPipe     io.ReadCloser
	errorPipe      io.ReadCloser
	isRunning      bool
	currentURL     string
	mu             sync.RWMutex
	logger         AudioLogger
	processExited  chan struct{} // Channel to signal when process has exited
	stderrBuffer   []string      // Buffer to store recent stderr lines for debugging
	maxStderrLines int           // Maximum number of stderr lines to keep
}

// NewFFmpegProcessor creates a new FFmpegProcessor instance
func NewFFmpegProcessor(config *FFmpegConfig, logger AudioLogger) StreamProcessor {
	return &FFmpegProcessor{
		config:         config,
		logger:         logger,
		maxStderrLines: 50, // Keep last 50 stderr lines for debugging
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
	fp.processExited = make(chan struct{})
	fp.stderrBuffer = make([]string, 0, fp.maxStderrLines)

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

	// Close pipes first to signal the process to stop
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
			fp.logger.Debug("FFmpeg process terminated gracefully", map[string]interface{}{
				"pid": fp.cmd.Process.Pid,
			})
		case <-time.After(5 * time.Second):
			// Force kill if graceful shutdown takes too long
			fp.logger.Warn("FFmpeg process did not terminate gracefully, force killing", map[string]interface{}{
				"pid":           fp.cmd.Process.Pid,
				"recent_stderr": fp.getRecentStderr(),
			})
			if err := syscall.Kill(-fp.cmd.Process.Pid, syscall.SIGKILL); err != nil {
				fp.logger.Error("Failed to force kill process group", err, map[string]interface{}{
					"pid": fp.cmd.Process.Pid,
				})
			}
			<-done // Wait for the process to actually exit
		}
	}

	// Signal that the process has exited
	if fp.processExited != nil {
		close(fp.processExited)
		fp.processExited = nil
	}

	fp.isRunning = false
	fp.cmd = nil
	fp.currentURL = ""
	fp.stderrBuffer = nil

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

	defer func() {
		if r := recover(); r != nil {
			fp.logger.Error("Panic in stderr monitor", fmt.Errorf("panic: %v", r), map[string]interface{}{
				"url": fp.currentURL,
			})
		}
	}()

	scanner := bufio.NewScanner(fp.errorPipe)
	for scanner.Scan() {
		line := scanner.Text()

		// Add to stderr buffer for debugging (thread-safe)
		fp.mu.Lock()
		if len(fp.stderrBuffer) >= fp.maxStderrLines {
			// Remove oldest line
			fp.stderrBuffer = fp.stderrBuffer[1:]
		}
		fp.stderrBuffer = append(fp.stderrBuffer, line)
		fp.mu.Unlock()

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

		// Check for critical error patterns that might indicate process failure
		if fp.isCriticalError(line) {
			fp.logger.Error("FFmpeg critical error detected", fmt.Errorf("critical error: %s", line), map[string]interface{}{
				"error": line,
				"url":   fp.currentURL,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		fp.logger.Error("Error reading FFmpeg stderr", err, map[string]interface{}{
			"url":           fp.currentURL,
			"recent_stderr": fp.getRecentStderr(),
		})
	}

	fp.logger.Debug("FFmpeg stderr monitoring stopped", map[string]interface{}{
		"url": fp.currentURL,
	})
}

// monitorProcess monitors the FFmpeg process and handles unexpected exits
func (fp *FFmpegProcessor) monitorProcess() {
	if fp.cmd == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fp.logger.Error("Panic in process monitor", fmt.Errorf("panic: %v", r), map[string]interface{}{
				"url": fp.currentURL,
			})
		}
	}()

	// Wait for the process to exit
	err := fp.cmd.Wait()

	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.isRunning {
		// Process exited unexpectedly
		fp.isRunning = false

		if err != nil {
			exitCode := -1
			if fp.cmd.ProcessState != nil {
				exitCode = fp.cmd.ProcessState.ExitCode()
			}

			fp.logger.Error("FFmpeg process exited unexpectedly", err, map[string]interface{}{
				"url":           fp.currentURL,
				"exit_code":     exitCode,
				"recent_stderr": fp.getRecentStderr(),
			})
		} else {
			fp.logger.Info("FFmpeg process completed normally", map[string]interface{}{
				"url": fp.currentURL,
			})
		}

		// Clean up resources
		fp.cleanupResources()
	}

	// Signal that the process has exited
	if fp.processExited != nil {
		close(fp.processExited)
		fp.processExited = nil
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

// isCriticalError checks if a stderr line indicates a critical error that might cause process failure
func (fp *FFmpegProcessor) isCriticalError(line string) bool {
	criticalPatterns := []string{
		"Segmentation fault",
		"segmentation fault",
		"Fatal error",
		"fatal error",
		"Assertion failed",
		"assertion failed",
		"Out of memory",
		"out of memory",
		"Killed",
		"killed",
		"Aborted",
		"aborted",
	}

	for _, pattern := range criticalPatterns {
		if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// getRecentStderr returns the recent stderr lines for debugging (thread-safe)
func (fp *FFmpegProcessor) getRecentStderr() []string {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	if fp.stderrBuffer == nil {
		return nil
	}

	// Return a copy to avoid race conditions
	result := make([]string, len(fp.stderrBuffer))
	copy(result, fp.stderrBuffer)
	return result
}

// cleanupResources cleans up all process-related resources
func (fp *FFmpegProcessor) cleanupResources() {
	// Clean up pipes
	if fp.outputPipe != nil {
		fp.outputPipe.Close()
		fp.outputPipe = nil
	}
	if fp.errorPipe != nil {
		fp.errorPipe.Close()
		fp.errorPipe = nil
	}

	// Clear process reference
	fp.cmd = nil
	fp.currentURL = ""
	fp.stderrBuffer = nil
}

// WaitForExit waits for the process to exit with a timeout
// This is useful for testing and ensuring proper cleanup
func (fp *FFmpegProcessor) WaitForExit(timeout time.Duration) error {
	fp.mu.RLock()
	processExited := fp.processExited
	isRunning := fp.isRunning
	fp.mu.RUnlock()

	if !isRunning {
		return nil // Already stopped
	}

	if processExited == nil {
		return fmt.Errorf("process exit channel not initialized")
	}

	select {
	case <-processExited:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for process to exit")
	}
}

// GetProcessInfo returns information about the current process for monitoring
func (fp *FFmpegProcessor) GetProcessInfo() map[string]interface{} {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	info := map[string]interface{}{
		"is_running":   fp.isRunning,
		"current_url":  fp.currentURL,
		"stderr_lines": len(fp.stderrBuffer),
	}

	if fp.cmd != nil && fp.cmd.Process != nil {
		info["pid"] = fp.cmd.Process.Pid
	}

	return info
}
