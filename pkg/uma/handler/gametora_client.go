package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/latoulicious/HKTM/internal/config"
	"github.com/latoulicious/HKTM/pkg/cron"
	"github.com/latoulicious/HKTM/pkg/uma/shared"
)

// Global Gametora client instance
var globalGametoraClient *GametoraClient

// GametoraClient represents the Gametora API client for stable JSON endpoints
type GametoraClient struct {
	baseURL    string
	httpClient *http.Client
	// cache          map[string]*CacheEntry
	cacheMutex     sync.RWMutex
	cacheTTL       time.Duration
	buildID        string
	buildMutex     sync.RWMutex
	buildIDManager *cron.BuildIDManager
}

// GetGametoraClient returns the global Gametora client instance
func GetGametoraClient() *GametoraClient {
	return globalGametoraClient
}

// GetBuildIDManager returns the build ID manager
func (c *GametoraClient) GetBuildIDManager() *cron.BuildIDManager {
	return c.buildIDManager
}

// RefreshBuildID manually triggers a build ID refresh
func (c *GametoraClient) RefreshBuildID() error {
	return c.refreshBuildID()
}

// StopBuildIDManager stops the build ID manager cron job
func (c *GametoraClient) StopBuildIDManager() {
	if c.buildIDManager != nil {
		c.buildIDManager.Stop()
	}
}

// refreshBuildID refreshes the build ID by fetching it from Gametora
func (c *GametoraClient) refreshBuildID() error {
	// Clear the current build ID to force a fresh fetch
	c.buildMutex.Lock()
	c.buildID = ""
	c.buildMutex.Unlock()

	// Fetch new build ID
	_, err := c.GetBuildID()
	return err
}

// NewGametoraClient creates a new Gametora API client
func NewGametoraClient(cfg *config.Config) *GametoraClient {
	client := &GametoraClient{
		baseURL: "https://gametora.com/_next/data",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		// cache:    make(map[string]*CacheEntry),
		cacheTTL: 30 * time.Minute, // Cache for 30 minutes
	}

	// Initialize build ID manager with config
	client.buildIDManager = cron.NewBuildIDManager(client.refreshBuildID, cfg)

	// Set global instance
	globalGametoraClient = client

	return client
}

