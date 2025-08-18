package embed

// Factory functions for creating embed builders

// NewAudioEmbeds creates a new AudioEmbeds instance
func NewAudioEmbeds() *AudioEmbeds {
	return &AudioEmbeds{
		baseColor: 0x0099ff,
		botName:   "Hokko Tarumae",
	}
}

// GetGlobalAudioEmbedBuilder returns a global instance of AudioEmbedBuilder
// This can be used across the application for consistent embed styling
var globalAudioEmbedBuilder AudioEmbedBuilder

func init() {
	globalAudioEmbedBuilder = NewAudioEmbedBuilder()
}

// GetGlobalAudioEmbedBuilder returns the global AudioEmbedBuilder instance
func GetGlobalAudioEmbedBuilder() AudioEmbedBuilder {
	return globalAudioEmbedBuilder
}