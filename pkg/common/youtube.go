package common

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// IsYouTubeURL checks if a URL appears to be from YouTube
func IsYouTubeURL(urlStr string) bool {
	return strings.Contains(urlStr, "youtube.com") || strings.Contains(urlStr, "youtu.be")
}

// ExtractYouTubeVideoID extracts the video ID from a YouTube URL
func ExtractYouTubeVideoID(youtubeURL string) string {
	// Handle youtube.com URLs
	if strings.Contains(youtubeURL, "youtube.com") {
		parsedURL, err := url.Parse(youtubeURL)
		if err != nil {
			return ""
		}

		// Check for v parameter
		if videoID := parsedURL.Query().Get("v"); videoID != "" {
			return videoID
		}

		// Check for embed URLs like /embed/VIDEO_ID
		if strings.Contains(parsedURL.Path, "/embed/") {
			parts := strings.Split(parsedURL.Path, "/embed/")
			if len(parts) > 1 {
				return strings.Split(parts[1], "?")[0] // Remove any query params
			}
		}
	}

	// Handle youtu.be URLs
	if strings.Contains(youtubeURL, "youtu.be") {
		parsedURL, err := url.Parse(youtubeURL)
		if err != nil {
			return ""
		}

		// Extract video ID from path
		videoID := strings.TrimPrefix(parsedURL.Path, "/")
		return strings.Split(videoID, "?")[0] // Remove any query params
	}

	// Fallback: use regex to find 11-character alphanumeric video ID
	re := regexp.MustCompile(`[a-zA-Z0-9_-]{11}`)
	matches := re.FindAllString(youtubeURL, -1)
	if len(matches) > 0 {
		return matches[0]
	}

	return ""
}

// GetYouTubeThumbnailURL generates a thumbnail URL from a video ID
func GetYouTubeThumbnailURL(videoID string) string {
	if videoID == "" {
		return ""
	}
	// Use maxresdefault for best quality, fallback to hqdefault if needed
	return fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", videoID)
}

// GetYouTubeMetadata extracts both title and duration from a YouTube URL with timeout and retry
func GetYouTubeMetadata(urlStr string) (title string, duration time.Duration, err error) {
	log.Printf("Extracting metadata from: %s", urlStr)

	// Add timeout and retry logic
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// Use yt-dlp to get both title and duration
		cmd := exec.CommandContext(ctx, "yt-dlp",
			"--no-playlist",
			"--no-warnings",
			"--print", "title",
			"--print", "duration",
			urlStr)

		var out bytes.Buffer
		cmd.Stdout = &out

		if err := cmd.Run(); err != nil {
			cancel()
			if attempt < maxRetries-1 {
				log.Printf("Metadata extraction attempt %d failed, retrying in 2 seconds...", attempt+1)
				time.Sleep(2 * time.Second)
				continue
			}
			log.Printf("Failed to get metadata after %d attempts: %v", maxRetries, err)
			return "Unknown Title", 0, fmt.Errorf("failed to extract metadata: %v", err)
		}

		cancel()

		output := strings.TrimSpace(out.String())
		lines := strings.Split(output, "\n")

		if len(lines) >= 1 {
			title = strings.TrimSpace(lines[0])
		}
		if len(lines) >= 2 {
			durationStr := strings.TrimSpace(lines[1])
			if durationStr != "" && durationStr != "None" {
				// yt-dlp returns duration in seconds
				if seconds, parseErr := strconv.ParseFloat(durationStr, 64); parseErr == nil {
					duration = time.Duration(seconds * float64(time.Second))
				}
			}
		}

		if title == "" {
			title = "Unknown Title"
		}

		log.Printf("Extracted metadata - Title: %s, Duration: %v", title, duration)
		return title, duration, nil
	}

	return "Unknown Title", 0, fmt.Errorf("failed to extract metadata after %d attempts", maxRetries)
}

// GetYouTubeAudioStreamWithMetadata extracts stream URL, title, and duration with improved reliability
func GetYouTubeAudioStreamWithMetadata(urlStr string) (streamURL, title string, duration time.Duration, err error) {
	log.Printf("Extracting audio stream and metadata from: %s", urlStr)

	// First, get metadata (title and duration)
	title, duration, metaErr := GetYouTubeMetadata(urlStr)
	if metaErr != nil {
		log.Printf("Warning: Failed to get metadata: %v", metaErr)
		title = "Unknown Title"
		duration = 0
	}

	// Extract a fresh stream URL
	streamURL, err = extractFreshStreamURL(urlStr)
	if err != nil {
		return "", title, duration, err
	}

	return streamURL, title, duration, nil
}

