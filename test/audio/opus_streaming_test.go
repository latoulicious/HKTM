package audio

import (
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/audio"
)

// TestOpusProcessor_StreamingScenario tests the PCM to Opus conversion in a realistic streaming scenario
func TestOpusProcessor_StreamingScenario(t *testing.T) {
	// Create Discord-compatible configuration
	config := &audio.OpusConfig{
		Bitrate:   128000, // 128 kbps
		FrameSize: 960,    // 20ms frames at 48kHz
	}

	processor := audio.NewOpusProcessor(config)

	// Initialize the processor
	err := processor.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize Opus processor: %v", err)
	}
	defer processor.Close()

	// Prepare for streaming
	err = processor.PrepareForStreaming()
	if err != nil {
		t.Fatalf("Failed to prepare for streaming: %v", err)
	}

	// Simulate streaming scenario: encode multiple frames in sequence
	// This simulates what would happen during real Discord audio streaming
	const numFrames = 50 // Simulate 1 second of audio (50 * 20ms = 1000ms)
	frameSize := processor.GetFrameSize()
	frameDuration := processor.GetFrameDuration()

	t.Logf("Streaming simulation: %d frames of %d samples each (%v per frame)",
		numFrames, frameSize, frameDuration)

	var totalOpusBytes int
	var totalPCMSamples int

	startTime := time.Now()

	for frameNum := 0; frameNum < numFrames; frameNum++ {
		// Generate test PCM data for this frame
		pcmFrame := generateTestPCMFrame(frameSize, frameNum)

		// Validate frame size before encoding
		err := processor.ValidateFrameSize(pcmFrame)
		if err != nil {
			t.Fatalf("Frame %d validation failed: %v", frameNum, err)
		}

		// Encode the frame
		frameStart := time.Now()
		opusData, err := processor.EncodeFrame(pcmFrame)
		encodingDuration := time.Since(frameStart)

		if err != nil {
			t.Fatalf("Failed to encode frame %d: %v", frameNum, err)
		}

		// Validate the encoded data
		if len(opusData) == 0 {
			t.Errorf("Frame %d produced empty Opus data", frameNum)
		}

		if len(opusData) > 4000 {
			t.Errorf("Frame %d produced oversized Opus data: %d bytes", frameNum, len(opusData))
		}

		totalOpusBytes += len(opusData)
		totalPCMSamples += len(pcmFrame)

		// Log every 10th frame to avoid spam
		if frameNum%10 == 0 {
			t.Logf("Frame %d: %d PCM samples -> %d Opus bytes (encoded in %v)",
				frameNum, len(pcmFrame), len(opusData), encodingDuration)
		}

		// Simulate real-time constraints: encoding should be much faster than frame duration
		if encodingDuration > frameDuration/2 {
			t.Errorf("Frame %d encoding too slow: %v (should be < %v)",
				frameNum, encodingDuration, frameDuration/2)
		}
	}

	totalDuration := time.Since(startTime)
	expectedDuration := time.Duration(numFrames) * frameDuration

	t.Logf("Streaming simulation completed:")
	t.Logf("  Total frames: %d", numFrames)
	t.Logf("  Total PCM samples: %d", totalPCMSamples)
	t.Logf("  Total Opus bytes: %d", totalOpusBytes)
	t.Logf("  Compression ratio: %.2f:1", float64(totalPCMSamples*2)/float64(totalOpusBytes))
	t.Logf("  Processing time: %v", totalDuration)
	t.Logf("  Expected audio duration: %v", expectedDuration)
	t.Logf("  Real-time factor: %.2fx", float64(expectedDuration)/float64(totalDuration))

	// Verify we can process faster than real-time (important for streaming)
	if totalDuration > expectedDuration {
		t.Errorf("Processing too slow: %v > %v (not real-time capable)", totalDuration, expectedDuration)
	}

	// Verify reasonable compression ratio (PCM is 16-bit, so 2 bytes per sample)
	compressionRatio := float64(totalPCMSamples*2) / float64(totalOpusBytes)
	if compressionRatio < 2.0 || compressionRatio > 20.0 {
		t.Errorf("Unexpected compression ratio: %.2f:1 (expected 2-20:1)", compressionRatio)
	}
}

// generateTestPCMFrame generates test PCM data for a given frame
func generateTestPCMFrame(frameSize int, frameNum int) []int16 {
	pcmFrame := make([]int16, frameSize)

	// Generate a simple test pattern that varies per frame
	// This creates a stereo sine wave pattern
	for i := 0; i < frameSize; i += 2 {
		// Calculate sample position within the frame
		samplePos := i / 2

		// Calculate sine values (simplified, not actual audio synthesis)
		// Use frame and sample position to create varying test data
		leftValue := 16000.0 * (float64(samplePos+frameNum*100) / 1000.0)
		rightValue := 12000.0 * (float64(samplePos+frameNum*150) / 800.0)

		// Clamp values to int16 range before casting
		if leftValue > 32767 {
			leftValue = 32767
		}
		if leftValue < -32768 {
			leftValue = -32768
		}
		if rightValue > 32767 {
			rightValue = 32767
		}
		if rightValue < -32768 {
			rightValue = -32768
		}

		leftSample := int16(leftValue)
		rightSample := int16(rightValue)

		pcmFrame[i] = leftSample    // Left channel
		pcmFrame[i+1] = rightSample // Right channel
	}

	return pcmFrame
}

// TestOpusProcessor_TimingAccuracy tests that frame timing is accurate for Discord
func TestOpusProcessor_TimingAccuracy(t *testing.T) {
	config := &audio.OpusConfig{
		Bitrate:   128000,
		FrameSize: 960,
	}

	processor := audio.NewOpusProcessor(config)
	err := processor.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	defer processor.Close()

	// Test timing accuracy
	frameDuration := processor.GetFrameDuration()
	expectedDuration := 20 * time.Millisecond

	if frameDuration != expectedDuration {
		t.Errorf("Incorrect frame duration: got %v, expected %v", frameDuration, expectedDuration)
	}

	// Test frame size accuracy
	frameSize := processor.GetFrameSize()
	expectedFrameSize := 1920 // 960 samples per channel * 2 channels

	if frameSize != expectedFrameSize {
		t.Errorf("Incorrect frame size: got %d, expected %d", frameSize, expectedFrameSize)
	}

	// Verify that 50 frames = 1 second (50 * 20ms = 1000ms)
	totalDuration := time.Duration(50) * frameDuration
	expectedTotal := 1 * time.Second

	if totalDuration != expectedTotal {
		t.Errorf("Timing calculation error: 50 frames = %v, expected %v", totalDuration, expectedTotal)
	}

	t.Logf("Timing verification passed:")
	t.Logf("  Frame duration: %v", frameDuration)
	t.Logf("  Frame size: %d samples", frameSize)
	t.Logf("  50 frames duration: %v", totalDuration)
}
