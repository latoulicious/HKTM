package shared

import "time"

// Character represents a Uma Musume character
type Character struct {
	ID              int    `json:"id"`
	NameEn          string `json:"name_en"`
	NameJp          string `json:"name_jp"`
	NameEnInternal  string `json:"name_en_internal"`
	CategoryLabel   string `json:"category_label"`
	CategoryLabelEn string `json:"category_label_en"`
	CategoryValue   string `json:"category_value"`
	ColorMain       string `json:"color_main"`
	ColorSub        string `json:"color_sub"`
	PreferredURL    string `json:"preferred_url"`
	RowNumber       int    `json:"row_number"`
	ThumbImg        string `json:"thumb_img"`
}

// CharacterSearchResult represents the result of a character search
type CharacterSearchResult struct {
	Found     bool
	Character *Character
	Error     error
	Query     string
}

// CharacterListResult represents the result of fetching all characters
type CharacterListResult struct {
	Found      bool
	Characters []Character
	Error      error
}

// APIResponse represents the response from umapyoi.net API
type APIResponse []Character

// CacheEntry represents a cached API response
type CacheEntry struct {
	Data      interface{}
	Timestamp time.Time
	TTL       time.Duration
}

// IsExpired checks if the cache entry has expired
func (ce *CacheEntry) IsExpired() bool {
	return time.Since(ce.Timestamp) > ce.TTL
}

// CharacterImage represents a character image from the API
type CharacterImage struct {
	Image    string `json:"image"`
	Uploaded string `json:"uploaded"`
}

// CharacterImageCategory represents a category of character images
type CharacterImageCategory struct {
	Images  []CharacterImage `json:"images"`
	Label   string           `json:"label"`
	LabelEn string           `json:"label_en"`
}

// CharacterImagesResult represents the result of fetching character images
type CharacterImagesResult struct {
	Found   bool
	Images  []CharacterImageCategory
	Error   error
	CharaID int
}

// SupportCard represents a support card from the API
type SupportCard struct {
	CharaID      int    `json:"chara_id"`
	Gametora     string `json:"gametora"`
	ID           int    `json:"id"`
	TitleEn      string `json:"title_en"`
	Title        string `json:"title"`
	Rarity       int    `json:"rarity"`
	RarityString string `json:"rarity_string"`
	StartDate    int64  `json:"start_date"`
	Type         string `json:"type"`
	TypeIconURL  string `json:"type_icon_url"`
}

// SupportCardSearchResult represents the result of a support card search
type SupportCardSearchResult struct {
	Found        bool
	SupportCard  *SupportCard
	SupportCards []SupportCard // Multiple cards for the same character
	Error        error
	Query        string
}

// SupportCardListResult represents the result of fetching support card list
type SupportCardListResult struct {
	Found        bool
	SupportCards []SupportCard
	Error        error
}
