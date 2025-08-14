package shared

// GametoraSupportsResponse represents the response from the supports.json endpoint
type GametoraSupportsResponse struct {
	PageProps struct {
		SupportData []struct {
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
		} `json:"supportData"`
	} `json:"pageProps"`
}

// SimplifiedSupportCard represents the simplified support card data from Gametora JSON API
type SimplifiedSupportCard struct {
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

// SimplifiedGametoraSearchResult represents the result of a simplified Gametora search
type SimplifiedGametoraSearchResult struct {
	Found        bool
	SupportCard  *SimplifiedSupportCard
	SupportCards []*SimplifiedSupportCard // Multiple cards for the same character
	Error        error
	Query        string
}
