package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/latoulicious/HKTM/pkg/uma/shared"
)

// Client represents the Uma Musume API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	// cache      map[string]*shared.CacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

// NewClient creates a new Uma Musume API client
func NewClient() *Client {
	return &Client{
		baseURL: "https://umapyoi.net/api",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		// cache:    make(map[string]*shared.CacheEntry),
		cacheTTL: 5 * time.Minute, // Cache for 5 minutes
	}
}

// SearchCharacter searches for a character by name
func (c *Client) SearchCharacter(query string) *shared.CharacterSearchResult {
	// // Check cache first
	// cacheKey := fmt.Sprintf("char_search_%s", strings.ToLower(query))
	// if cached := c.getFromCache(cacheKey); cached != nil {
	// 	if result, ok := cached.(*shared.CharacterSearchResult); ok {
	// 		return result
	// 	}
	// }

	// Make API request
	url := fmt.Sprintf("%s/v1/character/list", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		result := &shared.CharacterSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch character data: %v", err),
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &shared.CharacterSearchResult{
			Found: false,
			Error: fmt.Errorf("API returned status code: %d", resp.StatusCode),
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	var apiResp shared.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		result := &shared.CharacterSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to decode API response: %v", err),
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	// Find the best match
	bestMatch := c.findBestMatch(query, apiResp)

	result := &shared.CharacterSearchResult{
		Found:     bestMatch != nil,
		Character: bestMatch,
		Query:     query,
	}

	// c.setCache(cacheKey, result)
	return result
}

// GetAllCharacters fetches all characters from the API
func (c *Client) GetAllCharacters() *shared.CharacterListResult {
	// // Check cache first
	// cacheKey := "all_characters"
	// if cached := c.getFromCache(cacheKey); cached != nil {
	// 	if result, ok := cached.(*shared.CharacterListResult); ok {
	// 		return result
	// 	}
	// }

	// Make API request
	url := fmt.Sprintf("%s/v1/character/list", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		result := &shared.CharacterListResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch character data: %v", err),
		}
		// c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &shared.CharacterListResult{
			Found: false,
			Error: fmt.Errorf("API returned status code: %d", resp.StatusCode),
		}
		// c.setCache(cacheKey, result)
		return result
	}

	var apiResp shared.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		result := &shared.CharacterListResult{
			Found: false,
			Error: fmt.Errorf("failed to decode API response: %v", err),
		}
		// c.setCache(cacheKey, result)
		return result
	}

	result := &shared.CharacterListResult{
		Found:      true,
		Characters: apiResp,
	}

	// c.setCache(cacheKey, result)
	return result
}

// findBestMatch finds the best character match for the given query
func (c *Client) findBestMatch(query string, characters []shared.Character) *shared.Character {
	query = strings.ToLower(query)

	// First, try exact match with English name
	for _, char := range characters {
		if strings.ToLower(char.NameEn) == query {
			return &char
		}
	}

	// Then, try contains match with English name
	for _, char := range characters {
		if strings.Contains(strings.ToLower(char.NameEn), query) {
			return &char
		}
	}

	// Try Japanese name
	for _, char := range characters {
		if strings.Contains(strings.ToLower(char.NameJp), query) {
			return &char
		}
	}

	// Finally, try partial word match
	queryWords := strings.Fields(query)
	for _, char := range characters {
		charNameEn := strings.ToLower(char.NameEn)
		charNameJp := strings.ToLower(char.NameJp)
		for _, word := range queryWords {
			if strings.Contains(charNameEn, word) || strings.Contains(charNameJp, word) {
				return &char
			}
		}
	}

	return nil
}

