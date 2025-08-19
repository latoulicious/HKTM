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
	ytdlpConfig    *YtDlpConfig
	cmd            *exec.Cmd
	ytdlpCmd       *exec.Cmd // yt-dlp process for piping
	outputPipe     io.ReadCloser
	errorPipe      io.ReadCloser
	ytdlpErrorPipe io.ReadCloser // yt-dlp stderr pipe
	isRunning      bool
	currentURL     string
	retryCount     int // Current retry attempt
	maxRetries     int // Maximum retry attempts (3 as per requirements)
	mu             sync.RWMutex
	logger         AudioLogger
	pipelineLogger AudioLogger   // Pipeline-specific logger with context
	processExited  chan struct{} // Channel to signal when process has exited
	stderrBuffer   []string      // Buffer to store recent stderr lines for debugging
	maxStderrLines int           // Maximum number of stderr lines to keep
}

// NewFFmpegProcessor creates a new FFmpegProcessor instance
func NewFFmpegProcessor(config *FFmpegConfig, ytdlpConfig *YtDlpConfig, logger AudioLogger) StreamProcessor {
	// Create pipeline-specific logger context for FFmpeg operations
	pipelineLogger := logger.WithPipeline("ffmpeg")

	return &FFmpegProcessor{
		config:         config,
		ytdlpConfig:    ytdlpConfig,
		logger:         logger,
		pipelineLogger: pipelineLogger,
		maxStderrLines: 50, // Keep last 50 stderr lines for debugging
		maxRetries:     3,  // 3 attempts max as per requirements
	}
}

// StartStream starts the yt-dlp | FFmpeg pipeline for the given URL
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
	fp.retryCount = 0

	// Start the streaming pipeline with retry logic
	return fp.startStreamWithRetry(url, urlLogger)
}

// startStreamWithRetry attempts to start the streaming pipeline with retry logic
func (fp *FFmpegProcessor) startStreamWithRetry(url string, urlLogger AudioLogger) (io.ReadCloser, error) {
	var lastErr error

	for attempt := 0; attempt <= fp.maxRetries; attempt++ {
		fp.retryCount = attempt

		contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
		contextFields["attempt"] = attempt + 1
		contextFields["max_attempts"] = fp.maxRetries + 1

		if attempt > 0 {
			urlLogger.Info("Retrying stream start", contextFields)
			// Simple delay between retries: 2s, 5s, 10s
			delay := time.Duration(2+attempt*3) * time.Second
			time.Sleep(delay)
		} else {
			urlLogger.Info("Starting streaming pipeline", contextFields)
		}

		// Try to start the pipeline
		reader, err := fp.startPipeline(url, urlLogger)
		if err == nil {
			urlLogger.Info("Streaming pipeline started successfully", contextFields)
			return reader, nil
		}

		lastErr = err
		contextFields["error"] = err.Error()
		urlLogger.Warn("Stream start attempt failed", contextFields)

		// Clean up failed attempt
		fp.stopInternal()
	}

	// All retries exhausted
	contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
	contextFields["total_attempts"] = fp.maxRetries + 1
	contextFields["final_error"] = lastErr.Error()
	urlLogger.Error("All stream start attempts failed", lastErr, contextFields)

	return nil, fmt.Errorf("failed to start stream after %d attempts: %w", fp.maxRetries+1, lastErr)
}

