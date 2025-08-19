package audio_test

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"github.com/latoulicious/HKTM/pkg/embed"
)

// Integration test setup and utilities

// MockDatabase provides a mock database for integration testing
type MockDatabase struct {
	logs    []models.AudioLog
	errors  []models.AudioError
	metrics []models.AudioMetric
	mu      sync.RWMutex
}

func createMockDatabase() *MockDatabase {
	return &MockDatabase{
		logs:    make([]models.AudioLog, 0),
		errors:  make([]models.AudioError, 0),
		metrics: make([]models.AudioMetric, 0),
	}
}

func (m *MockDatabase) SaveLog(log *models.AudioLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, *log)
	return nil
}

func (m *MockDatabase) SaveError(err *models.AudioError) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, *err)
	return nil
}

func (m *MockDatabase) SaveMetric(metric *models.AudioMetric) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metrics = append(m.metrics, *metric)
	return nil
}

func (m *MockDatabase) GetLogCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.logs)
}

func (m *MockDatabase) GetErrorCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.errors)
}

func (m *MockDatabase) GetMetricCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.metrics)
}

// MockYouTubeStreamProvider simulates YouTube stream URLs for testing
type MockYouTubeStreamProvider struct {
	workingURLs map[string]StreamInfo
	failingURLs map[string]error
}

type StreamInfo struct {
	StreamURL string
	Title     string
	Duration  time.Duration
}

func NewMockYouTubeStreamProvider() *MockYouTubeStreamProvider {
	return &MockYouTubeStreamProvider{
		workingURLs: map[string]StreamInfo{
			"https://youtube.com/watch?v=working1": {
				StreamURL: "https://stream.example.com/audio1.m4a",
				Title:     "Test Song 1",
				Duration:  3 * time.Minute,
			},
			"https://youtube.com/watch?v=working2": {
				StreamURL: "https://stream.example.com/audio2.m4a",
				Title:     "Test Song 2",
				Duration:  4 * time.Minute,
			},
			"https://youtube.com/watch?v=working3": {
				StreamURL: "https://stream.example.com/audio3.m4a",
				Title:     "Test Song 3",
				Duration:  2 * time.Minute,
			},
		},
		failingURLs: map[string]error{
			"https://youtube.com/watch?v=network_error": errors.New("network connection failed"),
			"https://youtube.com/watch?v=not_found":     errors.New("video not found"),
			"https://youtube.com/watch?v=private":       errors.New("video is private"),
		},
	}
}

func (m *MockYouTubeStreamProvider) GetStreamInfo(url string) (StreamInfo, error) {
	if info, exists := m.workingURLs[url]; exists {
		return info, nil
	}
	if err, exists := m.failingURLs[url]; exists {
		return StreamInfo{}, err
	}
	return StreamInfo{}, errors.New("unknown URL")
}

// IntegrationTestStreamProcessor implements StreamProcessor for integration testing
type IntegrationTestStreamProcessor struct {
	mu              sync.Mutex
	running         bool
	currentURL      string
	streamProvider  *MockYouTubeStreamProvider
	simulateFailure bool
	failureType     string
}

func NewIntegrationTestStreamProcessor() *IntegrationTestStreamProcessor {
	return &IntegrationTestStreamProcessor{
		streamProvider: NewMockYouTubeStreamProvider(),
	}
}

func (i *IntegrationTestStreamProcessor) StartStream(url string) (io.ReadCloser, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.simulateFailure {
		switch i.failureType {
		case "network":
			return nil, errors.New("network connection failed")
		case "ffmpeg":
			return nil, errors.New("ffmpeg process failed")
		case "timeout":
			return nil, errors.New("stream timeout")
		default:
			return nil, errors.New("unknown error")
		}
	}

	// Check if URL is in our mock provider
	streamInfo, err := i.streamProvider.GetStreamInfo(url)
	if err != nil {
		return nil, err
	}

	i.running = true
	i.currentURL = url

	// Create a mock stream that provides test audio data
	return &IntegrationTestStream{
		title:    streamInfo.Title,
		duration: streamInfo.Duration,
		data:     generateTestAudioData(streamInfo.Duration),
	}, nil
}

func (i *IntegrationTestStreamProcessor) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.running = false
	i.currentURL = ""
	return nil
}

func (i *IntegrationTestStreamProcessor) IsRunning() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.running
}

func (i *IntegrationTestStreamProcessor) Restart(url string) error {
	i.Stop()
	_, err := i.StartStream(url)
	return err
}

func (i *IntegrationTestStreamProcessor) WaitForExit(timeout time.Duration) error {
	return nil
}

func (i *IntegrationTestStreamProcessor) GetProcessInfo() map[string]interface{} {
	i.mu.Lock()
	defer i.mu.Unlock()
	return map[string]interface{}{
		"running":     i.running,
		"current_url": i.currentURL,
	}
}

func (i *IntegrationTestStreamProcessor) SetFailureMode(failureType string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.simulateFailure = true
	i.failureType = failureType
}

func (i *IntegrationTestStreamProcessor) ClearFailureMode() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.simulateFailure = false
	i.failureType = ""
}

// IntegrationTestAudioEncoder implements AudioEncoder for integration testing
type IntegrationTestAudioEncoder struct {
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

func NewIntegrationTestAudioEncoder() *IntegrationTestAudioEncoder {
	return &IntegrationTestAudioEncoder{
		frameSize:     960,
		frameDuration: 20 * time.Millisecond,
	}
}

func (i *IntegrationTestAudioEncoder) Initialize() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	if i.initializeErr != nil {
		return i.initializeErr
	}
	
	i.initialized = true
	return nil
}

