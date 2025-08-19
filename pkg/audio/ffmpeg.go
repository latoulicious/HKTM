package audio

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
	tempFile       string        // Path to temporary audio file
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

	// Download the audio file first (this avoids HLS streaming issues)
	tempFile, err := fp.downloadAudioFile(url)
	if err != nil {
		urlLogger.Error("Failed to download audio file", err, CreateContextFieldsWithComponent("", "", url, "ffmpeg"))
		return nil, fmt.Errorf("failed to download audio file: %w", err)
	}

	urlLogger.Info("Successfully downloaded audio file", CreateContextFieldsWithComponent("", "", url, "ffmpeg"))

	// Build the FFmpeg command arguments with the local file
	args := fp.buildFFmpegArgsWithLocalFile(tempFile)

	// Log the FFmpeg command for debugging
	contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
	contextFields["ffmpeg_command"] = fp.config.BinaryPath + " " + strings.Join(args, " ")
	contextFields["temp_file"] = tempFile
	urlLogger.Info("Starting FFmpeg process", contextFields)

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

	// Give FFmpeg a moment to initialize and start processing the stream
	// This helps prevent immediate "file already closed" errors
	time.Sleep(100 * time.Millisecond)

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

	// Check if the URL is already a direct stream URL (Google Video manifest)
	if strings.Contains(url, "googlevideo.com") || strings.Contains(url, "manifest/hls_playlist") {
		contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
		contextFields["url_type"] = "direct_stream"
		contextFields["skip_yt_dlp"] = true
		fp.pipelineLogger.Info("URL is already a direct stream URL, skipping yt-dlp", contextFields)
		return url, nil
	}

	// Try multiple strategies to get a stable stream URL
	strategies := [][]string{
		// Strategy 1: Best audio with specific format preference (avoid HLS when possible)
		{"-f", "bestaudio[ext=m4a]/bestaudio[ext=webm]/bestaudio[ext=mp4]/bestaudio", "--get-url"},
		// Strategy 2: Force non-HLS formats
		{"-f", "bestaudio[protocol!=m3u8]/bestaudio", "--get-url"},
		// Strategy 3: Fallback to any audio
		{"-f", "bestaudio", "--get-url"},
	}

	var output []byte
	var err error
	
	for i, strategy := range strategies {
		fp.pipelineLogger.Debug(fmt.Sprintf("Trying yt-dlp strategy %d/%d", i+1, len(strategies)), CreateContextFieldsWithComponent("", "", url, "ffmpeg"))
		
		cmd := exec.Command("yt-dlp", strategy...)
		cmd.Args = append(cmd.Args, url)
		output, err = cmd.CombinedOutput()
		
		if err == nil {
			fp.pipelineLogger.Info(fmt.Sprintf("Successfully got URL using strategy %d", i+1), CreateContextFieldsWithComponent("", "", url, "ffmpeg"))
			break
		}
		
		// Log failed attempt
		contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
		contextFields["strategy"] = i + 1
		contextFields["yt_dlp_output"] = string(output)
		contextFields["yt_dlp_command"] = "yt-dlp " + strings.Join(strategy, " ")
		fp.pipelineLogger.Warn(fmt.Sprintf("yt-dlp strategy %d failed", i+1), contextFields)
	}
	
	if err != nil {
		contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
		contextFields["final_yt_dlp_output"] = string(output)
		fp.pipelineLogger.Error("All yt-dlp strategies failed", err, contextFields)
		return "", fmt.Errorf("yt-dlp failed to get stream URL: %w", err)
	}

	rawOutput := strings.TrimSpace(string(output))
	if rawOutput == "" {
		return "", fmt.Errorf("yt-dlp returned empty output")
	}

	// Parse the output to extract only the URL (yt-dlp may include warnings)
	lines := strings.Split(rawOutput, "\n")
	var streamURL string
	
	// Find the first line that looks like a URL (starts with http)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "http") {
			streamURL = line
			break
		}
	}
	
	if streamURL == "" {
		// Log the raw output for debugging
		contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
		contextFields["raw_yt_dlp_output"] = rawOutput
		fp.pipelineLogger.Error("No valid URL found in yt-dlp output", fmt.Errorf("no URL found"), contextFields)
		return "", fmt.Errorf("yt-dlp did not return a valid URL")
	}

	// Log successful URL extraction with truncated URL for security
	contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
	if len(streamURL) > 100 {
		contextFields["extracted_url"] = streamURL[:100] + "..."
	} else {
		contextFields["extracted_url"] = streamURL
	}
	contextFields["warnings_filtered"] = len(lines) > 1
	fp.pipelineLogger.Info("Successfully extracted stream URL from yt-dlp", contextFields)

	fp.pipelineLogger.Debug("Successfully got stream URL from yt-dlp", CreateContextFieldsWithComponent("", "", url, "ffmpeg"))
	return streamURL, nil
}

