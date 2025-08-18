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
	pipelineLogger AudioLogger  // Pipeline-specific logger with context
	processExited  chan struct{} // Channel to signal when process has exited
	stderrBuffer   []string      // Buffer to store recent stderr lines for debugging
	maxStderrLines int           // Maximum number of stderr lines to keep
}

// NewFFmpegProcessor creates a new FFmpegProcessor instance
func NewFFmpegProcessor(config *FFmpegConfig, logger AudioLogger) StreamProcessor {
	// Create pipeline-specific logger context for FFmpeg operations
	pipelineLogger := logger.WithPipeline("ffmpeg")
	
	return &FFmpegProcessor{
		config:         config,
		logger:         logger,
		pipelineLogger: pipelineLogger,
		maxStderrLines: 50, // Keep last 50 stderr lines for debugging
	}
}

// StartStream starts the yt-dlp + FFmpeg pipeline for the given URL
func (fp *FFmpegProcessor) StartStream(url string) (io.ReadCloser, error) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// Create pipeline-specific logger with URL context
	urlLogger := fp.pipelineLogger
	
	// Stop any existing stream first
	if fp.isRunning {
		if err := fp.stopInternal(); err != nil {
			urlLogger.Warn("Failed to stop existing stream before starting new one", CreateContextFieldsWithComponent("", "", url, "ffmpeg"))
		}
	}

	fp.currentURL = url

	// First, get the direct stream URL using yt-dlp
	streamURL, err := fp.getStreamURL(url)
	if err != nil {
		urlLogger.Error("Failed to get stream URL from yt-dlp", err, CreateContextFieldsWithComponent("", "", url, "ffmpeg"))
		return nil, fmt.Errorf("failed to get stream URL: %w", err)
	}

	urlLogger.Debug("Got stream URL from yt-dlp", CreateContextFieldsWithComponent("", "", url, "ffmpeg"))

	// Build the FFmpeg command arguments with the direct stream URL
	args := fp.buildFFmpegArgsWithStreamURL(streamURL)

	urlLogger.Debug("Starting FFmpeg process", CreateContextFieldsWithComponent("", "", url, "ffmpeg"))

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

	urlLogger.Info("FFmpeg stream started successfully", CreateContextFieldsWithComponent("", "", url, "ffmpeg"))

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

	fp.pipelineLogger.Debug("Stopping FFmpeg process", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))

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
			fp.pipelineLogger.Warn("Failed to send SIGTERM to process group", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
		}

		// Wait for graceful shutdown with timeout
		done := make(chan error, 1)
		go func() {
			done <- fp.cmd.Wait()
		}()

		select {
		case <-done:
			// Process terminated gracefully
			fp.pipelineLogger.Debug("FFmpeg process terminated gracefully", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
		case <-time.After(5 * time.Second):
			// Force kill if graceful shutdown takes too long
			contextFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
			contextFields["recent_stderr"] = fp.getRecentStderr()
			fp.pipelineLogger.Warn("FFmpeg process did not terminate gracefully, force killing", contextFields)
			if err := syscall.Kill(-fp.cmd.Process.Pid, syscall.SIGKILL); err != nil {
				fp.pipelineLogger.Error("Failed to force kill process group", err, CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
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

	fp.pipelineLogger.Info("FFmpeg process stopped", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
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
	contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
	contextFields["old_url"] = fp.currentURL
	fp.pipelineLogger.Info("Restarting FFmpeg stream", contextFields)

	// Stop current stream
	if err := fp.Stop(); err != nil {
		fp.pipelineLogger.Error("Failed to stop current stream during restart", err, CreateContextFieldsWithComponent("", "", url, "ffmpeg"))
		return fmt.Errorf("failed to stop current stream: %w", err)
	}

	// Start new stream
	_, err := fp.StartStream(url)
	return err
}

// getStreamURL uses yt-dlp to extract the direct stream URL
func (fp *FFmpegProcessor) getStreamURL(url string) (string, error) {
	fp.pipelineLogger.Debug("Getting stream URL with yt-dlp", CreateContextFieldsWithComponent("", "", url, "ffmpeg"))

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
			fp.pipelineLogger.Error("Panic in stderr monitor", fmt.Errorf("panic: %v", r), CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
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
		contextFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
		contextFields["output"] = line
		fp.pipelineLogger.Debug("FFmpeg stderr", contextFields)

		// Check for specific error patterns
		if fp.isErrorLine(line) {
			errorFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
			errorFields["error"] = line
			fp.pipelineLogger.Warn("FFmpeg error detected", errorFields)
		}

		// Check for critical error patterns that might indicate process failure
		if fp.isCriticalError(line) {
			criticalFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
			criticalFields["error"] = line
			fp.pipelineLogger.Error("FFmpeg critical error detected", fmt.Errorf("critical error: %s", line), criticalFields)
		}
	}

	if err := scanner.Err(); err != nil {
		contextFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
		contextFields["recent_stderr"] = fp.getRecentStderr()
		fp.pipelineLogger.Error("Error reading FFmpeg stderr", err, contextFields)
	}

	fp.pipelineLogger.Debug("FFmpeg stderr monitoring stopped", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
}

// monitorProcess monitors the FFmpeg process and handles unexpected exits
func (fp *FFmpegProcessor) monitorProcess() {
	if fp.cmd == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fp.pipelineLogger.Error("Panic in process monitor", fmt.Errorf("panic: %v", r), CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
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

			contextFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
			contextFields["exit_code"] = exitCode
			contextFields["recent_stderr"] = fp.getRecentStderr()
			fp.pipelineLogger.Error("FFmpeg process exited unexpectedly", err, contextFields)
		} else {
			fp.pipelineLogger.Info("FFmpeg process completed normally", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
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