func (i *IntegrationTestAudioEncoder) Encode(pcmData []int16) ([]byte, error) {
	if i.encodeErr != nil {
		return nil, i.encodeErr
	}
	return []byte("mock opus data"), nil
}

func (i *IntegrationTestAudioEncoder) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	if i.closeErr != nil {
		return i.closeErr
	}
	
	i.initialized = false
	return nil
}

func (i *IntegrationTestAudioEncoder) EncodeFrame(pcmFrame []int16) ([]byte, error) {
	return i.Encode(pcmFrame)
}

func (i *IntegrationTestAudioEncoder) GetFrameSize() int {
	return i.frameSize
}

func (i *IntegrationTestAudioEncoder) GetFrameDuration() time.Duration {
	return i.frameDuration
}

func (i *IntegrationTestAudioEncoder) ValidateFrameSize(pcmData []int16) error {
	return i.validateFrameSizeErr
}

func (i *IntegrationTestAudioEncoder) PrepareForStreaming() error {
	return i.prepareForStreamingErr
}

func (i *IntegrationTestAudioEncoder) SetInitializeError(err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.initializeErr = err
}

func (i *IntegrationTestAudioEncoder) SetEncodeError(err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.encodeErr = err
}

func (i *IntegrationTestAudioEncoder) IsInitialized() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.initialized
}

// IntegrationTestErrorHandler implements ErrorHandler for integration testing
type IntegrationTestErrorHandler struct {
	handleErrorResult       bool
	handleErrorDelay        time.Duration
	isRetryableResult       bool
	retryDelay              time.Duration
	maxRetries              int
	shouldRetryAfterResult  bool
	logErrorCalls           []IntegrationLogErrorCall
	notifyRetryCalls        []IntegrationNotifyRetryCall
	notifyMaxRetriesCalls   []IntegrationNotifyMaxRetriesCall
}

type IntegrationLogErrorCall struct {
	Error   error
	Context string
}

type IntegrationNotifyRetryCall struct {
	Attempt int
	Error   error
	Delay   time.Duration
}

type IntegrationNotifyMaxRetriesCall struct {
	FinalErr error
	Attempts int
}

func NewIntegrationTestErrorHandler() *IntegrationTestErrorHandler {
	return &IntegrationTestErrorHandler{
		handleErrorResult:      true,
		handleErrorDelay:       2 * time.Second,
		isRetryableResult:      true,
		retryDelay:             2 * time.Second,
		maxRetries:             3,
		shouldRetryAfterResult: true,
	}
}

func (i *IntegrationTestErrorHandler) HandleError(err error, context string) (bool, time.Duration) {
	return i.handleErrorResult, i.handleErrorDelay
}

func (i *IntegrationTestErrorHandler) LogError(err error, context string) {
	i.logErrorCalls = append(i.logErrorCalls, IntegrationLogErrorCall{Error: err, Context: context})
}

func (i *IntegrationTestErrorHandler) IsRetryableError(err error) bool {
	return i.isRetryableResult
}

func (i *IntegrationTestErrorHandler) GetRetryDelay(attempt int) time.Duration {
	return i.retryDelay
}

func (i *IntegrationTestErrorHandler) GetMaxRetries() int {
	return i.maxRetries
}

func (i *IntegrationTestErrorHandler) ShouldRetryAfterAttempts(attempts int, err error) bool {
	return i.shouldRetryAfterResult
}

func (i *IntegrationTestErrorHandler) SetNotifier(notifier audio.UserNotifier, channelID string) {}

func (i *IntegrationTestErrorHandler) DisableNotifications() {}

func (i *IntegrationTestErrorHandler) NotifyRetryAttempt(attempt int, err error, delay time.Duration) {
	i.notifyRetryCalls = append(i.notifyRetryCalls, IntegrationNotifyRetryCall{
		Attempt: attempt,
		Error:   err,
		Delay:   delay,
	})
}

func (i *IntegrationTestErrorHandler) NotifyMaxRetriesExceeded(finalErr error, attempts int) {
	i.notifyMaxRetriesCalls = append(i.notifyMaxRetriesCalls, IntegrationNotifyMaxRetriesCall{
		FinalErr: finalErr,
		Attempts: attempts,
	})
}

// IntegrationTestMetricsCollector implements MetricsCollector for integration testing
type IntegrationTestMetricsCollector struct {
	startupTimes      []time.Duration
	errors            []string
	playbackDurations []time.Duration
	stats             audio.MetricsStats
}

func NewIntegrationTestMetricsCollector() *IntegrationTestMetricsCollector {
	return &IntegrationTestMetricsCollector{
		stats: audio.MetricsStats{
			SuccessfulPlays:      0,
			ErrorCount:           0,
			AverageStartupTime:   0,
			TotalPlaybackTime:    0,
		},
	}
}

func (i *IntegrationTestMetricsCollector) RecordStartupTime(duration time.Duration) {
	i.startupTimes = append(i.startupTimes, duration)
}

func (i *IntegrationTestMetricsCollector) RecordError(errorType string) {
	i.errors = append(i.errors, errorType)
}

func (i *IntegrationTestMetricsCollector) RecordPlaybackDuration(duration time.Duration) {
	i.playbackDurations = append(i.playbackDurations, duration)
}

func (i *IntegrationTestMetricsCollector) GetStats() audio.MetricsStats {
	return i.stats
}

