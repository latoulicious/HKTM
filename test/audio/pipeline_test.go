package audio_test

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/audio"
)

// Mock implementations for testing AudioPipelineController

// MockStreamProcessor implements StreamProcessor interface
type MockStreamProcessor struct {
	mu              sync.Mutex
	running         bool
	startStreamErr  error
	stopErr         error
	stream          io.ReadCloser
	processInfo     map[string]interface{}
	waitForExitErr  error
	restartErr      error
}

func NewMockStreamProcessor() *MockStreamProcessor {
	return &MockStreamProcessor{
		processInfo: make(map[string]interface{}),
		stream:      &MockReadCloser{data: []byte("mock audio data")},
	}
}

func (m *MockStreamProcessor) StartStream(url string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.startStreamErr != nil {
		return nil, m.startStreamErr
	}
	
	m.running = true
	return m.stream, nil
}

func (m *MockStreamProcessor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.stopErr != nil {
		return m.stopErr
	}
	
	m.running = false
	return nil
}

func (m *MockStreamProcessor) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *MockStreamProcessor) Restart(url string) error {
	return m.restartErr
}

func (m *MockStreamProcessor) WaitForExit(timeout time.Duration) error {
	return m.waitForExitErr
}

func (m *MockStreamProcessor) GetProcessInfo() map[string]interface{} {
	return m.processInfo
}

// MockAudioEncoder implements AudioEncoder interface
type MockAudioEncoder struct {
	mu                      sync.Mutex
	initialized             bool
	initializeErr           error
	encodeErr               error
	closeErr                error
	frameSize               int
	frameDuration           time.Duration
	validateFrameSizeErr    error
	prepareForStreamingErr  error
}

func NewMockAudioEncoder() *MockAudioEncoder {
	return &MockAudioEncoder{
		frameSize:     960,
		frameDuration: 20 * time.Millisecond,
	}
}

func (m *MockAudioEncoder) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.initializeErr != nil {
		return m.initializeErr
	}
	
	m.initialized = true
	return nil
}

func (m *MockAudioEncoder) Encode(pcmData []int16) ([]byte, error) {
	if m.encodeErr != nil {
		return nil, m.encodeErr
	}
	return []byte("mock opus data"), nil
}

func (m *MockAudioEncoder) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closeErr != nil {
		return m.closeErr
	}
	
	m.initialized = false
	return nil
}

func (m *MockAudioEncoder) EncodeFrame(pcmFrame []int16) ([]byte, error) {
	return m.Encode(pcmFrame)
}

func (m *MockAudioEncoder) GetFrameSize() int {
	return m.frameSize
}

func (m *MockAudioEncoder) GetFrameDuration() time.Duration {
	return m.frameDuration
}

func (m *MockAudioEncoder) ValidateFrameSize(pcmData []int16) error {
	return m.validateFrameSizeErr
}

func (m *MockAudioEncoder) PrepareForStreaming() error {
	return m.prepareForStreamingErr
}

// MockErrorHandler implements ErrorHandler interface
type MockErrorHandler struct {
	handleErrorResult       bool
	handleErrorDelay        time.Duration
	isRetryableResult       bool
	retryDelay              time.Duration
	maxRetries              int
	shouldRetryAfterResult  bool
	logErrorCalls           []LogErrorCall
	notifyRetryCalls        []NotifyRetryCall
	notifyMaxRetriesCalls   []NotifyMaxRetriesCall
}

type LogErrorCall struct {
	Error   error
	Context string
}

type NotifyRetryCall struct {
	Attempt int
	Error   error
	Delay   time.Duration
}

type NotifyMaxRetriesCall struct {
	FinalErr error
	Attempts int
}

func NewMockErrorHandler() *MockErrorHandler {
	return &MockErrorHandler{
		handleErrorResult:      true,
		handleErrorDelay:       2 * time.Second,
		isRetryableResult:      true,
		retryDelay:             2 * time.Second,
		maxRetries:             3,
		shouldRetryAfterResult: true,
	}
}

