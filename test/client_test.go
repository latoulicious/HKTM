package test

// import (
// 	"testing"

// 	"github.com/latoulicious/HKTM/pkg/uma"
// )

// // TestNewClient tests the NewClient function
// func TestNewClient(t *testing.T) {
// 	client := uma.NewClient()
// 	if client == nil {
// 		t.Fatal("NewClient() returned nil")
// 	}
// }

// func TestGetCharacterImages(t *testing.T) {
// 	client := uma.NewClient()

// 	// Test with a known character ID (Admire Vega)
// 	result := client.GetCharacterImages(5191)

// 	if !result.Found {
// 		t.Error("Expected to find images for character ID 5191")
// 	}

// 	if len(result.Images) == 0 {
// 		t.Error("Expected to find image categories")
// 	}

// 	// Check that we have the expected categories
// 	expectedCategories := []string{"Uniform", "Racing Outfit", "Concept Art", "Starting Future"}
// 	foundCategories := make(map[string]bool)

// 	for _, category := range result.Images {
// 		foundCategories[category.LabelEn] = true
// 	}

// 	for _, expected := range expectedCategories {
// 		if !foundCategories[expected] {
// 			t.Errorf("Expected category '%s' not found", expected)
// 		}
// 	}

// 	// Check that each category has images
// 	for _, category := range result.Images {
// 		if len(category.Images) == 0 {
// 			t.Errorf("Category '%s' has no images", category.LabelEn)
// 		}

// 		for _, image := range category.Images {
// 			if image.Image == "" {
// 				t.Errorf("Image URL is empty in category '%s'", category.LabelEn)
// 			}
// 		}
// 	}
// }

// // TestSearchCharacterExactMatch tests exact character matching
// func TestSearchCharacterExactMatch(t *testing.T) {
// 	client := uma.NewClient()

// 	// Test exact match using the public SearchCharacter method
// 	result := client.SearchCharacter("Oguri Cap")
// 	if result.Error != nil {
// 		t.Errorf("Expected no error, got %v", result.Error)
// 	}

// 	if !result.Found {
// 		t.Error("Expected to find 'Oguri Cap' with exact match")
// 	}

// 	if result.Character == nil {
// 		t.Error("Expected character to be found, but got nil")
// 	}

// 	if result.Character.NameEn != "Oguri Cap" {
// 		t.Errorf("Expected character name to be 'Oguri Cap', got '%s'", result.Character.NameEn)
// 	}
// }

// // TestSearchCharacterPartialMatch tests partial character matching
// func TestSearchCharacterPartialMatch(t *testing.T) {
// 	client := uma.NewClient()

// 	// Test partial match
// 	result := client.SearchCharacter("oguri")
// 	if result.Error != nil {
// 		t.Errorf("Expected no error, got %v", result.Error)
// 	}

// 	if !result.Found {
// 		t.Error("Expected to find character with partial match 'oguri'")
// 	}

// 	if result.Character == nil {
// 		t.Error("Expected character to be found, but got nil")
// 	}
// }

// // TestSearchCharacterNoMatch tests when no character is found
// func TestSearchCharacterNoMatch(t *testing.T) {
// 	client := uma.NewClient()

// 	// Test no match
// 	result := client.SearchCharacter("nonexistentcharacter12345")
// 	if result.Error != nil {
// 		t.Errorf("Expected no error, got %v", result.Error)
// 	}

// 	if result.Found {
// 		t.Error("Expected no match for 'nonexistentcharacter12345'")
// 	}

// 	if result.Character != nil {
// 		t.Error("Expected no character to be found")
// 	}
// }

// func TestSearchSupportCard(t *testing.T) {
// 	client := uma.NewClient()
// 	result := client.SearchSupportCard("daring tact")

// 	if result.Error != nil {
// 		t.Errorf("Expected no error, got %v", result.Error)
// 	}

// 	if !result.Found {
// 		t.Error("Expected to find support card, but didn't")
// 	}

// 	if result.SupportCard == nil {
// 		t.Error("Expected support card to be found, but got nil")
// 	}

// 	// Check if the support card has the expected properties
// 	if result.SupportCard.ID == 0 {
// 		t.Error("Expected support card to have an ID")
// 	}

// 	if result.SupportCard.TitleEn == "" && result.SupportCard.Title == "" {
// 		t.Error("Expected support card to have a title")
// 	}
// }