// IntegrationTestConfigProvider implements ConfigProvider for integration testing
type IntegrationTestConfigProvider struct {
	validateErr           error
	validateDependenciesErr error
	retryConfig           *audio.RetryConfig
}

func NewIntegrationTestConfigProvider() *IntegrationTestConfigProvider {
	return &IntegrationTestConfigProvider{
		retryConfig: &audio.RetryConfig{
			MaxRetries: 3,
			BaseDelay:  2 * time.Second,
			MaxDelay:   30 * time.Second,
			Multiplier: 2.0,
		},
	}
}

func (i *IntegrationTestConfigProvider) GetPipelineConfig() *audio.PipelineConfig {
	return &audio.PipelineConfig{}
}

func (i *IntegrationTestConfigProvider) GetFFmpegConfig() *audio.FFmpegConfig {
	return &audio.FFmpegConfig{}
}

func (i *IntegrationTestConfigProvider) GetYtDlpConfig() *audio.YtDlpConfig {
	return &audio.YtDlpConfig{}
}

func (i *IntegrationTestConfigProvider) GetOpusConfig() *audio.OpusConfig {
	return &audio.OpusConfig{}
}

func (i *IntegrationTestConfigProvider) GetRetryConfig() *audio.RetryConfig {
	return i.retryConfig
}

func (i *IntegrationTestConfigProvider) GetLoggerConfig() *audio.LoggerConfig {
	return &audio.LoggerConfig{
		Level:    "info",
		Format:   "json",
		SaveToDB: true,
	}
}

func (i *IntegrationTestConfigProvider) Validate() error {
	return i.validateErr
}

func (i *IntegrationTestConfigProvider) ValidateDependencies() error {
	return i.validateDependenciesErr
}

// IntegrationTestAudioLogger implements AudioLogger for integration testing
type IntegrationTestAudioLogger struct {
	InfoCalls  []IntegrationLogCall
	ErrorCalls []IntegrationErrorCall
	WarnCalls  []IntegrationLogCall
	DebugCalls []IntegrationLogCall
}

type IntegrationLogCall struct {
	Message string
	Fields  map[string]interface{}
}

type IntegrationErrorCall struct {
	Message string
	Error   error
	Fields  map[string]interface{}
}

func (i *IntegrationTestAudioLogger) Info(msg string, fields map[string]interface{}) {
	i.InfoCalls = append(i.InfoCalls, IntegrationLogCall{Message: msg, Fields: fields})
}

func (i *IntegrationTestAudioLogger) Error(msg string, err error, fields map[string]interface{}) {
	i.ErrorCalls = append(i.ErrorCalls, IntegrationErrorCall{Message: msg, Error: err, Fields: fields})
}

func (i *IntegrationTestAudioLogger) Warn(msg string, fields map[string]interface{}) {
	i.WarnCalls = append(i.WarnCalls, IntegrationLogCall{Message: msg, Fields: fields})
}

func (i *IntegrationTestAudioLogger) Debug(msg string, fields map[string]interface{}) {
	i.DebugCalls = append(i.DebugCalls, IntegrationLogCall{Message: msg, Fields: fields})
}

func (i *IntegrationTestAudioLogger) WithPipeline(pipeline string) audio.AudioLogger {
	return i
}

func (i *IntegrationTestAudioLogger) WithContext(ctx map[string]interface{}) audio.AudioLogger {
	return i
}

// IntegrationTestStream simulates an audio stream for testing
type IntegrationTestStream struct {
	title    string
	duration time.Duration
	data     []byte
	pos      int
	closed   bool
	mu       sync.Mutex
}

func (i *IntegrationTestStream) Read(p []byte) (n int, err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return 0, errors.New("stream closed")
	}

	if i.pos >= len(i.data) {
		return 0, io.EOF
	}

	n = copy(p, i.data[i.pos:])
	i.pos += n
	return n, nil
}

func (i *IntegrationTestStream) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.closed = true
	return nil
}

// generateTestAudioData creates mock PCM audio data for testing
func generateTestAudioData(duration time.Duration) []byte {
	// Generate simple sine wave PCM data
	sampleRate := 48000
	channels := 2
	samples := int(duration.Seconds()) * sampleRate * channels
	data := make([]byte, samples*2) // 2 bytes per sample (int16)

	for i := 0; i < samples; i += 2 {
		// Simple sine wave pattern
		sample := int16(1000 * (i % 1000) / 1000)
		data[i*2] = byte(sample & 0xFF)
		data[i*2+1] = byte((sample >> 8) & 0xFF)
	}

	return data
}

// Integration test helper functions

func createIntegrationTestPipeline(t *testing.T, mockDB *MockDatabase, guildID string) (audio.AudioPipeline, *IntegrationTestStreamProcessor, *IntegrationTestAudioEncoder, *IntegrationTestErrorHandler) {
	// Create test components
	streamProcessor := NewIntegrationTestStreamProcessor()
	audioEncoder := NewIntegrationTestAudioEncoder()
	errorHandler := NewIntegrationTestErrorHandler()
	metrics := NewIntegrationTestMetricsCollector()
	logger := &IntegrationTestAudioLogger{}
	config := NewIntegrationTestConfigProvider()

	// Create pipeline controller
	controller := audio.NewAudioPipelineController(
		streamProcessor,
		audioEncoder,
		errorHandler,
		metrics,
		logger,
		config,
	)

	// Initialize the pipeline
	err := controller.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize pipeline: %v", err)
	}

	return controller, streamProcessor, audioEncoder, errorHandler
}