func (m *MockErrorHandler) HandleError(err error, context string) (bool, time.Duration) {
	return m.handleErrorResult, m.handleErrorDelay
}

func (m *MockErrorHandler) LogError(err error, context string) {
	m.logErrorCalls = append(m.logErrorCalls, LogErrorCall{Error: err, Context: context})
}

func (m *MockErrorHandler) IsRetryableError(err error) bool {
	return m.isRetryableResult
}

func (m *MockErrorHandler) GetRetryDelay(attempt int) time.Duration {
	return m.retryDelay
}

func (m *MockErrorHandler) GetMaxRetries() int {
	return m.maxRetries
}

func (m *MockErrorHandler) ShouldRetryAfterAttempts(attempts int, err error) bool {
	return m.shouldRetryAfterResult
}

func (m *MockErrorHandler) SetNotifier(notifier audio.UserNotifier, channelID string) {}

func (m *MockErrorHandler) DisableNotifications() {}

func (m *MockErrorHandler) NotifyRetryAttempt(attempt int, err error, delay time.Duration) {
	m.notifyRetryCalls = append(m.notifyRetryCalls, NotifyRetryCall{
		Attempt: attempt,
		Error:   err,
		Delay:   delay,
	})
}

func (m *MockErrorHandler) NotifyMaxRetriesExceeded(finalErr error, attempts int) {
	m.notifyMaxRetriesCalls = append(m.notifyMaxRetriesCalls, NotifyMaxRetriesCall{
		FinalErr: finalErr,
		Attempts: attempts,
	})
}

// MockMetricsCollector implements MetricsCollector interface
type MockMetricsCollector struct {
	startupTimes      []time.Duration
	errors            []string
	playbackDurations []time.Duration
	stats             audio.MetricsStats
}

func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		stats: audio.MetricsStats{
			SuccessfulPlays:      0,
			ErrorCount:           0,
			AverageStartupTime:   0,
			TotalPlaybackTime:    0,
		},
	}
}

func (m *MockMetricsCollector) RecordStartupTime(duration time.Duration) {
	m.startupTimes = append(m.startupTimes, duration)
}

func (m *MockMetricsCollector) RecordError(errorType string) {
	m.errors = append(m.errors, errorType)
}

func (m *MockMetricsCollector) RecordPlaybackDuration(duration time.Duration) {
	m.playbackDurations = append(m.playbackDurations, duration)
}

func (m *MockMetricsCollector) GetStats() audio.MetricsStats {
	return m.stats
}

// MockConfigProvider implements ConfigProvider interface
type MockConfigProvider struct {
	validateErr           error
	validateDependenciesErr error
	retryConfig           *audio.RetryConfig
}

func NewMockConfigProvider() *MockConfigProvider {
	return &MockConfigProvider{
		retryConfig: &audio.RetryConfig{
			MaxRetries: 3,
			BaseDelay:  2 * time.Second,
			MaxDelay:   30 * time.Second,
			Multiplier: 2.0,
		},
	}
}

func (m *MockConfigProvider) GetPipelineConfig() *audio.PipelineConfig {
	return &audio.PipelineConfig{}
}

func (m *MockConfigProvider) GetFFmpegConfig() *audio.FFmpegConfig {
	return &audio.FFmpegConfig{}
}

func (m *MockConfigProvider) GetYtDlpConfig() *audio.YtDlpConfig {
	return &audio.YtDlpConfig{}
}

func (m *MockConfigProvider) GetOpusConfig() *audio.OpusConfig {
	return &audio.OpusConfig{}
}

func (m *MockConfigProvider) GetRetryConfig() *audio.RetryConfig {
	return m.retryConfig
}

func (m *MockConfigProvider) GetLoggerConfig() *audio.LoggerConfig {
	return &audio.LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: true,
	}
}

func (m *MockConfigProvider) Validate() error {
	return m.validateErr
}

func (m *MockConfigProvider) ValidateDependencies() error {
	return m.validateDependenciesErr
}

