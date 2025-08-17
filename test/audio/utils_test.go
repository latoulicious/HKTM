package audio_test

import (
	"strings"
	"testing"
	"time"

	"github.com/latoulicious/HKTM/pkg/audio"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "whitespace only URL",
			url:     "   ",
			wantErr: true,
		},
		{
			name:    "valid HTTP URL",
			url:     "http://example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL",
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid YouTube URL",
			url:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			wantErr: false,
		},
		{
			name:    "YouTube URL without protocol",
			url:     "youtube.com/watch?v=dQw4w9WgXcQ",
			wantErr: false,
		},
		{
			name:    "youtu.be URL",
			url:     "youtu.be/dQw4w9WgXcQ",
			wantErr: false,
		},
		{
			name:    "invalid URL format",
			url:     "not-a-url",
			wantErr: true,
		},
		{
			name:    "malformed HTTP URL",
			url:     "http://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := audio.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "seconds only",
			duration: 30 * time.Second,
			expected: "30.0s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m 30s",
		},
		{
			name:     "hours, minutes and seconds",
			duration: 1*time.Hour + 30*time.Minute + 45*time.Second,
			expected: "1h 30m 45s",
		},
		{
			name:     "exactly one minute",
			duration: 1 * time.Minute,
			expected: "1m 0s",
		},
		{
			name:     "exactly one hour",
			duration: 1 * time.Hour,
			expected: "1h 0m 0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := audio.FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCreateContextFields(t *testing.T) {
	tests := []struct {
		name    string
		guildID string
		userID  string
		url     string
	}{
		{
			name:    "all fields provided",
			guildID: "guild123",
			userID:  "user456",
			url:     "https://youtube.com/watch?v=test",
		},
		{
			name:    "only guild ID",
			guildID: "guild123",
			userID:  "",
			url:     "",
		},
		{
			name:    "empty fields",
			guildID: "",
			userID:  "",
			url:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := audio.CreateContextFields(tt.guildID, tt.userID, tt.url)

			// Check that timestamp is always present
			if _, exists := fields["timestamp"]; !exists {
				t.Error("CreateContextFields() should always include timestamp")
			}

			// Check guild_id
			if tt.guildID != "" {
				if fields["guild_id"] != tt.guildID {
					t.Errorf("CreateContextFields() guild_id = %v, expected %v", fields["guild_id"], tt.guildID)
				}
			} else {
				if _, exists := fields["guild_id"]; exists {
					t.Error("CreateContextFields() should not include empty guild_id")
				}
			}

			// Check user_id
			if tt.userID != "" {
				if fields["user_id"] != tt.userID {
					t.Errorf("CreateContextFields() user_id = %v, expected %v", fields["user_id"], tt.userID)
				}
			} else {
				if _, exists := fields["user_id"]; exists {
					t.Error("CreateContextFields() should not include empty user_id")
				}
			}

			// Check url
			if tt.url != "" {
				if fields["url"] != tt.url {
					t.Errorf("CreateContextFields() url = %v, expected %v", fields["url"], tt.url)
				}
			} else {
				if _, exists := fields["url"]; exists {
					t.Error("CreateContextFields() should not include empty url")
				}
			}
		})
	}
}

func TestCreateContextFieldsWithComponent(t *testing.T) {
	guildID := "guild123"
	userID := "user456"
	url := "https://youtube.com/test"
	component := "audio"

	fields := audio.CreateContextFieldsWithComponent(guildID, userID, url, component)

	// Should include all base fields
	if fields["guild_id"] != guildID {
		t.Errorf("CreateContextFieldsWithComponent() guild_id = %v, expected %v", fields["guild_id"], guildID)
	}

	if fields["user_id"] != userID {
		t.Errorf("CreateContextFieldsWithComponent() user_id = %v, expected %v", fields["user_id"], userID)
	}

	if fields["url"] != url {
		t.Errorf("CreateContextFieldsWithComponent() url = %v, expected %v", fields["url"], url)
	}

	// Should include component
	if fields["component"] != component {
		t.Errorf("CreateContextFieldsWithComponent() component = %v, expected %v", fields["component"], component)
	}

	// Should include timestamp
	if _, exists := fields["timestamp"]; !exists {
		t.Error("CreateContextFieldsWithComponent() should include timestamp")
	}
}

func TestIsYouTubeURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "youtube.com URL",
			url:      "https://www.youtube.com/watch?v=test",
			expected: true,
		},
		{
			name:     "youtu.be URL",
			url:      "https://youtu.be/test",
			expected: true,
		},
		{
			name:     "youtube.com without protocol",
			url:      "youtube.com/watch?v=test",
			expected: true,
		},
		{
			name:     "youtu.be without protocol",
			url:      "youtu.be/test",
			expected: true,
		},
		{
			name:     "non-YouTube URL",
			url:      "https://example.com",
			expected: false,
		},
		{
			name:     "empty URL",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := audio.IsYouTubeURL(tt.url)
			if result != tt.expected {
				t.Errorf("IsYouTubeURL() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "empty URL",
			url:      "",
			expected: "",
		},
		{
			name:     "YouTube URL with query params",
			url:      "https://www.youtube.com/watch?v=test&list=playlist&index=1",
			expected: "https://www.youtube.com/watch?v=test",
		},
		{
			name:     "regular URL with query params",
			url:      "https://example.com/path?param=value&secret=hidden",
			expected: "https://example.com/path",
		},
		{
			name:     "URL with fragment",
			url:      "https://example.com/path#fragment",
			expected: "https://example.com/path",
		},
		{
			name:     "very long URL",
			url:      "https://example.com/" + strings.Repeat("a", 200),
			expected: "https://example.com/" + strings.Repeat("a", 81) + "...",
		},
		{
			name:     "malformed URL",
			url:      "not-a-url-but-very-long-" + strings.Repeat("x", 200),
			expected: "not-a-url-but-very-long-" + strings.Repeat("x", 76) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := audio.SanitizeURL(tt.url)
			if result != tt.expected {
				t.Errorf("SanitizeURL() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