// GetBuildID fetches the current build ID from Gametora
func (c *GametoraClient) GetBuildID() (string, error) {
	c.buildMutex.RLock()
	if c.buildID != "" {
		defer c.buildMutex.RUnlock()
		return c.buildID, nil
	}
	c.buildMutex.RUnlock()

	// Fetch the main page to get the build ID
	resp, err := c.httpClient.Get("https://gametora.com/umamusume/supports")
	if err != nil {
		return "", fmt.Errorf("failed to fetch build ID: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body := make([]byte, 1024*1024) // 1MB buffer
	n, err := resp.Body.Read(body)
	if err != nil && n == 0 {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Look for build ID patterns
	bodyStr := string(body[:n])

	// Try to find build ID using different approaches

	// Use a simple string search approach
	if strings.Contains(bodyStr, "_next/data/") {
		// Find the pattern _next/data/{build_id}/
		start := strings.Index(bodyStr, "_next/data/")
		if start != -1 {
			start += len("_next/data/")
			end := strings.Index(bodyStr[start:], "/")
			if end != -1 {
				buildID := bodyStr[start : start+end]
				if len(buildID) > 10 && len(buildID) < 50 {
					c.buildMutex.Lock()
					c.buildID = buildID
					c.buildMutex.Unlock()
					return buildID, nil
				}
			}
		}
	}

	// Try to find buildId in JSON
	if strings.Contains(bodyStr, "buildId") {
		start := strings.Index(bodyStr, "buildId")
		if start != -1 {
			// Look for the value after buildId
			valueStart := strings.Index(bodyStr[start:], "\"")
			if valueStart != -1 {
				valueStart += start + valueStart + 1
				valueEnd := strings.Index(bodyStr[valueStart:], "\"")
				if valueEnd != -1 {
					buildID := bodyStr[valueStart : valueStart+valueEnd]
					if len(buildID) > 10 && len(buildID) < 50 {
						c.buildMutex.Lock()
						c.buildID = buildID
						c.buildMutex.Unlock()
						return buildID, nil
					}
				}
			}
		}
	}

	// If no build ID found, try a hardcoded one as fallback
	// This is the build ID from your example
	fallbackBuildID := "4Lod4e9rq2HCjy-VKjMHJ"
	c.buildMutex.Lock()
	c.buildID = fallbackBuildID
	c.buildMutex.Unlock()

	return fallbackBuildID, nil
}

// SearchSimplifiedSupportCard searches for a support card using the Gametora JSON API and returns simplified structure
func (c *GametoraClient) SearchSimplifiedSupportCard(query string) *shared.SimplifiedGametoraSearchResult {
	// Check cache first
	// cacheKey := fmt.Sprintf("gametora_simplified_support_%s", strings.ToLower(query))
	// if cached := c.getFromCache(cacheKey); cached != nil {
	// 	if result, ok := cached.(*SimplifiedGametoraSearchResult); ok {
	// 		return result
	// 	}
	// }

	// Get build ID
	buildID, err := c.GetBuildID()
	if err != nil {
		result := &shared.SimplifiedGametoraSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to get build ID: %v", err),
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	// First, get the list of all support cards
	supportsURL := fmt.Sprintf("%s/%s/umamusume/supports.json", c.baseURL, buildID)
	resp, err := c.httpClient.Get(supportsURL)
	if err != nil {
		result := &shared.SimplifiedGametoraSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch supports list: %v", err),
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &shared.SimplifiedGametoraSearchResult{
			Found: false,
			Error: fmt.Errorf("supports API returned status code: %d", resp.StatusCode),
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	var supportsResp shared.GametoraSupportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&supportsResp); err != nil {
		result := &shared.SimplifiedGametoraSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to decode supports response: %v", err),
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	// Find all matches
	query = strings.ToLower(strings.TrimSpace(query))
	var allMatches []struct {
		URLName     string  `json:"url_name"`
		SupportID   int     `json:"support_id"`
		CharID      int     `json:"char_id"`
		CharName    string  `json:"char_name"`
		NameJp      string  `json:"name_jp"`
		NameKo      string  `json:"name_ko"`
		NameTw      string  `json:"name_tw"`
		Rarity      int     `json:"rarity"`
		Type        string  `json:"type"`
		Obtained    string  `json:"obtained"`
		Release     string  `json:"release"`
		ReleaseKo   string  `json:"release_ko,omitempty"`
		ReleaseZhTw string  `json:"release_zh_tw,omitempty"`
		ReleaseEn   string  `json:"release_en,omitempty"`
		Effects     [][]int `json:"effects"`
		Hints       struct {
			HintSkills []struct {
				ID     int      `json:"id"`
				Type   []string `json:"type"`
				NameEn string   `json:"name_en"`
				IconID int      `json:"iconid"`
			} `json:"hint_skills"`
			HintOthers []struct {
				HintType  int `json:"hint_type"`
				HintValue int `json:"hint_value"`
			} `json:"hint_others"`
		} `json:"hints"`
		EventSkills []struct {
			ID     int      `json:"id"`
			Type   []string `json:"type"`
			NameEn string   `json:"name_en"`
			Rarity int      `json:"rarity"`
			IconID int      `json:"iconid"`
		} `json:"event_skills"`
		Unique *struct {
			Level   int `json:"level"`
			Effects []struct {
				Type   int `json:"type"`
				Value  int `json:"value"`
				Value1 int `json:"value_1,omitempty"`
				Value2 int `json:"value_2,omitempty"`
				Value3 int `json:"value_3,omitempty"`
				Value4 int `json:"value_4,omitempty"`
			} `json:"effects"`
		} `json:"unique,omitempty"`
	}

	for _, support := range supportsResp.PageProps.SupportData {
		urlName := strings.ToLower(support.URLName)

		// Simple URL name matching - check if all query words are in the URL name
		queryWords := strings.Fields(query)
		allWordsMatch := true

		for _, word := range queryWords {
			if len(word) > 2 && !strings.Contains(urlName, word) {
				allWordsMatch = false
				break
			}
		}

		// Only add if all words match in the URL name
		if allWordsMatch && len(queryWords) > 0 {
			allMatches = append(allMatches, struct {
				URLName     string  `json:"url_name"`
				SupportID   int     `json:"support_id"`
				CharID      int     `json:"char_id"`
				CharName    string  `json:"char_name"`
				NameJp      string  `json:"name_jp"`
				NameKo      string  `json:"name_ko"`
				NameTw      string  `json:"name_tw"`
				Rarity      int     `json:"rarity"`
				Type        string  `json:"type"`
				Obtained    string  `json:"obtained"`
				Release     string  `json:"release"`
				ReleaseKo   string  `json:"release_ko,omitempty"`
				ReleaseZhTw string  `json:"release_zh_tw,omitempty"`
				ReleaseEn   string  `json:"release_en,omitempty"`
				Effects     [][]int `json:"effects"`
				Hints       struct {
					HintSkills []struct {
						ID     int      `json:"id"`
						Type   []string `json:"type"`
						NameEn string   `json:"name_en"`
						IconID int      `json:"iconid"`
					} `json:"hint_skills"`
					HintOthers []struct {
						HintType  int `json:"hint_type"`
						HintValue int `json:"hint_value"`
					} `json:"hint_others"`
				} `json:"hints"`
				EventSkills []struct {
					ID     int      `json:"id"`
					Type   []string `json:"type"`
					NameEn string   `json:"name_en"`
					Rarity int      `json:"rarity"`
					IconID int      `json:"iconid"`
				} `json:"event_skills"`
				Unique *struct {
					Level   int `json:"level"`
					Effects []struct {
						Type   int `json:"type"`
						Value  int `json:"value"`
						Value1 int `json:"value_1,omitempty"`
						Value2 int `json:"value_2,omitempty"`
						Value3 int `json:"value_3,omitempty"`
						Value4 int `json:"value_4,omitempty"`
					} `json:"effects"`
				} `json:"unique,omitempty"`
			}{
				URLName:     support.URLName,
				SupportID:   support.SupportID,
				CharID:      support.CharID,
				CharName:    support.CharName,
				NameJp:      support.NameJp,
				NameKo:      support.NameKo,
				NameTw:      support.NameTw,
				Rarity:      support.Rarity,
				Type:        support.Type,
				Obtained:    support.Obtained,
				Release:     support.Release,
				ReleaseKo:   support.ReleaseKo,
				ReleaseZhTw: support.ReleaseZhTw,
				ReleaseEn:   support.ReleaseEn,
				Effects:     support.Effects,
				Hints:       support.Hints,
				EventSkills: support.EventSkills,
				Unique:      support.Unique,
			})
		}
	}

	if len(allMatches) == 0 {
		result := &shared.SimplifiedGametoraSearchResult{
			Found: false,
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	// Sort matches by rarity (highest first) since we no longer use scores
	for i := 0; i < len(allMatches)-1; i++ {
		for j := i + 1; j < len(allMatches); j++ {
			if allMatches[i].Rarity < allMatches[j].Rarity {
				allMatches[i], allMatches[j] = allMatches[j], allMatches[i]
			}
		}
	}

	// Convert all matches to simplified structure
	var simplifiedCards []*shared.SimplifiedSupportCard
	for _, match := range allMatches {
		simplifiedCard := &shared.SimplifiedSupportCard{
			URLName:     match.URLName,
			SupportID:   match.SupportID,
			CharID:      match.CharID,
			CharName:    match.CharName,
			NameJp:      match.NameJp,
			NameKo:      match.NameKo,
			NameTw:      match.NameTw,
			Rarity:      match.Rarity,
			Type:        match.Type,
			Obtained:    match.Obtained,
			Release:     match.Release,
			ReleaseKo:   match.ReleaseKo,
			ReleaseZhTw: match.ReleaseZhTw,
			ReleaseEn:   match.ReleaseEn,
			Effects:     match.Effects,
			Hints:       match.Hints,
			EventSkills: match.EventSkills,
			Unique:      match.Unique,
		}
		simplifiedCards = append(simplifiedCards, simplifiedCard)
	}

	result := &shared.SimplifiedGametoraSearchResult{
		Found:        true,
		SupportCard:  simplifiedCards[0], // Best match as primary
		SupportCards: simplifiedCards,    // All matches
		Query:        query,
	}

	// c.setCache(cacheKey, result)
	return result
}

// getFromCache retrieves an item from cache
// func (c *GametoraClient) getFromCache(key string) interface{} {
// 	c.cacheMutex.RLock()
// 	defer c.cacheMutex.RUnlock()

// 	if entry, exists := c.cache[key]; exists && !entry.IsExpired() {
// 		return entry.Data
// 	}

// 	return nil
// }

// setCache stores an item in cache
// func (c *GametoraClient) setCache(key string, data interface{}) {
// 	c.cacheMutex.Lock()
// 	defer c.cacheMutex.Unlock()

// 	c.cache[key] = &CacheEntry{
// 		Data:      data,
// 		Timestamp: time.Now(),
// 		TTL:       c.cacheTTL,
// 	}
// }

// GetSupportCardImageURL generates the image URL for a support card based on its URL name
func (c *GametoraClient) GetSupportCardImageURL(urlName string) string {
	// Extract the ID from the URL name (e.g., "10001-special-week" -> "10001")
	parts := strings.Split(urlName, "-")
	if len(parts) > 0 {
		return fmt.Sprintf("https://gametora.com/images/umamusume/supports/tex_support_card_%s.png", parts[0])
	}
	return ""
}