// MockAudioLogger implements AudioLogger interface for testing
type MockAudioLogger struct {
	InfoCalls  []LogCall
	ErrorCalls []ErrorCall
	WarnCalls  []LogCall
	DebugCalls []LogCall
}

type LogCall struct {
	Message string
	Fields  map[string]interface{}
}

type ErrorCall struct {
	Message string
	Error   error
	Fields  map[string]interface{}
}

func (m *MockAudioLogger) Info(msg string, fields map[string]interface{}) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockAudioLogger) Error(msg string, err error, fields map[string]interface{}) {
	m.ErrorCalls = append(m.ErrorCalls, ErrorCall{Message: msg, Error: err, Fields: fields})
}

func (m *MockAudioLogger) Warn(msg string, fields map[string]interface{}) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockAudioLogger) Debug(msg string, fields map[string]interface{}) {
	m.DebugCalls = append(m.DebugCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockAudioLogger) WithPipeline(pipeline string) audio.AudioLogger {
	return m
}

func (m *MockAudioLogger) WithContext(ctx map[string]interface{}) audio.AudioLogger {
	return m
}

// MockReadCloser implements io.ReadCloser for testing
type MockReadCloser struct {
	data   []byte
	pos    int
	closed bool
}

func (m *MockReadCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, errors.New("stream closed")
	}
	
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockReadCloser) Close() error {
	m.closed = true
	return nil
}

// Test helper functions
func createTestPipelineController() (*audio.AudioPipelineController, *MockStreamProcessor, *MockAudioEncoder, *MockErrorHandler, *MockMetricsCollector, *MockAudioLogger, *MockConfigProvider) {
	streamProcessor := NewMockStreamProcessor()
	audioEncoder := NewMockAudioEncoder()
	errorHandler := NewMockErrorHandler()
	metrics := NewMockMetricsCollector()
	logger := &MockAudioLogger{}
	config := NewMockConfigProvider()
	
	controller := audio.NewAudioPipelineController(
		streamProcessor,
		audioEncoder,
		errorHandler,
		metrics,
		logger,
		config,
	)
	
	return controller, streamProcessor, audioEncoder, errorHandler, metrics, logger, config
}

func createMockVoiceConnection() *discordgo.VoiceConnection {
	return &discordgo.VoiceConnection{
		GuildID:   "test-guild-123",
		ChannelID: "test-channel-456",
		OpusSend:  make(chan []byte, 100),
	}
}

// Test AudioPipelineController basic functionality

func TestAudioPipelineController_Initialize(t *testing.T) {
	controller, _, encoder, _, _, logger, config := createTestPipelineController()
	
	// Test successful initialization
	err := controller.Initialize()
	if err != nil {
		t.Errorf("Initialize() failed: %v", err)
	}
	
	if !controller.IsInitialized() {
		t.Error("Controller should be initialized after successful Initialize() call")
	}
	
	// Verify encoder was initialized
	if !encoder.initialized {
		t.Error("Audio encoder should be initialized")
	}
	
	// Verify logger was called
	if len(logger.InfoCalls) == 0 {
		t.Error("Expected initialization to be logged")
	}
	
	// Test double initialization (should not error)
	err = controller.Initialize()
	if err != nil {
		t.Errorf("Double initialization should not error: %v", err)
	}
}

func TestAudioPipelineController_Initialize_ConfigValidationFailure(t *testing.T) {
	controller, _, _, _, _, _, config := createTestPipelineController()
	
	// Set config validation to fail
	config.validateErr = errors.New("invalid config")
	
	err := controller.Initialize()
	if err == nil {
		t.Error("Initialize() should fail when config validation fails")
	}
	
	if controller.IsInitialized() {
		t.Error("Controller should not be initialized after config validation failure")
	}
	
	if !strings.Contains(err.Error(), "configuration validation failed") {
		t.Errorf("Error should mention configuration validation failure: %v", err)
	}
}