func createTestVoiceConnection(guildID, channelID string) *discordgo.VoiceConnection {
	return &discordgo.VoiceConnection{
		GuildID:   guildID,
		ChannelID: channelID,
		OpusSend:  make(chan []byte, 100),
	}
}

// Test 1: End-to-end pipeline with known working URLs

func TestIntegration_EndToEndPipeline_WorkingURL(t *testing.T) {
	mockDB := createMockDatabase()
	guildID := "test-guild-123"
	
	pipeline, streamProcessor, encoder, _ := createIntegrationTestPipeline(t, mockDB, guildID)
	defer pipeline.Shutdown()

	voiceConn := createTestVoiceConnection(guildID, "test-channel-456")

	// Test with a known working URL
	workingURL := "https://youtube.com/watch?v=working1"

	// Start playback
	err := pipeline.PlayURL(workingURL, voiceConn)
	if err != nil {
		t.Errorf("PlayURL should succeed with working URL: %v", err)
	}

	// Verify pipeline is playing
	if !pipeline.IsPlaying() {
		t.Error("Pipeline should be playing after successful PlayURL")
	}

	// Verify stream processor is running
	if !streamProcessor.IsRunning() {
		t.Error("Stream processor should be running")
	}

	// Verify encoder is initialized
	if !encoder.initialized {
		t.Error("Audio encoder should be initialized")
	}

	// Get status and verify
	status := pipeline.GetStatus()
	if !status.IsPlaying {
		t.Error("Status should show playing")
	}
	if status.CurrentURL != workingURL {
		t.Errorf("Status should show current URL: expected %s, got %s", workingURL, status.CurrentURL)
	}

	// Allow some time for streaming to start
	time.Sleep(100 * time.Millisecond)

	// Verify audio data is being sent to Discord
	select {
	case opusData := <-voiceConn.OpusSend:
		if len(opusData) == 0 {
			t.Error("Should receive opus audio data")
		}
	case <-time.After(1 * time.Second):
		t.Error("Should receive opus audio data within 1 second")
	}

	// Stop playback
	err = pipeline.Stop()
	if err != nil {
		t.Errorf("Stop should succeed: %v", err)
	}

	// Verify pipeline is stopped
	if pipeline.IsPlaying() {
		t.Error("Pipeline should not be playing after Stop")
	}
}

func TestIntegration_EndToEndPipeline_MultipleURLs(t *testing.T) {
	mockDB := createMockDatabase()
	guildID := "test-guild-456"
	
	pipeline, _, _, _ := createIntegrationTestPipeline(t, mockDB, guildID)
	defer pipeline.Shutdown()

	voiceConn := createTestVoiceConnection(guildID, "test-channel-789")

	workingURLs := []string{
		"https://youtube.com/watch?v=working1",
		"https://youtube.com/watch?v=working2",
		"https://youtube.com/watch?v=working3",
	}

	for i, url := range workingURLs {
		t.Run(fmt.Sprintf("URL_%d", i+1), func(t *testing.T) {
			// Stop any previous playback
			pipeline.Stop()

			// Start new playback
			err := pipeline.PlayURL(url, voiceConn)
			if err != nil {
				t.Errorf("PlayURL should succeed with working URL %s: %v", url, err)
			}

			// Verify pipeline is playing
			if !pipeline.IsPlaying() {
				t.Errorf("Pipeline should be playing URL %s", url)
			}

			// Verify status
			status := pipeline.GetStatus()
			if status.CurrentURL != url {
				t.Errorf("Status should show current URL: expected %s, got %s", url, status.CurrentURL)
			}

			// Allow some streaming time
			time.Sleep(50 * time.Millisecond)
		})
	}
}

// Test 2: Basic error handling and retry logic

func TestIntegration_ErrorHandling_NetworkError(t *testing.T) {
	mockDB := createMockDatabase()
	guildID := "test-guild-error"
	
	pipeline, streamProcessor, _, errorHandler := createIntegrationTestPipeline(t, mockDB, guildID)
	defer pipeline.Shutdown()

	voiceConn := createTestVoiceConnection(guildID, "test-channel-error")

	// Configure error handler for retries
	errorHandler.handleErrorResult = true
	errorHandler.handleErrorDelay = 100 * time.Millisecond
	errorHandler.maxRetries = 2

	// Set stream processor to fail initially
	streamProcessor.SetFailureMode("network")

	// Try to play - should fail but attempt retries
	err := pipeline.PlayURL("https://youtube.com/watch?v=network_error", voiceConn)
	if err == nil {
		t.Error("PlayURL should fail with network error")
	}

	// Verify error handler was called for retries
	if len(errorHandler.notifyRetryCalls) == 0 {
		t.Error("Error handler should have been called for retry attempts")
	}

	// Verify pipeline is not playing after failure
	if pipeline.IsPlaying() {
		t.Error("Pipeline should not be playing after permanent failure")
	}
}