// startPipeline starts the yt-dlp | ffmpeg pipeline
func (fp *FFmpegProcessor) startPipeline(url string, urlLogger AudioLogger) (io.ReadCloser, error) {
	// Build yt-dlp command: yt-dlp -o - [url]
	ytdlpArgs := fp.buildYtdlpArgs(url)
	fp.ytdlpCmd = exec.Command(fp.ytdlpConfig.BinaryPath, ytdlpArgs...)

	// Build FFmpeg command: ffmpeg -i pipe:0 [options] pipe:1
	ffmpegArgs := fp.buildFFmpegPipeArgs()
	fp.cmd = exec.Command(fp.config.BinaryPath, ffmpegArgs...)

	// Set up process groups for proper cleanup
	fp.ytdlpCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	fp.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Create pipes: yt-dlp stdout -> ffmpeg stdin
	ytdlpStdout, err := fp.ytdlpCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create yt-dlp stdout pipe: %w", err)
	}

	// Connect yt-dlp output to ffmpeg input
	fp.cmd.Stdin = ytdlpStdout

	// Create FFmpeg output pipe (audio data)
	ffmpegStdout, err := fp.cmd.StdoutPipe()
	if err != nil {
		ytdlpStdout.Close()
		return nil, fmt.Errorf("failed to create ffmpeg stdout pipe: %w", err)
	}
	fp.outputPipe = ffmpegStdout

	// Create stderr pipes for monitoring
	ytdlpStderr, err := fp.ytdlpCmd.StderrPipe()
	if err != nil {
		ytdlpStdout.Close()
		ffmpegStdout.Close()
		return nil, fmt.Errorf("failed to create yt-dlp stderr pipe: %w", err)
	}
	fp.ytdlpErrorPipe = ytdlpStderr

	ffmpegStderr, err := fp.cmd.StderrPipe()
	if err != nil {
		ytdlpStdout.Close()
		ffmpegStdout.Close()
		ytdlpStderr.Close()
		return nil, fmt.Errorf("failed to create ffmpeg stderr pipe: %w", err)
	}
	fp.errorPipe = ffmpegStderr

	// Log the commands for debugging
	contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
	contextFields["ytdlp_command"] = fp.ytdlpConfig.BinaryPath + " " + strings.Join(ytdlpArgs, " ")
	contextFields["ffmpeg_command"] = fp.config.BinaryPath + " " + strings.Join(ffmpegArgs, " ")
	urlLogger.Info("Starting yt-dlp | ffmpeg pipeline", contextFields)

	// Start yt-dlp first
	if err := fp.ytdlpCmd.Start(); err != nil {
		ytdlpStdout.Close()
		ffmpegStdout.Close()
		ytdlpStderr.Close()
		ffmpegStderr.Close()
		return nil, fmt.Errorf("failed to start yt-dlp process: %w", err)
	}

	// Start FFmpeg
	if err := fp.cmd.Start(); err != nil {
		fp.ytdlpCmd.Process.Kill()
		ytdlpStdout.Close()
		ffmpegStdout.Close()
		ytdlpStderr.Close()
		ffmpegStderr.Close()
		return nil, fmt.Errorf("failed to start ffmpeg process: %w", err)
	}

	fp.isRunning = true
	fp.processExited = make(chan struct{})
	fp.stderrBuffer = make([]string, 0, fp.maxStderrLines)

	// Start monitoring both processes
	go fp.monitorYtdlpStderr()
	go fp.monitorStderr()
	go fp.monitorProcess()

	urlLogger.Info("yt-dlp | ffmpeg pipeline started successfully", contextFields)

	// Give the pipeline a moment to initialize
	time.Sleep(100 * time.Millisecond)

	return ffmpegStdout, nil
}

// buildYtdlpArgs constructs the yt-dlp command arguments for piping
func (fp *FFmpegProcessor) buildYtdlpArgs(url string) []string {
	args := []string{
		"-o", "-", // Output to stdout for piping
		"--quiet",               // Reduce output noise
		"--no-warnings",         // Suppress warnings
		"--format", "bestaudio", // Get best audio quality
	}

	// Add custom arguments from configuration
	args = append(args, fp.ytdlpConfig.CustomArgs...)

	// Add the URL as the last argument
	args = append(args, url)

	return args
}

// buildFFmpegPipeArgs constructs FFmpeg arguments for reading from pipe
func (fp *FFmpegProcessor) buildFFmpegPipeArgs() []string {
	args := []string{
		// Input from pipe (yt-dlp output)
		"-i", "pipe:0",
		// Output format options
		"-f", fp.config.AudioFormat,
		"-ar", fmt.Sprintf("%d", fp.config.SampleRate),
		"-ac", fmt.Sprintf("%d", fp.config.Channels),
		// Stability options for streaming
		"-avoid_negative_ts", "make_zero",
		"-fflags", "+genpts",
		// Reduce output noise
		"-hide_banner",
		"-loglevel", "error",
	}

	// Add custom arguments from configuration (but avoid duplicates)
	for _, customArg := range fp.config.CustomArgs {
		// Skip arguments we already added
		if customArg != "-avoid_negative_ts" && customArg != "make_zero" &&
			customArg != "-fflags" && customArg != "+genpts" &&
			customArg != "-hide_banner" && customArg != "-loglevel" && customArg != "error" {
			args = append(args, customArg)
		}
	}

	// Output to stdout
	args = append(args, "pipe:1")

	return args
}

// Stop stops the current FFmpeg process
func (fp *FFmpegProcessor) Stop() error {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	return fp.stopInternal()
}