func TestAudioPipelineController_Initialize_DependencyValidationFailure(t *testing.T) {
	controller, _, _, _, _, _, config := createTestPipelineController()
	
	// Set dependency validation to fail
	config.validateDependenciesErr = errors.New("ffmpeg not found")
	
	err := controller.Initialize()
	if err == nil {
		t.Error("Initialize() should fail when dependency validation fails")
	}
	
	if controller.IsInitialized() {
		t.Error("Controller should not be initialized after dependency validation failure")
	}
	
	if !strings.Contains(err.Error(), "dependency validation failed") {
		t.Errorf("Error should mention dependency validation failure: %v", err)
	}
}

func TestAudioPipelineController_Initialize_EncoderInitializationFailure(t *testing.T) {
	controller, _, encoder, _, _, _, _ := createTestPipelineController()
	
	// Set encoder initialization to fail
	encoder.initializeErr = errors.New("encoder init failed")
	
	err := controller.Initialize()
	if err == nil {
		t.Error("Initialize() should fail when encoder initialization fails")
	}
	
	if controller.IsInitialized() {
		t.Error("Controller should not be initialized after encoder initialization failure")
	}
	
	if !strings.Contains(err.Error(), "audio encoder initialization failed") {
		t.Errorf("Error should mention encoder initialization failure: %v", err)
	}
}

func TestAudioPipelineController_PlayURL_NotInitialized(t *testing.T) {
	controller, _, _, _, _, _, _ := createTestPipelineController()
	voiceConn := createMockVoiceConnection()
	
	// Try to play without initializing
	err := controller.PlayURL("https://youtube.com/watch?v=test", voiceConn)
	if err == nil {
		t.Error("PlayURL() should fail when controller is not initialized")
	}
	
	if !strings.Contains(err.Error(), "pipeline not initialized") {
		t.Errorf("Error should mention pipeline not initialized: %v", err)
	}
}

func TestAudioPipelineController_PlayURL_InvalidURL(t *testing.T) {
	controller, _, _, _, _, _, _ := createTestPipelineController()
	voiceConn := createMockVoiceConnection()
	
	// Initialize first
	controller.Initialize()
	
	// Try to play with invalid URL
	err := controller.PlayURL("", voiceConn)
	if err == nil {
		t.Error("PlayURL() should fail with empty URL")
	}
	
	if !strings.Contains(err.Error(), "invalid URL") {
		t.Errorf("Error should mention invalid URL: %v", err)
	}
}

func TestAudioPipelineController_PlayURL_AlreadyPlaying(t *testing.T) {
	controller, _, _, _, _, _, _ := createTestPipelineController()
	voiceConn := createMockVoiceConnection()
	
	// Initialize and start playing
	controller.Initialize()
	controller.PlayURL("https://youtube.com/watch?v=test1", voiceConn)
	
	// Try to play another URL while already playing
	err := controller.PlayURL("https://youtube.com/watch?v=test2", voiceConn)
	if err == nil {
		t.Error("PlayURL() should fail when already playing")
	}
	
	if !strings.Contains(err.Error(), "already playing") {
		t.Errorf("Error should mention already playing: %v", err)
	}
}

func TestAudioPipelineController_PlayURL_Success(t *testing.T) {
	controller, streamProcessor, encoder, _, metrics, logger, _ := createTestPipelineController()
	voiceConn := createMockVoiceConnection()
	
	// Initialize
	controller.Initialize()
	
	// Test successful playback start
	err := controller.PlayURL("https://youtube.com/watch?v=test", voiceConn)
	if err != nil {
		t.Errorf("PlayURL() should succeed: %v", err)
	}
	
	// Verify state
	if !controller.IsPlaying() {
		t.Error("Controller should be playing after successful PlayURL()")
	}
	
	status := controller.GetStatus()
	if status.CurrentURL != "https://youtube.com/watch?v=test" {
		t.Errorf("Status should show current URL: %v", status.CurrentURL)
	}
	
	// Verify components were called
	if !streamProcessor.IsRunning() {
		t.Error("Stream processor should be running")
	}
	
	if !encoder.initialized {
		t.Error("Audio encoder should be initialized")
	}
	
	// Verify metrics were recorded
	if len(metrics.startupTimes) == 0 {
		t.Error("Startup time should be recorded")
	}
	
	// Verify logging
	if len(logger.InfoCalls) == 0 {
		t.Error("Playback start should be logged")
	}
}