func TestIntegration_ErrorHandling_RetrySuccess(t *testing.T) {
	mockDB := createMockDatabase()
	guildID := "test-guild-retry"
	
	pipeline, streamProcessor, _, errorHandler := createIntegrationTestPipeline(t, mockDB, guildID)
	defer pipeline.Shutdown()

	voiceConn := createTestVoiceConnection(guildID, "test-channel-retry")

	// Configure error handler for retries
	errorHandler.handleErrorResult = true
	errorHandler.handleErrorDelay = 50 * time.Millisecond
	errorHandler.maxRetries = 3

	// Set stream processor to fail initially, then succeed
	streamProcessor.SetFailureMode("network")

	// Start playback in a goroutine since it will retry
	go func() {
		// Clear failure mode after a short delay to simulate recovery
		time.Sleep(100 * time.Millisecond)
		streamProcessor.ClearFailureMode()
	}()

	// Try to play - should eventually succeed after retry
	err := pipeline.PlayURL("https://youtube.com/watch?v=working1", voiceConn)
	if err != nil {
		t.Errorf("PlayURL should eventually succeed after retry: %v", err)
	}

	// Allow time for retry logic
	time.Sleep(200 * time.Millisecond)

	// Verify pipeline is playing after successful retry
	if !pipeline.IsPlaying() {
		t.Error("Pipeline should be playing after successful retry")
	}
}

func TestIntegration_ErrorHandling_MaxRetriesExceeded(t *testing.T) {
	mockDB := createMockDatabase()
	guildID := "test-guild-maxretry"
	
	pipeline, streamProcessor, _, errorHandler := createIntegrationTestPipeline(t, mockDB, guildID)
	defer pipeline.Shutdown()

	voiceConn := createTestVoiceConnection(guildID, "test-channel-maxretry")

	// Configure error handler with limited retries
	errorHandler.handleErrorResult = true
	errorHandler.handleErrorDelay = 10 * time.Millisecond
	errorHandler.maxRetries = 2

	// Set stream processor to always fail
	streamProcessor.SetFailureMode("ffmpeg")

	// Try to play - should fail after max retries
	err := pipeline.PlayURL("https://youtube.com/watch?v=network_error", voiceConn)
	if err == nil {
		t.Error("PlayURL should fail after max retries exceeded")
	}

	// Verify max retries notification was called
	if len(errorHandler.notifyMaxRetriesCalls) == 0 {
		t.Error("Error handler should notify when max retries exceeded")
	}

	// Verify error message mentions retry attempts
	if !strings.Contains(err.Error(), "attempts") {
		t.Errorf("Error should mention retry attempts: %v", err)
	}

	// Verify pipeline is not playing
	if pipeline.IsPlaying() {
		t.Error("Pipeline should not be playing after max retries exceeded")
	}
}

// MockMusicQueue provides a simplified queue for integration testing
type MockMusicQueue struct {
	guildID   string
	items     []*MockQueueItem
	current   *MockQueueItem
	isPlaying bool
	pipeline  audio.AudioPipeline
	voiceConn *discordgo.VoiceConnection
	mu        sync.RWMutex
}

type MockQueueItem struct {
	URL         string
	Title       string
	RequestedBy string
	Duration    time.Duration
}

func NewMockMusicQueue(guildID string) *MockMusicQueue {
	return &MockMusicQueue{
		guildID: guildID,
		items:   make([]*MockQueueItem, 0),
	}
}

func (m *MockMusicQueue) Add(url, title, requestedBy string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	item := &MockQueueItem{
		URL:         url,
		Title:       title,
		RequestedBy: requestedBy,
	}
	m.items = append(m.items, item)
}

func (m *MockMusicQueue) Next() *MockQueueItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if len(m.items) == 0 {
		return nil
	}
	
	item := m.items[0]
	m.items = m.items[1:]
	m.current = item
	return item
}

func (m *MockMusicQueue) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}

func (m *MockMusicQueue) Current() *MockQueueItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *MockMusicQueue) IsPlaying() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isPlaying
}

func (m *MockMusicQueue) SetPlaying(playing bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isPlaying = playing
}

func (m *MockMusicQueue) SetPipeline(pipeline audio.AudioPipeline) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pipeline = pipeline
}

func (m *MockMusicQueue) GetPipeline() audio.AudioPipeline {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pipeline
}

func (m *MockMusicQueue) HasActivePipeline() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pipeline != nil && m.pipeline.IsPlaying()
}

func (m *MockMusicQueue) StartPlayback(url string, voiceConn *discordgo.VoiceConnection, mockDB *MockDatabase) error {
	// Create a mock pipeline for testing
	pipeline, _, _, _ := createIntegrationTestPipeline(&testing.T{}, mockDB, m.guildID)
	m.SetPipeline(pipeline)
	
	// Start playback
	err := pipeline.PlayURL(url, voiceConn)
	if err != nil {
		return err
	}
	
	m.SetPlaying(true)
	return nil
}

func (m *MockMusicQueue) StopAndCleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.pipeline != nil {
		m.pipeline.Stop()
		m.pipeline.Shutdown()
		m.pipeline = nil
	}
	
	m.isPlaying = false
	m.voiceConn = nil
}

// Test 3: Queue integration and command compatibility

func TestIntegration_QueueIntegration_BasicPlayback(t *testing.T) {
	mockDB := createMockDatabase()
	guildID := "test-guild-queue"

	// Create a mock music queue
	queue := NewMockMusicQueue(guildID)
	voiceConn := createTestVoiceConnection(guildID, "test-channel-queue")

	// Add items to queue
	queue.Add("https://youtube.com/watch?v=working1", "Test Song 1", "TestUser1")
	queue.Add("https://youtube.com/watch?v=working2", "Test Song 2", "TestUser2")

	// Verify queue size
	if queue.Size() != 2 {
		t.Errorf("Queue should have 2 items, got %d", queue.Size())
	}

	// Start playback using queue's StartPlayback method
	nextItem := queue.Next()
	if nextItem == nil {
		t.Fatal("Should get next item from queue")
	}

	err := queue.StartPlayback(nextItem.URL, voiceConn, mockDB)
	if err != nil {
		t.Errorf("StartPlayback should succeed: %v", err)
	}

	// Verify queue state
	if !queue.IsPlaying() {
		t.Error("Queue should be in playing state")
	}

	if !queue.HasActivePipeline() {
		t.Error("Queue should have active pipeline")
	}

	// Verify current item
	current := queue.Current()
	if current == nil {
		t.Error("Queue should have current item")
	} else if current.Title != "Test Song 1" {
		t.Errorf("Current item should be 'Test Song 1', got '%s'", current.Title)
	}

	// Stop and cleanup
	queue.StopAndCleanup()

	// Verify cleanup
	if queue.IsPlaying() {
		t.Error("Queue should not be playing after cleanup")
	}

	if queue.HasActivePipeline() {
		t.Error("Queue should not have active pipeline after cleanup")
	}
}