// extractFreshStreamURL extracts a fresh stream URL with multiple strategies
func extractFreshStreamURL(urlStr string) (streamURL string, err error) {
	strategies := [][]string{
		// Strategy 1: Best audio with format preference (updated for current YouTube)
		{"-f", "bestaudio[ext=m4a]/bestaudio[ext=webm]/bestaudio[ext=mp4]/bestaudio"},

		// Strategy 2: Android client (often bypasses restrictions)
		{"-f", "bestaudio", "--extractor-args", "youtube:player_client=android"},

		// Strategy 3: Web client with cookies
		{"-f", "bestaudio", "--extractor-args", "youtube:player_client=web"},

		// Strategy 4: Last resort - any audio
		{"-f", "worst[ext=m4a]/worst"},
	}

	maxRetries := 2
	for retry := 0; retry < maxRetries; retry++ {
		for i, strategy := range strategies {
			log.Printf("Trying extraction strategy %d/%d (retry %d/%d)", i+1, len(strategies), retry+1, maxRetries)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

			args := append([]string{"--no-playlist", "--no-warnings", "-g"}, strategy...)
			args = append(args, urlStr)

			cmd := exec.CommandContext(ctx, "yt-dlp", args...)
			var out bytes.Buffer
			cmd.Stdout = &out

			if err := cmd.Run(); err != nil {
				cancel()
				log.Printf("Strategy %d failed: %v", i+1, err)
				continue
			}

			cancel()

			streamURL = strings.TrimSpace(out.String())
			if streamURL != "" {
				// Take first URL if multiple are returned
				urls := strings.Split(streamURL, "\n")
				if len(urls) > 0 && urls[0] != "" {
					streamURL = urls[0]
					log.Printf("Successfully extracted stream URL using strategy %d", i+1)
					return streamURL, nil
				}
			}
		}

		if retry < maxRetries-1 {
			log.Printf("All strategies failed on attempt %d, retrying in 3 seconds...", retry+1)
			time.Sleep(3 * time.Second)
		}
	}

	return "", fmt.Errorf("failed to extract audio stream URL after trying all strategies with %d retries", maxRetries)
}

// GetFreshYouTubeStreamURL extracts a fresh stream URL for immediate use
// This function should be called just before starting playback to minimize URL expiration
func GetFreshYouTubeStreamURL(urlStr string) (streamURL string, err error) {
	log.Printf("Extracting fresh stream URL for immediate use: %s", urlStr)
	return extractFreshStreamURL(urlStr)
}

// SearchYouTubeAndGetURL searches for a query on YouTube and returns the first result's URL with timeout
func SearchYouTubeAndGetURL(query string) (url string, title string, duration time.Duration, err error) {
	log.Printf("Searching YouTube for: %s", query)

	// Add timeout and retry logic
	maxRetries := 2
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)

		// Use yt-dlp to search for videos
		cmd := exec.CommandContext(ctx, "yt-dlp",
			"--no-playlist",
			"--no-warnings",
			"--print", "webpage_url",
			"--print", "title",
			"--print", "duration",
			"--max-downloads", "1", // Only get the first result
			"ytsearch1:"+query) // Search for 1 result

		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		runErr := cmd.Run()
		cancel()

		output := strings.TrimSpace(out.String())

		// Parse the output regardless of exit status
		lines := strings.Split(output, "\n")

		if len(lines) >= 1 {
			url = strings.TrimSpace(lines[0])
		}
		if len(lines) >= 2 {
			title = strings.TrimSpace(lines[1])
		}
		if len(lines) >= 3 {
			durationStr := strings.TrimSpace(lines[2])
			if durationStr != "" && durationStr != "None" {
				// yt-dlp returns duration in seconds
				if seconds, parseErr := strconv.ParseFloat(durationStr, 64); parseErr == nil {
					duration = time.Duration(seconds * float64(time.Second))
				}
			}
		}

		// If we got a URL, return it even if there was an error
		if url != "" {
			if title == "" {
				title = "Unknown Title"
			}
			log.Printf("Search result - URL: %s, Title: %s, Duration: %v", url, title, duration)
			return url, title, duration, nil
		}

		// If we didn't get a URL and this isn't the last attempt, retry
		if attempt < maxRetries-1 {
			log.Printf("Search attempt %d failed, retrying in 2 seconds...", attempt+1)
			time.Sleep(2 * time.Second)
			continue
		}

		// Last attempt failed
		if runErr != nil {
			log.Printf("Failed to search YouTube: %v, stderr: %s", runErr, stderr.String())
			return "", "", 0, fmt.Errorf("failed to search YouTube: %v", runErr)
		}
		return "", "", 0, fmt.Errorf("no search results found")
	}

	return "", "", 0, fmt.Errorf("failed to search YouTube after %d attempts", maxRetries)
}

// IsURL checks if a string appears to be a URL
func IsURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://") ||
		strings.HasPrefix(str, "www.") || IsYouTubeURL(str)
}