func TestAudioPipelineController_PlayURL_StreamProcessorFailure(t *testing.T) {
	controller, streamProcessor, _, errorHandler, _, _, _ := createTestPipelineController()
	voiceConn := createMockVoiceConnection()
	
	// Initialize
	controller.Initialize()
	
	// Set stream processor to fail
	streamProcessor.startStreamErr = errors.New("stream start failed")
	
	// Test playback failure
	err := controller.PlayURL("https://youtube.com/watch?v=test", voiceConn)
	if err == nil {
		t.Error("PlayURL() should fail when stream processor fails")
	}
	
	// Verify error handler was called
	if len(errorHandler.logErrorCalls) == 0 {
		t.Error("Error should be logged through error handler")
	}
}

func TestAudioPipelineController_PlayURL_EncoderPreparationFailure(t *testing.T) {
	controller, _, encoder, errorHandler, _, _, _ := createTestPipelineController()
	voiceConn := createMockVoiceConnection()
	
	// Initialize
	controller.Initialize()
	
	// Set encoder preparation to fail
	encoder.prepareForStreamingErr = errors.New("encoder preparation failed")
	
	// Test playback failure
	err := controller.PlayURL("https://youtube.com/watch?v=test", voiceConn)
	if err == nil {
		t.Error("PlayURL() should fail when encoder preparation fails")
	}
	
	// Verify error handler was called
	if len(errorHandler.logErrorCalls) == 0 {
		t.Error("Error should be logged through error handler")
	}
}

func TestAudioPipelineController_Stop_Success(t *testing.T) {
	controller, streamProcessor, encoder, _, _, logger, _ := createTestPipelineController()
	voiceConn := createMockVoiceConnection()
	
	// Initialize and start playing
	controller.Initialize()
	controller.PlayURL("https://youtube.com/watch?v=test", voiceConn)
	
	// Stop playback
	err := controller.Stop()
	if err != nil {
		t.Errorf("Stop() should succeed: %v", err)
	}
	
	// Verify state
	if controller.IsPlaying() {
		t.Error("Controller should not be playing after Stop()")
	}
	
	status := controller.GetStatus()
	if status.CurrentURL != "" {
		t.Error("Status should not show current URL after stop")
	}
	
	// Verify components were stopped
	if streamProcessor.IsRunning() {
		t.Error("Stream processor should not be running after stop")
	}
	
	// Verify logging
	foundStopLog := false
	for _, call := range logger.InfoCalls {
		if strings.Contains(call.Message, "Stopping") || strings.Contains(call.Message, "stopped") {
			foundStopLog = true
			break
		}
	}
	if !foundStopLog {
		t.Error("Stop should be logged")
	}
}

func TestAudioPipelineController_Stop_WhenNotPlaying(t *testing.T) {
	controller, _, _, _, _, _, _ := createTestPipelineController()
	
	// Initialize but don't start playing
	controller.Initialize()
	
	// Stop when not playing (should not error)
	err := controller.Stop()
	if err != nil {
		t.Errorf("Stop() should not error when not playing: %v", err)
	}
}

func TestAudioPipelineController_Shutdown_Success(t *testing.T) {
	controller, streamProcessor, encoder, _, _, logger, _ := createTestPipelineController()
	voiceConn := createMockVoiceConnection()
	
	// Initialize and start playing
	controller.Initialize()
	controller.PlayURL("https://youtube.com/watch?v=test", voiceConn)
	
	// Shutdown
	err := controller.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() should succeed: %v", err)
	}
	
	// Verify state
	if controller.IsInitialized() {
		t.Error("Controller should not be initialized after shutdown")
	}
	
	if controller.IsPlaying() {
		t.Error("Controller should not be playing after shutdown")
	}
	
	// Verify components were shut down
	if streamProcessor.IsRunning() {
		t.Error("Stream processor should not be running after shutdown")
	}
	
	if encoder.initialized {
		t.Error("Audio encoder should not be initialized after shutdown")
	}
	
	// Verify logging
	foundShutdownLog := false
	for _, call := range logger.InfoCalls {
		if strings.Contains(call.Message, "Shutting down") || strings.Contains(call.Message, "shutdown") {
			foundShutdownLog = true
			break
		}
	}
	if !foundShutdownLog {
		t.Error("Shutdown should be logged")
	}
}