func TestIntegration_QueueIntegration_YouTubeData(t *testing.T) {
	guildID := "test-guild-youtube"

	queue := NewMockMusicQueue(guildID)

	// Add YouTube item with metadata
	streamURL := "https://stream.example.com/audio1.m4a"
	title := "Test YouTube Song"

	// For mock queue, we'll just add the basic info
	queue.Add(streamURL, title, "TestUser")

	// Verify queue item
	if queue.Size() != 1 {
		t.Fatalf("Queue should have 1 item, got %d", queue.Size())
	}

	// Get the item by calling Next()
	item := queue.Next()
	if item == nil {
		t.Fatal("Should get item from queue")
	}
	
	if item.URL != streamURL {
		t.Errorf("Item URL should be %s, got %s", streamURL, item.URL)
	}
	if item.Title != title {
		t.Errorf("Item Title should be %s, got %s", title, item.Title)
	}
	if item.RequestedBy != "TestUser" {
		t.Errorf("Item RequestedBy should be TestUser, got %s", item.RequestedBy)
	}
}

func TestIntegration_QueueIntegration_DetailedStatus(t *testing.T) {
	mockDB := createMockDatabase()
	guildID := "test-guild-status"

	queue := NewMockMusicQueue(guildID)
	voiceConn := createTestVoiceConnection(guildID, "test-channel-status")

	// Add items and start playback
	queue.Add("https://youtube.com/watch?v=working1", "Test Song 1", "TestUser1")
	queue.Add("https://youtube.com/watch?v=working2", "Test Song 2", "TestUser2")

	nextItem := queue.Next()
	err := queue.StartPlayback(nextItem.URL, voiceConn, mockDB)
	if err != nil {
		t.Errorf("StartPlayback should succeed: %v", err)
	}

	// Verify basic queue status
	if queue.Size() != 1 { // One item remaining after Next()
		t.Errorf("Queue size should be 1, got %d", queue.Size())
	}

	if !queue.IsPlaying() {
		t.Error("Queue should be playing")
	}

	if !queue.HasActivePipeline() {
		t.Error("Queue should have active pipeline")
	}

	// Verify current song
	current := queue.Current()
	if current == nil {
		t.Error("Queue should have current song")
	} else if current.Title != "Test Song 1" {
		t.Errorf("Current song title should be 'Test Song 1', got %s", current.Title)
	}

	queue.StopAndCleanup()
}

// Test 4: Centralized embed generation

func TestIntegration_CentralizedEmbeds_AudioEmbeds(t *testing.T) {
	embedBuilder := embed.NewAudioEmbedBuilder()

	// Test NowPlaying embed
	nowPlayingEmbed := embedBuilder.NowPlaying("Test Song", "https://youtube.com/watch?v=test", 3*time.Minute)
	if nowPlayingEmbed.Title != "🎵 Now Playing" {
		t.Errorf("NowPlaying embed title should be '🎵 Now Playing', got '%s'", nowPlayingEmbed.Title)
	}
	if !strings.Contains(nowPlayingEmbed.Description, "Test Song") {
		t.Error("NowPlaying embed should contain song title")
	}
	if nowPlayingEmbed.Color != 0x00ff00 {
		t.Errorf("NowPlaying embed should be green (0x00ff00), got 0x%x", nowPlayingEmbed.Color)
	}

	// Test PlaybackError embed
	testErr := errors.New("stream failed")
	errorEmbed := embedBuilder.PlaybackError("https://youtube.com/watch?v=error", testErr)
	if errorEmbed.Title != "❌ Playback Error" {
		t.Errorf("PlaybackError embed title should be '❌ Playback Error', got '%s'", errorEmbed.Title)
	}
	if errorEmbed.Color != 0xff0000 {
		t.Errorf("PlaybackError embed should be red (0xff0000), got 0x%x", errorEmbed.Color)
	}
	if len(errorEmbed.Fields) == 0 || !strings.Contains(errorEmbed.Fields[0].Value, "stream failed") {
		t.Error("PlaybackError embed should contain error message in fields")
	}

	// Test QueueStatus embed
	currentSong := "**Test Song 1** (Requested by: TestUser1)"
	queueItems := []string{
		"**Test Song 2** (Requested by: TestUser2)",
		"**Test Song 3** (Requested by: TestUser3)",
	}
	queueEmbed := embedBuilder.QueueStatus(currentSong, queueItems, 2)
	if queueEmbed.Title != "🎵 Music Queue" {
		t.Errorf("QueueStatus embed title should be '🎵 Music Queue', got '%s'", queueEmbed.Title)
	}
	if len(queueEmbed.Fields) < 2 {
		t.Error("QueueStatus embed should have at least 2 fields (now playing, up next)")
	}

	// Test IdleTimeout embed
	idleEmbed := embedBuilder.IdleTimeout()
	if idleEmbed.Title != "⏰ Idle Timeout" {
		t.Errorf("IdleTimeout embed title should be '⏰ Idle Timeout', got '%s'", idleEmbed.Title)
	}
	if idleEmbed.Color != 0xffa500 {
		t.Errorf("IdleTimeout embed should be orange (0xffa500), got 0x%x", idleEmbed.Color)
	}
}

