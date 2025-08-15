package audio

import (
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/audio"
)

// MockAudioLogger implements the AudioLogger interface for testing
type MockAudioLogger struct {
	logs []LogEntry
}

type LogEntry struct {
	Level   string
	Message string
	Error   error
	Fields  map[string]interface{}
}

func (m *MockAudioLogger) Info(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "INFO", Message: msg, Fields: fields})
}

func (m *MockAudioLogger) Error(msg string, err error, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "ERROR", Message: msg, Error: err, Fields: fields})
}

func (m *MockAudioLogger) Warn(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "WARN", Message: msg, Fields: fields})
}

func (m *MockAudioLogger) Debug(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "DEBUG", Message: msg, Fields: fields})
}

func TestFFmpegProcessor_Creation(t *testing.T) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{"-reconnect", "1"},
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(config, logger)

	if processor == nil {
		t.Fatal("NewFFmpegProcessor returned nil")
	}

	// Test initial state
	if processor.IsRunning() {
		t.Error("Processor should not be running initially")
	}
}

func TestFFmpegProcessor_IsRunning(t *testing.T) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(config, logger)

	// Initially should not be running
	if processor.IsRunning() {
		t.Error("Processor should not be running initially")
	}
}

func TestFFmpegProcessor_Stop_WhenNotRunning(t *testing.T) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(config, logger)

	// Stopping when not running should not error
	err := processor.Stop()
	if err != nil {
		t.Errorf("Stop() returned error when not running: %v", err)
	}
}

func TestFFmpegProcessor_StartStream_InvalidURL(t *testing.T) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(config, logger)

	// Test with invalid URL - this should fail at the yt-dlp stage
	_, err := processor.StartStream("invalid-url")
	if err == nil {
		t.Error("StartStream should fail with invalid URL")
	}

	// Should not be running after failed start
	if processor.IsRunning() {
		t.Error("Processor should not be running after failed start")
	}
}

func TestFFmpegProcessor_Restart_WhenNotRunning(t *testing.T) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(config, logger)

	// Restart when not running should behave like StartStream
	err := processor.Restart("invalid-url")
	if err == nil {
		t.Error("Restart should fail with invalid URL")
	}
}

// TestFFmpegProcessor_ConfigValidation tests that the processor uses the provided configuration
func TestFFmpegProcessor_ConfigValidation(t *testing.T) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "custom-ffmpeg",
		AudioFormat: "s32le",
		SampleRate:  44100,
		Channels:    1,
		CustomArgs:  []string{"-custom", "arg"},
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(config, logger)

	if processor == nil {
		t.Fatal("NewFFmpegProcessor returned nil")
	}

	// We can't easily test the internal config usage without exposing internals,
	// but we can verify the processor was created successfully with custom config
}

// TestFFmpegProcessor_ProcessMonitoring tests the enhanced process monitoring features
func TestFFmpegProcessor_ProcessMonitoring(t *testing.T) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(config, logger)

	// Test GetProcessInfo when not running
	info := processor.GetProcessInfo()
	if info["is_running"].(bool) {
		t.Error("Process should not be running initially")
	}
	if info["current_url"].(string) != "" {
		t.Error("Current URL should be empty initially")
	}
	if info["stderr_lines"].(int) != 0 {
		t.Error("Stderr lines should be 0 initially")
	}
}

// TestFFmpegProcessor_WaitForExit tests the WaitForExit functionality
func TestFFmpegProcessor_WaitForExit_WhenNotRunning(t *testing.T) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(config, logger)

	// WaitForExit should return immediately when not running
	err := processor.WaitForExit(1 * time.Second)
	if err != nil {
		t.Errorf("WaitForExit should not error when not running: %v", err)
	}
}

// TestFFmpegProcessor_ResourceCleanup tests that resources are properly cleaned up
func TestFFmpegProcessor_ResourceCleanup(t *testing.T) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(config, logger)

	// Test multiple stop calls don't cause issues
	err1 := processor.Stop()
	err2 := processor.Stop()

	if err1 != nil {
		t.Errorf("First Stop() call should not error: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second Stop() call should not error: %v", err2)
	}

	// Verify state is clean
	if processor.IsRunning() {
		t.Error("Processor should not be running after stop")
	}
}

// TestFFmpegProcessor_StderrBuffering tests the stderr buffering functionality
func TestFFmpegProcessor_StderrBuffering(t *testing.T) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	logger := &MockAudioLogger{}
	processor := audio.NewFFmpegProcessor(config, logger)

	// Test that GetProcessInfo works correctly
	info := processor.GetProcessInfo()
	if info == nil {
		t.Fatal("GetProcessInfo should not return nil")
	}

	// Verify expected fields exist
	if _, exists := info["is_running"]; !exists {
		t.Error("GetProcessInfo should include is_running field")
	}
	if _, exists := info["current_url"]; !exists {
		t.Error("GetProcessInfo should include current_url field")
	}
	if _, exists := info["stderr_lines"]; !exists {
		t.Error("GetProcessInfo should include stderr_lines field")
	}
}

// Benchmark test for processor creation
func BenchmarkFFmpegProcessor_Creation(b *testing.B) {
	config := &audio.FFmpegConfig{
		BinaryPath:  "ffmpeg",
		AudioFormat: "s16le",
		SampleRate:  48000,
		Channels:    2,
		CustomArgs:  []string{},
	}

	logger := &MockAudioLogger{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor := audio.NewFFmpegProcessor(config, logger)
		_ = processor
	}
}
