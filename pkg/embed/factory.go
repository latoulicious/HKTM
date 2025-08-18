package embed

// DefaultEmbedFactory implements EmbedFactory interface
type DefaultEmbedFactory struct{}

// NewEmbedFactory creates a new DefaultEmbedFactory instance
func NewEmbedFactory() EmbedFactory {
	return &DefaultEmbedFactory{}
}

// CreateAudioEmbedBuilder creates an AudioEmbedBuilder instance
func (f *DefaultEmbedFactory) CreateAudioEmbedBuilder() AudioEmbedBuilder {
	return NewAudioEmbedBuilder()
}

// CreateBasicEmbedBuilder creates a basic EmbedBuilder instance
func (f *DefaultEmbedFactory) CreateBasicEmbedBuilder() EmbedBuilder {
	return NewAudioEmbedBuilder() // AudioEmbeds implements EmbedBuilder interface
}

// Global factory instance for convenience
var globalFactory EmbedFactory = NewEmbedFactory()

// CreateAudioEmbeds creates an AudioEmbedBuilder using the global factory
func CreateAudioEmbeds() AudioEmbedBuilder {
	return globalFactory.CreateAudioEmbedBuilder()
}

// CreateBasicEmbeds creates a basic EmbedBuilder using the global factory
func CreateBasicEmbeds() EmbedBuilder {
	return globalFactory.CreateBasicEmbedBuilder()
}

// CreateErrorEmbeds creates an error-specific EmbedBuilder
func CreateErrorEmbeds() EmbedBuilder {
	return NewErrorEmbedBuilder()
}

// CreateSuccessEmbeds creates a success-specific EmbedBuilder
func CreateSuccessEmbeds() EmbedBuilder {
	return NewSuccessEmbedBuilder()
}