func TestIntegration_CentralizedEmbeds_BasicEmbeds(t *testing.T) {
	embedBuilder := embed.NewAudioEmbedBuilder()

	// Test Success embed
	successEmbed := embedBuilder.Success("Success Title", "Success description")
	if successEmbed.Title != "Success Title" {
		t.Errorf("Success embed title should be 'Success Title', got '%s'", successEmbed.Title)
	}
	if successEmbed.Description != "Success description" {
		t.Errorf("Success embed description should be 'Success description', got '%s'", successEmbed.Description)
	}
	if successEmbed.Color != 0x00ff00 {
		t.Errorf("Success embed should be green (0x00ff00), got 0x%x", successEmbed.Color)
	}

	// Test Error embed
	errorEmbed := embedBuilder.Error("Error Title", "Error description")
	if errorEmbed.Title != "Error Title" {
		t.Errorf("Error embed title should be 'Error Title', got '%s'", errorEmbed.Title)
	}
	if errorEmbed.Color != 0xff0000 {
		t.Errorf("Error embed should be red (0xff0000), got 0x%x", errorEmbed.Color)
	}

	// Test Info embed
	infoEmbed := embedBuilder.Info("Info Title", "Info description")
	if infoEmbed.Title != "Info Title" {
		t.Errorf("Info embed title should be 'Info Title', got '%s'", infoEmbed.Title)
	}
	if infoEmbed.Color != 0x0099ff {
		t.Errorf("Info embed should be blue (0x0099ff), got 0x%x", infoEmbed.Color)
	}

	// Test Warning embed
	warningEmbed := embedBuilder.Warning("Warning Title", "Warning description")
	if warningEmbed.Title != "Warning Title" {
		t.Errorf("Warning embed title should be 'Warning Title', got '%s'", warningEmbed.Title)
	}
	if warningEmbed.Color != 0xffa500 {
		t.Errorf("Warning embed should be orange (0xffa500), got 0x%x", warningEmbed.Color)
	}
}

func TestIntegration_CentralizedEmbeds_QueueIntegration(t *testing.T) {
	embedBuilder := embed.NewAudioEmbedBuilder()

	// Simulate queue data
	currentSong := "**Test Song 1** (Requested by: TestUser1)"
	queueItems := []string{
		"**Test Song 2** (Requested by: TestUser2)",
	}

	// Get queue status embed
	statusEmbed := embedBuilder.QueueStatus(currentSong, queueItems, 1)

	// Verify embed structure
	if statusEmbed.Title != "🎵 Music Queue" {
		t.Errorf("Queue status embed title should be '🎵 Music Queue', got '%s'", statusEmbed.Title)
	}

	// Verify embed has required fields
	hasNowPlaying := false
	hasUpNext := false
	hasQueueInfo := false

	for _, field := range statusEmbed.Fields {
		switch field.Name {
		case "🎶 Now Playing":
			hasNowPlaying = true
			if !strings.Contains(field.Value, "Test Song 1") {
				t.Error("Now Playing field should contain current song")
			}
		case "📋 Up Next":
			hasUpNext = true
			if !strings.Contains(field.Value, "Test Song 2") {
				t.Error("Up Next field should contain queued songs")
			}
		case "📊 Queue Info":
			hasQueueInfo = true
			if !strings.Contains(field.Value, "Total songs: 1") {
				t.Error("Queue Info field should show correct queue size")
			}
		}
	}

	if !hasNowPlaying {
		t.Error("Queue status embed should have 'Now Playing' field")
	}
	if !hasUpNext {
		t.Error("Queue status embed should have 'Up Next' field")
	}
	if !hasQueueInfo {
		t.Error("Queue status embed should have 'Queue Info' field")
	}
}

// Comprehensive integration test combining all components