func TestAudioPipelineController_Shutdown_WithErrors(t *testing.T) {
	controller, streamProcessor, encoder, _, _, logger, _ := createTestPipelineController()
	
	// Initialize
	controller.Initialize()
	
	// Set components to fail during shutdown
	streamProcessor.stopErr = errors.New("stream processor stop failed")
	encoder.closeErr = errors.New("encoder close failed")
	
	// Shutdown (should complete despite errors)
	err := controller.Shutdown()
	if err == nil {
		t.Error("Shutdown() should return error when components fail to shutdown")
	}
	
	// Verify state is still reset despite errors
	if controller.IsInitialized() {
		t.Error("Controller should not be initialized after shutdown, even with errors")
	}
	
	// Verify errors were logged
	foundErrorLog := false
	for _, call := range logger.ErrorCalls {
		if strings.Contains(call.Message, "shutdown") {
			foundErrorLog = true
			break
		}
	}
	if !foundErrorLog {
		t.Error("Shutdown errors should be logged")
	}
}

func TestAudioPipelineController_Shutdown_NotInitialized(t *testing.T) {
	controller, _, _, _, _, _, _ := createTestPipelineController()
	
	// Shutdown without initializing (should not error)
	err := controller.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() should not error when not initialized: %v", err)
	}
}

func TestAudioPipelineController_GetStatus(t *testing.T) {
	controller, _, _, _, _, _, _ := createTestPipelineController()
	voiceConn := createMockVoiceConnection()
	
	// Test status when not initialized
	status := controller.GetStatus()
	if status.IsPlaying {
		t.Error("Status should show not playing when not initialized")
	}
	if status.CurrentURL != "" {
		t.Error("Status should show empty URL when not initialized")
	}
	
	// Initialize and start playing
	controller.Initialize()
	controller.PlayURL("https://youtube.com/watch?v=test", voiceConn)
	
	// Test status when playing
	status = controller.GetStatus()
	if !status.IsPlaying {
		t.Error("Status should show playing when playing")
	}
	if status.CurrentURL != "https://youtube.com/watch?v=test" {
		t.Errorf("Status should show current URL: %v", status.CurrentURL)
	}
	if status.StartTime.IsZero() {
		t.Error("Status should show start time when playing")
	}
}

func TestAudioPipelineController_StateManagement(t *testing.T) {
	controller, _, _, _, _, _, _ := createTestPipelineController()
	voiceConn := createMockVoiceConnection()
	
	// Test initial state
	if controller.IsPlaying() {
		t.Error("Controller should not be playing initially")
	}
	
	if controller.IsInitialized() {
		t.Error("Controller should not be initialized initially")
	}
	
	// Test after initialization
	controller.Initialize()
	if !controller.IsInitialized() {
		t.Error("Controller should be initialized after Initialize()")
	}
	
	if controller.IsPlaying() {
		t.Error("Controller should not be playing after just Initialize()")
	}
	
	// Test after starting playback
	controller.PlayURL("https://youtube.com/watch?v=test", voiceConn)
	if !controller.IsPlaying() {
		t.Error("Controller should be playing after PlayURL()")
	}
	
	// Test after stopping
	controller.Stop()
	if controller.IsPlaying() {
		t.Error("Controller should not be playing after Stop()")
	}
	
	if !controller.IsInitialized() {
		t.Error("Controller should still be initialized after Stop()")
	}
	
	// Test after shutdown
	controller.Shutdown()
	if controller.IsInitialized() {
		t.Error("Controller should not be initialized after Shutdown()")
	}
	
	if controller.IsPlaying() {
		t.Error("Controller should not be playing after Shutdown()")
	}
}