// stopInternal stops the process without acquiring the lock (internal use)
func (fp *FFmpegProcessor) stopInternal() error {
	if !fp.isRunning {
		return nil
	}

	fp.pipelineLogger.Debug("Stopping yt-dlp | ffmpeg pipeline", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))

	// Close pipes first to signal the processes to stop
	if fp.outputPipe != nil {
		fp.outputPipe.Close()
		fp.outputPipe = nil
	}
	if fp.errorPipe != nil {
		fp.errorPipe.Close()
		fp.errorPipe = nil
	}
	if fp.ytdlpErrorPipe != nil {
		fp.ytdlpErrorPipe.Close()
		fp.ytdlpErrorPipe = nil
	}

	// Stop both processes with timeout
	fp.stopProcessWithTimeout(fp.cmd, "ffmpeg")
	fp.stopProcessWithTimeout(fp.ytdlpCmd, "yt-dlp")

	// Signal that the processes have exited
	if fp.processExited != nil {
		close(fp.processExited)
		fp.processExited = nil
	}

	fp.isRunning = false
	fp.cmd = nil
	fp.ytdlpCmd = nil
	fp.currentURL = ""
	fp.stderrBuffer = nil

	fp.pipelineLogger.Info("yt-dlp | ffmpeg pipeline stopped", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
	return nil
}

// stopProcessWithTimeout stops a process with graceful termination and timeout
func (fp *FFmpegProcessor) stopProcessWithTimeout(cmd *exec.Cmd, processName string) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	contextFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
	contextFields["process"] = processName

	// Try graceful termination first
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
		contextFields["error"] = err.Error()
		fp.pipelineLogger.Warn("Failed to send SIGTERM to process group", contextFields)
	}

	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Process terminated gracefully
		fp.pipelineLogger.Debug("Process terminated gracefully", contextFields)
	case <-time.After(5 * time.Second):
		// Force kill if graceful shutdown takes too long
		contextFields["recent_stderr"] = fp.getRecentStderr()
		fp.pipelineLogger.Warn("Process did not terminate gracefully, force killing", contextFields)
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			contextFields["kill_error"] = err.Error()
			fp.pipelineLogger.Error("Failed to force kill process group", err, contextFields)
		}
		<-done // Wait for the process to actually exit
	}
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

// monitorYtdlpStderr monitors the stderr output from yt-dlp for errors and warnings
func (fp *FFmpegProcessor) monitorYtdlpStderr() {
	if fp.ytdlpErrorPipe == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fp.pipelineLogger.Error("Panic in yt-dlp stderr monitor", fmt.Errorf("panic: %v", r), CreateContextFieldsWithComponent("", "", fp.currentURL, "ytdlp"))
		}
	}()

	scanner := bufio.NewScanner(fp.ytdlpErrorPipe)
	for scanner.Scan() {
		line := scanner.Text()

		// Log yt-dlp output for debugging
		contextFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ytdlp")
		contextFields["output"] = line
		fp.pipelineLogger.Debug("yt-dlp stderr", contextFields)

		// Also log to console immediately
		fmt.Printf("[yt-dlp] %s\n", line)

		// Check for yt-dlp specific errors
		if fp.isYtdlpError(line) {
			errorFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ytdlp")
			errorFields["error"] = line
			fp.pipelineLogger.Warn("yt-dlp error detected", errorFields)
		}
	}

	if err := scanner.Err(); err != nil {
		contextFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ytdlp")
		fp.pipelineLogger.Error("Error reading yt-dlp stderr", err, contextFields)
	}

	fp.pipelineLogger.Debug("yt-dlp stderr monitoring stopped", CreateContextFieldsWithComponent("", "", fp.currentURL, "ytdlp"))
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

		// Log FFmpeg output for debugging (both to database and console for immediate visibility)
		contextFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
		contextFields["output"] = line
		fp.pipelineLogger.Debug("FFmpeg stderr", contextFields)

		// Also log to console immediately to avoid database delays
		fmt.Printf("[FFmpeg] %s\n", line)

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

		// Check for specific HLS/stream related errors
		if fp.isStreamError(line) {
			streamFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
			streamFields["stream_error"] = line
			fp.pipelineLogger.Error("FFmpeg stream error detected", fmt.Errorf("stream error: %s", line), streamFields)
		}
	}

	if err := scanner.Err(); err != nil {
		contextFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
		contextFields["recent_stderr"] = fp.getRecentStderr()
		fp.pipelineLogger.Error("Error reading FFmpeg stderr", err, contextFields)
	}

	fp.pipelineLogger.Debug("FFmpeg stderr monitoring stopped", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
}