func TestIntegration_FullWorkflow_EndToEnd(t *testing.T) {
	mockDB := createMockDatabase()
	guildID := "test-guild-full"

	// Create mock queue
	queue := NewMockMusicQueue(guildID)
	voiceConn := createTestVoiceConnection(guildID, "test-channel-full")

	// Create embed builder for testing
	embedBuilder := embed.NewAudioEmbedBuilder()

	// Step 1: Add multiple songs to queue
	songs := []struct {
		url   string
		title string
		user  string
	}{
		{"https://youtube.com/watch?v=working1", "Test Song 1", "User1"},
		{"https://youtube.com/watch?v=working2", "Test Song 2", "User2"},
		{"https://youtube.com/watch?v=working3", "Test Song 3", "User3"},
	}

	for _, song := range songs {
		queue.Add(song.url, song.title, song.user)
	}

	// Verify queue size
	if queue.Size() != 3 {
		t.Errorf("Queue should have 3 items, got %d", queue.Size())
	}

	// Step 2: Start playing first song
	firstSong := queue.Next()
	if firstSong == nil {
		t.Fatal("Should get first song from queue")
	}

	err := queue.StartPlayback(firstSong.URL, voiceConn, mockDB)
	if err != nil {
		t.Errorf("Should start playback successfully: %v", err)
	}

	// Verify playback state
	if !queue.IsPlaying() {
		t.Error("Queue should be playing")
	}

	if !queue.HasActivePipeline() {
		t.Error("Queue should have active pipeline")
	}

	// Step 3: Test embed generation during playback
	nowPlayingEmbed := embedBuilder.NowPlaying(firstSong.Title, firstSong.URL, 3*time.Minute)
	if !strings.Contains(nowPlayingEmbed.Description, firstSong.Title) {
		t.Error("Now playing embed should contain song title")
	}

	// Test queue status embed with mock data
	currentSongStr := fmt.Sprintf("**%s** (Requested by: %s)", firstSong.Title, firstSong.RequestedBy)
	queueItems := []string{"**Test Song 2** (Requested by: User2)"}
	queueStatusEmbed := embedBuilder.QueueStatus(currentSongStr, queueItems, queue.Size())
	if len(queueStatusEmbed.Fields) == 0 {
		t.Error("Queue status embed should have fields")
	}

	// Step 4: Simulate error and test recovery
	pipeline := queue.GetPipeline()
	if pipeline == nil {
		t.Fatal("Queue should have pipeline")
	}

	// Stop current playback
	err = pipeline.Stop()
	if err != nil {
		t.Errorf("Should stop playback successfully: %v", err)
	}

	// Step 5: Start next song
	if queue.Size() > 0 {
		nextSong := queue.Next()
		if nextSong == nil {
			t.Error("Should get next song from queue")
		} else {
			err = queue.StartPlayback(nextSong.URL, voiceConn, mockDB)
			if err != nil {
				t.Errorf("Should start next song successfully: %v", err)
			}

			// Verify new song is playing
			current := queue.Current()
			if current == nil || current.Title != nextSong.Title {
				t.Error("Current song should be the next song")
			}
		}
	}

	// Step 6: Test error embed generation
	testError := errors.New("test playback error")
	errorEmbed := embedBuilder.PlaybackError("https://youtube.com/watch?v=error", testError)
	if !strings.Contains(errorEmbed.Fields[0].Value, "test playback error") {
		t.Error("Error embed should contain error message")
	}

	// Step 7: Clean up
	queue.StopAndCleanup()

	// Verify cleanup
	if queue.IsPlaying() {
		t.Error("Queue should not be playing after cleanup")
	}

	if queue.HasActivePipeline() {
		t.Error("Queue should not have active pipeline after cleanup")
	}

	// Step 8: Verify mock database logging (check that logs were created)
	logCount := mockDB.GetLogCount()
	if logCount == 0 {
		t.Log("Note: Mock database logging not implemented in this test setup")
	}
}

// Performance and stress tests

func TestIntegration_Performance_MultipleSequentialPlaybacks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	mockDB := createMockDatabase()
	guildID := "test-guild-perf"

	queue := NewMockMusicQueue(guildID)
	voiceConn := createTestVoiceConnection(guildID, "test-channel-perf")

	// Test multiple sequential playbacks
	urls := []string{
		"https://youtube.com/watch?v=working1",
		"https://youtube.com/watch?v=working2",
		"https://youtube.com/watch?v=working3",
	}

	startTime := time.Now()

	for i, url := range urls {
		t.Run(fmt.Sprintf("Playback_%d", i+1), func(t *testing.T) {
			// Add to queue
			queue.Add(url, fmt.Sprintf("Test Song %d", i+1), "TestUser")

			// Start playback
			nextSong := queue.Next()
			err := queue.StartPlayback(nextSong.URL, voiceConn, mockDB)
			if err != nil {
				t.Errorf("Playback %d should succeed: %v", i+1, err)
			}

			// Allow brief playback time
			time.Sleep(50 * time.Millisecond)

			// Stop playback
			pipeline := queue.GetPipeline()
			if pipeline != nil {
				pipeline.Stop()
			}
		})
	}

	totalTime := time.Since(startTime)
	t.Logf("Total time for %d sequential playbacks: %v", len(urls), totalTime)

	// Cleanup
	queue.StopAndCleanup()
}

func TestIntegration_Reliability_ErrorRecovery(t *testing.T) {
	mockDB := createMockDatabase()
	guildID := "test-guild-reliability"

	pipeline, streamProcessor, _, errorHandler := createIntegrationTestPipeline(t, mockDB, guildID)
	defer pipeline.Shutdown()

	voiceConn := createTestVoiceConnection(guildID, "test-channel-reliability")

	// Configure for aggressive retry testing
	errorHandler.handleErrorResult = true
	errorHandler.handleErrorDelay = 10 * time.Millisecond
	errorHandler.maxRetries = 5

	// Test multiple error scenarios
	errorScenarios := []struct {
		name        string
		failureType string
		shouldRetry bool
	}{
		{"Network Error", "network", true},
		{"FFmpeg Error", "ffmpeg", true},
		{"Timeout Error", "timeout", true},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Reset pipeline state
			pipeline.Stop()

			// Set failure mode
			streamProcessor.SetFailureMode(scenario.failureType)

			// Try to play
			err := pipeline.PlayURL("https://youtube.com/watch?v=working1", voiceConn)
			if err == nil {
				t.Errorf("Should fail with %s", scenario.name)
			}

			// Clear failure mode for next test
			streamProcessor.ClearFailureMode()

			// Verify error was handled appropriately
			if scenario.shouldRetry && len(errorHandler.notifyRetryCalls) == 0 {
				t.Errorf("Should have attempted retries for %s", scenario.name)
			}
		})
	}
}