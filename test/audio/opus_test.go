package audio

import (
	"testing"

	"github.com/latoulicious/HKTM/pkg/audio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpusProcessor_Initialize(t *testing.T) {
	config := &audio.OpusConfig{
		Bitrate:   128000,
		FrameSize: 960,
	}

	processor := audio.NewOpusProcessor(config)

	// Test initialization
	err := processor.Initialize()
	require.NoError(t, err)

	// Test double initialization should fail
	err = processor.Initialize()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already initialized")

	// Clean up
	err = processor.Close()
	assert.NoError(t, err)
}

func TestOpusProcessor_Encode(t *testing.T) {
	config := &audio.OpusConfig{
		Bitrate:   128000,
		FrameSize: 960,
	}

	processor := audio.NewOpusProcessor(config)

	// Test encoding without initialization should fail
	pcmData := make([]int16, 1920) // 960 samples * 2 channels
	_, err := processor.Encode(pcmData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")

	// Initialize the processor
	err = processor.Initialize()
	require.NoError(t, err)

	// Test encoding with correct frame size
	opusData, err := processor.Encode(pcmData)
	assert.NoError(t, err)
	assert.NotEmpty(t, opusData)

	// Test encoding with wrong frame size should fail
	wrongSizePCM := make([]int16, 100)
	_, err = processor.Encode(wrongSizePCM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PCM frame size")

	// Test encoding with empty data should fail
	_, err = processor.Encode([]int16{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty PCM data")

	// Clean up
	err = processor.Close()
	assert.NoError(t, err)
}

func TestOpusProcessor_Close(t *testing.T) {
	config := &audio.OpusConfig{
		Bitrate:   128000,
		FrameSize: 960,
	}

	processor := audio.NewOpusProcessor(config)

	// Test closing without initialization should not error
	err := processor.Close()
	assert.NoError(t, err)

	// Initialize and then close
	err = processor.Initialize()
	require.NoError(t, err)

	err = processor.Close()
	assert.NoError(t, err)

	// Test encoding after close should fail
	pcmData := make([]int16, 1920)
	_, err = processor.Encode(pcmData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not initialized")

	// Test double close should not error
	err = processor.Close()
	assert.NoError(t, err)
}

func TestOpusProcessor_Configuration(t *testing.T) {
	config := &audio.OpusConfig{
		Bitrate:   64000,
		FrameSize: 480,
	}

	processor := audio.NewOpusProcessor(config)

	// Test that configuration is preserved
	retrievedConfig := processor.(*audio.OpusProcessor).GetConfig()
	assert.Equal(t, config.Bitrate, retrievedConfig.Bitrate)
	assert.Equal(t, config.FrameSize, retrievedConfig.FrameSize)
}

func TestOpusProcessor_IsInitialized(t *testing.T) {
	config := &audio.OpusConfig{
		Bitrate:   128000,
		FrameSize: 960,
	}

	processor := audio.NewOpusProcessor(config)

	// Should not be initialized initially
	assert.False(t, processor.(*audio.OpusProcessor).IsInitialized())

	// Should be initialized after Initialize()
	err := processor.Initialize()
	require.NoError(t, err)
	assert.True(t, processor.(*audio.OpusProcessor).IsInitialized())

	// Should not be initialized after Close()
	err = processor.Close()
	require.NoError(t, err)
	assert.False(t, processor.(*audio.OpusProcessor).IsInitialized())
}
