package uma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// DebugSearchSupportCard is a debug function that prints detailed information about the search process
func (c *GametoraClient) DebugSearchSupportCard(query string) {
	fmt.Printf("🔍 Debugging search for: %s\n", query)

	// Get build ID
	buildID, err := c.GetBuildID()
	if err != nil {
		fmt.Printf("❌ Failed to get build ID: %v\n", err)
		return
	}
	fmt.Printf("📦 Build ID: %s\n", buildID)

	// Fetch supports list
	supportsURL := fmt.Sprintf("%s/%s/umamusume/supports.json", c.baseURL, buildID)
	fmt.Printf("🌐 Fetching from: %s\n", supportsURL)

	resp, err := c.httpClient.Get(supportsURL)
	if err != nil {
		fmt.Printf("❌ Failed to fetch supports: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ API returned status: %d\n", resp.StatusCode)
		return
	}

	var supportsResp GametoraSupportsResponse
	if err := json.NewDecoder(resp.Body).Decode(&supportsResp); err != nil {
		fmt.Printf("❌ Failed to decode response: %v\n", err)
		return
	}

	fmt.Printf("📊 Found %d support cards\n", len(supportsResp.PageProps.SupportData))

	// Show first few cards for reference
	fmt.Printf("\n📋 First 5 support cards:\n")
	for i, support := range supportsResp.PageProps.SupportData {
		if i >= 5 {
			break
		}
		fmt.Printf("  %d. %s (%s) - %s\n", i+1, support.NameJp, support.NameJp, support.CharName)
	}

	// Search for the query
	query = strings.ToLower(strings.TrimSpace(query))
	fmt.Printf("\n🔎 Searching for: '%s'\n", query)

	var matches []struct {
		Support struct {
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
		Score  int
		Reason string
	}

	for _, support := range supportsResp.PageProps.SupportData {
		urlName := strings.ToLower(support.URLName)
		charName := strings.ToLower(support.CharName)

		score := 0
		reason := ""

		// Exact matches
		if urlName == query || charName == query {
			score = 100
			reason = "exact match"
		} else if strings.HasPrefix(urlName, query) || strings.HasPrefix(charName, query) {
			score = 80
			reason = "prefix match"
		} else if strings.Contains(urlName, query) || strings.Contains(charName, query) {
			score = 60
			reason = "contains match"
		} else {
			// Word-by-word matching
			queryWords := strings.Fields(query)
			for _, word := range queryWords {
				if len(word) > 2 {
					if strings.Contains(urlName, word) || strings.Contains(charName, word) {
						score += 10
						reason = "word match"
					}
				}
			}
		}

		if score > 0 {
			matches = append(matches, struct {
				Support struct {
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
				Score  int
				Reason string
			}{
				Support: struct {
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
					Hints: struct {
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
					}{
						HintSkills: support.Hints.HintSkills,
						HintOthers: support.Hints.HintOthers,
					},
					EventSkills: support.EventSkills,
					Unique:      support.Unique,
				},
				Score:  score,
				Reason: reason,
			})
		}
	}

	if len(matches) == 0 {
		fmt.Printf("❌ No matches found for '%s'\n", query)
	} else {
		fmt.Printf("✅ Found %d potential matches:\n", len(matches))
		for i, match := range matches {
			fmt.Printf("  %d. %s (%s) - Score: %d (%s)\n",
				i+1, match.Support.NameJp, match.Support.NameJp, match.Score, match.Reason)
		}
	}
}