// monitorProcess monitors both yt-dlp and FFmpeg processes and handles unexpected exits
func (fp *FFmpegProcessor) monitorProcess() {
	if fp.cmd == nil && fp.ytdlpCmd == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fp.pipelineLogger.Error("Panic in process monitor", fmt.Errorf("panic: %v", r), CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
		}
	}()

	// Monitor both processes concurrently
	ffmpegDone := make(chan error, 1)
	ytdlpDone := make(chan error, 1)

	// Monitor FFmpeg process
	if fp.cmd != nil {
		go func() {
			ffmpegDone <- fp.cmd.Wait()
		}()
	} else {
		close(ffmpegDone)
	}

	// Monitor yt-dlp process
	if fp.ytdlpCmd != nil {
		go func() {
			ytdlpDone <- fp.ytdlpCmd.Wait()
		}()
	} else {
		close(ytdlpDone)
	}

	// Wait for either process to exit
	var ffmpegErr, ytdlpErr error
	var processName string

	select {
	case ffmpegErr = <-ffmpegDone:
		processName = "ffmpeg"
		// If FFmpeg exits, yt-dlp should also be stopped
		if fp.ytdlpCmd != nil && fp.ytdlpCmd.Process != nil {
			fp.ytdlpCmd.Process.Kill()
		}
	case ytdlpErr = <-ytdlpDone:
		processName = "yt-dlp"
		// If yt-dlp exits, FFmpeg should also be stopped
		if fp.cmd != nil && fp.cmd.Process != nil {
			fp.cmd.Process.Kill()
		}
	}

	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.isRunning {
		// Process exited unexpectedly
		fp.isRunning = false

		// Determine which error to report
		var err error
		var exitCode int = -1

		if processName == "ffmpeg" && ffmpegErr != nil {
			err = ffmpegErr
			if fp.cmd.ProcessState != nil {
				exitCode = fp.cmd.ProcessState.ExitCode()
			}
		} else if processName == "yt-dlp" && ytdlpErr != nil {
			err = ytdlpErr
			if fp.ytdlpCmd.ProcessState != nil {
				exitCode = fp.ytdlpCmd.ProcessState.ExitCode()
			}
		}

		if err != nil {
			contextFields := CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg")
			contextFields["exit_code"] = exitCode
			contextFields["failed_process"] = processName
			contextFields["recent_stderr"] = fp.getRecentStderr()
			fp.pipelineLogger.Error("Pipeline process exited unexpectedly", err, contextFields)

			// Also log to console immediately
			fmt.Printf("[%s] Process exited with code %d, error: %v\n", processName, exitCode, err)
			fmt.Printf("[Pipeline] Recent stderr: %v\n", fp.getRecentStderr())
		} else {
			fp.pipelineLogger.Info("Pipeline processes completed normally", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
			fmt.Printf("[Pipeline] Processes completed normally\n")
		}

		// Clean up resources
		fp.cleanupResources()
	}

	// Signal that the processes have exited
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

// isStreamError checks if a stderr line indicates a stream-related error
func (fp *FFmpegProcessor) isStreamError(line string) bool {
	streamErrorPatterns := []string{
		"Connection refused",
		"connection refused",
		"HTTP error",
		"http error",
		"Server returned",
		"server returned",
		"403 Forbidden",
		"404 Not Found",
		"Immediate exit requested",
		"immediate exit requested",
		"No such file or directory",
		"Protocol not found",
		"protocol not found",
		"Invalid data found",
		"invalid data found",
		"End of file",
		"end of file",
		"I/O error",
		"i/o error",
		"Network is unreachable",
		"network is unreachable",
		"Operation timed out",
		"operation timed out",
	}

	for _, pattern := range streamErrorPatterns {
		if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// isYtdlpError checks if a stderr line indicates a yt-dlp error
func (fp *FFmpegProcessor) isYtdlpError(line string) bool {
	ytdlpErrorPatterns := []string{
		"ERROR:",
		"error:",
		"Unable to download",
		"unable to download",
		"HTTP Error",
		"http error",
		"Video unavailable",
		"video unavailable",
		"Private video",
		"private video",
		"This video is not available",
		"this video is not available",
		"Sign in to confirm",
		"sign in to confirm",
		"403 Forbidden",
		"404 Not Found",
		"Connection refused",
		"connection refused",
		"Timeout",
		"timeout",
	}

	for _, pattern := range ytdlpErrorPatterns {
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
	if fp.ytdlpErrorPipe != nil {
		fp.ytdlpErrorPipe.Close()
		fp.ytdlpErrorPipe = nil
	}

	// Clear process references
	fp.cmd = nil
	fp.ytdlpCmd = nil
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
		// Check if process is still alive
		if fp.cmd.ProcessState != nil {
			info["process_exited"] = fp.cmd.ProcessState.Exited()
			info["exit_code"] = fp.cmd.ProcessState.ExitCode()
		}
	}

	return info
}

// IsProcessAlive checks if the FFmpeg process is still running
func (fp *FFmpegProcessor) IsProcessAlive() bool {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	if fp.cmd == nil || fp.cmd.Process == nil {
		return false
	}

	// If ProcessState is available and shows exited, process is dead
	if fp.cmd.ProcessState != nil && fp.cmd.ProcessState.Exited() {
		return false
	}

	return fp.isRunning
}