// buildFFmpegArgsWithStreamURL constructs the FFmpeg command arguments with a direct stream URL
func (fp *FFmpegProcessor) buildFFmpegArgsWithStreamURL(streamURL string) []string {
	args := []string{
		// Input options for better stream handling
		"-reconnect", "1",
		"-reconnect_streamed", "1", 
		"-reconnect_delay_max", "5",
		"-i", streamURL,
		// Output format options
		"-f", fp.config.AudioFormat,
		"-ar", fmt.Sprintf("%d", fp.config.SampleRate),
		"-ac", fmt.Sprintf("%d", fp.config.Channels),
		// Additional stability options
		"-avoid_negative_ts", "make_zero",
		"-fflags", "+genpts",
	}

	// Add custom arguments from configuration (but avoid duplicates)
	for _, customArg := range fp.config.CustomArgs {
		// Skip if we already added these
		if customArg != "-reconnect" && customArg != "1" && 
		   customArg != "-reconnect_delay_max" && customArg != "5" {
			args = append(args, customArg)
		}
	}

	// Add output to stdout
	args = append(args, "pipe:1")

	return args
}

// downloadAudioFile downloads the audio using yt-dlp to a temporary file
func (fp *FFmpegProcessor) downloadAudioFile(url string) (string, error) {
	// Create a temporary file for the audio
	tempFile := fmt.Sprintf("/tmp/audio_%d.m4a", time.Now().UnixNano())
	
	contextFields := CreateContextFieldsWithComponent("", "", url, "ffmpeg")
	contextFields["temp_file"] = tempFile
	fp.pipelineLogger.Debug("Starting audio download", contextFields)
	
	// Use yt-dlp to download the audio file
	cmd := exec.Command("yt-dlp", "-f", "bestaudio", "-o", tempFile, url)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		contextFields["yt_dlp_output"] = string(output)
		fp.pipelineLogger.Error("yt-dlp download failed", err, contextFields)
		return "", fmt.Errorf("yt-dlp download failed: %w", err)
	}
	
	// Verify the file was created and has content
	info, statErr := os.Stat(tempFile)
	if statErr != nil || info.Size() == 0 {
		fp.pipelineLogger.Error("Downloaded file is missing or empty", statErr, contextFields)
		return "", fmt.Errorf("downloaded file is missing or empty")
	}
	
	contextFields["file_size"] = fmt.Sprintf("%.2f MB", float64(info.Size())/1024/1024)
	fp.pipelineLogger.Info("Audio download completed", contextFields)
	
	// Store the temp file path for cleanup later
	fp.tempFile = tempFile
	
	return tempFile, nil
}

// buildFFmpegArgsWithLocalFile constructs FFmpeg arguments for a local file
func (fp *FFmpegProcessor) buildFFmpegArgsWithLocalFile(filePath string) []string {
	args := []string{
		// Input options for faster processing
		"-re", // Read input at native frame rate (important for real-time streaming)
		"-i", filePath,
		// Output format options
		"-f", fp.config.AudioFormat,
		"-ar", fmt.Sprintf("%d", fp.config.SampleRate),
		"-ac", fmt.Sprintf("%d", fp.config.Channels),
		// Performance optimizations
		"-threads", "0", // Use all available CPU threads
		"-preset", "ultrafast", // Fastest encoding preset
		"-avoid_negative_ts", "make_zero",
		"-fflags", "+genpts",
	}

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
			
			// Also log to console immediately
			fmt.Printf("[FFmpeg] Process exited with code %d, error: %v\n", exitCode, err)
			fmt.Printf("[FFmpeg] Recent stderr: %v\n", fp.getRecentStderr())
		} else {
			fp.pipelineLogger.Info("FFmpeg process completed normally", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
			fmt.Printf("[FFmpeg] Process completed normally\n")
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

	// Clean up temporary file
	if fp.tempFile != "" {
		if err := os.Remove(fp.tempFile); err != nil {
			fp.pipelineLogger.Warn("Failed to remove temporary file", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
		} else {
			fp.pipelineLogger.Debug("Cleaned up temporary file", CreateContextFieldsWithComponent("", "", fp.currentURL, "ffmpeg"))
		}
		fp.tempFile = ""
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