// GetCharacterImages fetches all images for a character by ID
func (c *Client) GetCharacterImages(charaID int) *shared.CharacterImagesResult {
	// // Check cache first
	// cacheKey := fmt.Sprintf("char_images_%d", charaID)
	// if cached := c.getFromCache(cacheKey); cached != nil {
	// 	if result, ok := cached.(*CharacterImagesResult); ok {
	// 		return result
	// 	}
	// }

	// Make API request
	url := fmt.Sprintf("%s/v1/character/images/%d", c.baseURL, charaID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		result := &shared.CharacterImagesResult{
			Found:   false,
			Error:   fmt.Errorf("failed to fetch character images: %v", err),
			CharaID: charaID,
		}
		// c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &shared.CharacterImagesResult{
			Found:   false,
			Error:   fmt.Errorf("API returned status code: %d", resp.StatusCode),
			CharaID: charaID,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	var images []shared.CharacterImageCategory
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		result := &shared.CharacterImagesResult{
			Found:   false,
			Error:   fmt.Errorf("failed to decode API response: %v", err),
			CharaID: charaID,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	result := &shared.CharacterImagesResult{
		Found:   true,
		Images:  images,
		CharaID: charaID,
	}

	// c.setCache(cacheKey, result)
	return result
}

// SearchSupportCard searches for a support card by name
func (c *Client) SearchSupportCard(query string) *shared.SupportCardSearchResult {
	// // Check cache first
	// cacheKey := fmt.Sprintf("support_search_%s", strings.ToLower(query))
	// if cached := c.getFromCache(cacheKey); cached != nil {
	// 	if result, ok := cached.(*SupportCardSearchResult); ok {
	// 		return result
	// 	}
	// }

	// First, get the list of support cards
	listResult := c.GetSupportCardList()
	if !listResult.Found {
		result := &shared.SupportCardSearchResult{
			Found: false,
			Error: listResult.Error,
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	// Find all matches for the same character
	matches := c.findAllSupportCardMatches(query, listResult.SupportCards)
	if len(matches) == 0 {
		result := &shared.SupportCardSearchResult{
			Found: false,
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	// Get detailed information for all matched cards
	var detailedCards []shared.SupportCard
	for _, match := range matches {
		detailedResult := c.GetSupportCard(match.ID)
		if detailedResult.Found && detailedResult.SupportCard != nil {
			detailedCards = append(detailedCards, *detailedResult.SupportCard)
		}
	}

	if len(detailedCards) == 0 {
		result := &shared.SupportCardSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch detailed information for any matched cards"),
			Query: query,
		}
		// c.setCache(cacheKey, result)
		return result
	}

	// Sort by rarity (SSR > SR > R)
	sort.Slice(detailedCards, func(i, j int) bool {
		rarityOrder := map[string]int{"SSR": 3, "SR": 2, "R": 1}
		return rarityOrder[detailedCards[i].RarityString] > rarityOrder[detailedCards[j].RarityString]
	})

	result := &shared.SupportCardSearchResult{
		Found:        true,
		SupportCard:  &detailedCards[0], // Keep the first one for backward compatibility
		SupportCards: detailedCards,
		Query:        query,
	}

	// c.setCache(cacheKey, result)
	return result
}

// GetSupportCardList fetches the list of all support cards
func (c *Client) GetSupportCardList() *shared.SupportCardListResult {
	// // Check cache first
	// cacheKey := "support_list"
	// if cached := c.getFromCache(cacheKey); cached != nil {
	// 	if result, ok := cached.(*SupportCardListResult); ok {
	// 		return result
	// 	}
	// }

	// Make API request
	url := fmt.Sprintf("%s/v1/support", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		result := &shared.SupportCardListResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch support card list: %v", err),
		}
		// c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &shared.SupportCardListResult{
			Found: false,
			Error: fmt.Errorf("API returned status code: %d", resp.StatusCode),
		}
		// c.setCache(cacheKey, result)
		return result
	}

	var supportCards []shared.SupportCard
	if err := json.NewDecoder(resp.Body).Decode(&supportCards); err != nil {
		result := &shared.SupportCardListResult{
			Found: false,
			Error: fmt.Errorf("failed to decode API response: %v", err),
		}
		// c.setCache(cacheKey, result)
		return result
	}

	result := &shared.SupportCardListResult{
		Found:        true,
		SupportCards: supportCards,
	}

	// c.setCache(cacheKey, result)
	return result
}

// GetSupportCard fetches detailed information for a specific support card
func (c *Client) GetSupportCard(supportID int) *shared.SupportCardSearchResult {
	// // Check cache first
	// cacheKey := fmt.Sprintf("support_detail_%d", supportID)
	// if cached := c.getFromCache(cacheKey); cached != nil {
	// 	if result, ok := cached.(*SupportCardSearchResult); ok {
	// 		return result
	// 	}
	// }

	// Make API request
	url := fmt.Sprintf("%s/v1/support/%d", c.baseURL, supportID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		result := &shared.SupportCardSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to fetch support card details: %v", err),
		}
		// c.setCache(cacheKey, result)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result := &shared.SupportCardSearchResult{
			Found: false,
			Error: fmt.Errorf("API returned status code: %d", resp.StatusCode),
		}
		// c.setCache(cacheKey, result)
		return result
	}

	var supportCard shared.SupportCard
	if err := json.NewDecoder(resp.Body).Decode(&supportCard); err != nil {
		result := &shared.SupportCardSearchResult{
			Found: false,
			Error: fmt.Errorf("failed to decode API response: %v", err),
		}
		// c.setCache(cacheKey, result)
		return result
	}

	result := &shared.SupportCardSearchResult{
		Found:       true,
		SupportCard: &supportCard,
	}

	// c.setCache(cacheKey, result)
	return result
}

// findAllSupportCardMatches finds all support cards that match the query,
// grouping by character ID to find all versions of the same character's support cards.
func (c *Client) findAllSupportCardMatches(query string, supportCards []shared.SupportCard) []shared.SupportCard {
	query = strings.ToLower(query)
	var matches []shared.SupportCard
	var matchedCharIDs []int

	// First pass: find all cards that match the query
	for _, card := range supportCards {
		titleEn := strings.ToLower(card.TitleEn)
		titleJp := strings.ToLower(card.Title)
		gametora := strings.ToLower(card.Gametora)

		// Check if any field contains the query
		matched := false

		// Direct match (highest priority)
		if strings.Contains(titleEn, query) || strings.Contains(titleJp, query) || strings.Contains(gametora, query) {
			matched = true
		}

		// Handle space-separated queries vs hyphen-separated gametora
		if !matched && strings.Contains(query, " ") {
			// Replace spaces with hyphens for gametora matching
			queryWithHyphens := strings.ReplaceAll(query, " ", "-")
			if strings.Contains(gametora, queryWithHyphens) {
				matched = true
			}

			// Also try replacing hyphens with spaces in gametora
			gametoraWithSpaces := strings.ReplaceAll(gametora, "-", " ")
			if strings.Contains(gametoraWithSpaces, query) {
				matched = true
			}
		}

		// Word-by-word matching (only if no direct match found)
		if !matched {
			queryWords := strings.Fields(query)
			// Only do word-by-word matching if we have multiple words
			if len(queryWords) > 1 {
				allWordsMatched := true
				for _, word := range queryWords {
					wordMatched := false
					if strings.Contains(titleEn, word) || strings.Contains(titleJp, word) || strings.Contains(gametora, word) {
						wordMatched = true
					}
					if !wordMatched {
						allWordsMatched = false
						break
					}
				}
				if allWordsMatched {
					matched = true
				}
			}
		}

		if matched {
			matches = append(matches, card)
			matchedCharIDs = append(matchedCharIDs, card.CharaID)
		}
	}

	// If we found matches, also include all other cards for the same characters
	if len(matches) > 0 {
		// Create a set of matched character IDs
		charIDSet := make(map[int]bool)
		for _, id := range matchedCharIDs {
			charIDSet[id] = true
		}

		// Find all cards for the same characters
		var allCardsForCharacter []shared.SupportCard
		for _, card := range supportCards {
			if charIDSet[card.CharaID] {
				allCardsForCharacter = append(allCardsForCharacter, card)
			}
		}

		return allCardsForCharacter
	}

	return matches
}

// // getFromCache retrieves an item from cache
// func (c *Client) getFromCache(key string) interface{} {
// 	c.cacheMutex.RLock()
// 	defer c.cacheMutex.RUnlock()

// 	if entry, exists := c.cache[key]; exists && !entry.IsExpired() {
// 		return entry.Data
// 	}

// 	return nil
// }

// // setCache stores an item in cache
// func (c *Client) setCache(key string, data interface{}) {
// 	c.cacheMutex.Lock()
// 	defer c.cacheMutex.Unlock()

// 	c.cache[key] = &CacheEntry{
// 		Data:      data,
// 		Timestamp: time.Now(),
// 		TTL:       c.cacheTTL,
// 	}
// }
