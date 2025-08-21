package test

// import (
// 	"testing"

// 	"github.com/latoulicious/HKTM/internal/config"
// 	"github.com/latoulicious/HKTM/pkg/uma"
// 	"github.com/latoulicious/HKTM/pkg/uma/navigation"
// )

// // createTestConfig creates a test configuration
// func createTestConfig() *config.Config {
// 	return &config.Config{
// 		DiscordToken: "test-token",
// 		OwnerID:      "test-owner",
// 		CronEnabled:  true,
// 		CronSchedule: "0 0 */6 * * *", // Every 6 hours
// 	}
// }

// // TestSupportCardSearch tests the support card search functionality
// func TestSupportCardSearch(t *testing.T) {
// 	query := "10001-special-week" // Use a known support card for testing
// 	t.Logf("🔍 Testing Uma search for: %s", query)

// 	// Create Gametora client with test config
// 	cfg := createTestConfig()
// 	client := uma.NewGametoraClient(cfg)

// 	// Test the actual search
// 	result := client.SearchSimplifiedSupportCard(query)
// 	if !result.Found {
// 		t.Fatalf("Support card not found: %s", query)
// 	}

// 	if result.SupportCard == nil {
// 		t.Fatal("Expected support card to be found, but got nil")
// 	}

// 	t.Logf("✅ Found support card: %s", result.SupportCard.NameJp)
// 	t.Logf("   Character: %s", result.SupportCard.CharName)
// 	t.Logf("   Rarity: %s", uma.GetRarityText(result.SupportCard.Rarity))
// 	t.Logf("   Type: %s", result.SupportCard.Type)
// 	t.Logf("   Support ID: %d", result.SupportCard.SupportID)
// 	t.Logf("   URL Name: %s", result.SupportCard.URLName)

// 	// Test image URL generation
// 	imageURL := client.GetSupportCardImageURL(result.SupportCard.URLName)
// 	if imageURL == "" {
// 		t.Error("Expected image URL to be generated")
// 	}
// 	t.Logf("   Image URL: %s", imageURL)
// }

// // TestSupportCardMultipleVersions tests when multiple versions of a support card are found
// func TestSupportCardMultipleVersions(t *testing.T) {
// 	query := "10001-special-week" // This should have multiple versions
// 	cfg := createTestConfig()
// 	client := uma.NewGametoraClient(cfg)

// 	result := client.SearchSimplifiedSupportCard(query)
// 	if !result.Found {
// 		t.Fatalf("Support card not found: %s", query)
// 	}

// 	if len(result.SupportCards) < 2 {
// 		t.Skip("This test requires multiple versions of a support card")
// 	}

// 	t.Logf("📋 All versions found (%d):", len(result.SupportCards))
// 	for i, card := range result.SupportCards {
// 		t.Logf("  %d. %s (%s) - ID: %d", i+1, card.NameJp, uma.GetRarityText(card.Rarity), card.SupportID)
// 	}

// 	// Test navigation embed creation
// 	t.Log("🧭 Testing navigation embed for version 1 (SSR):")
// 	navManager := navigation.GetSupportCardNavigationManager()
// 	navEmbed := navManager.CreateSupportCardEmbed(result.SupportCards[0], result.SupportCards, 0)

// 	if navEmbed.Title == "" {
// 		t.Error("Expected navigation embed to have a title")
// 	}

// 	if navEmbed.Footer.Text == "" {
// 		t.Error("Expected navigation embed to have footer text")
// 	}

// 	t.Logf("  Title: %s", navEmbed.Title)
// 	t.Logf("  Footer: %s", navEmbed.Footer.Text)
// 	t.Logf("  Fields: %d", len(navEmbed.Fields))
// }

// // TestSupportCardHints tests support card hints functionality
// func TestSupportCardHints(t *testing.T) {
// 	query := "10001-special-week"
// 	cfg := createTestConfig()
// 	client := uma.NewGametoraClient(cfg)

// 	result := client.SearchSimplifiedSupportCard(query)
// 	if !result.Found {
// 		t.Fatalf("Support card not found: %s", query)
// 	}

// 	if len(result.SupportCard.Hints.HintSkills) > 0 {
// 		t.Logf("💡 Support Hints (%d):", len(result.SupportCard.Hints.HintSkills))
// 		for i, hint := range result.SupportCard.Hints.HintSkills {
// 			t.Logf("  %d. %s", i+1, hint.NameEn)
// 		}
// 	} else {
// 		t.Log("No support hints found for this card")
// 	}
// }

// // TestSupportCardEventSkills tests support card event skills functionality
// func TestSupportCardEventSkills(t *testing.T) {
// 	query := "10001-special-week"
// 	cfg := createTestConfig()
// 	client := uma.NewGametoraClient(cfg)

// 	result := client.SearchSimplifiedSupportCard(query)
// 	if !result.Found {
// 		t.Fatalf("Support card not found: %s", query)
// 	}

// 	if len(result.SupportCard.EventSkills) > 0 {
// 		t.Logf("🎉 Event Skills (%d):", len(result.SupportCard.EventSkills))
// 		for i, event := range result.SupportCard.EventSkills {
// 			t.Logf("  %d. %s", i+1, event.NameEn)
// 		}
// 	} else {
// 		t.Log("No event skills found for this card")
// 	}
// }

// // TestSupportCardNotFound tests the behavior when a support card is not found
// func TestSupportCardNotFound(t *testing.T) {
// 	query := "nonexistent-support-card-12345"
// 	cfg := createTestConfig()
// 	client := uma.NewGametoraClient(cfg)

// 	result := client.SearchSimplifiedSupportCard(query)
// 	if result.Found {
// 		t.Error("Expected support card not to be found")
// 	}

// 	if result.SupportCard != nil {
// 		t.Error("Expected support card to be nil when not found")
// 	}

// 	t.Logf("❌ Support card not found: %s", query)
// 	if result.Error != nil {
// 		t.Logf("   Error: %v", result.Error)
// 	}
// }

// // // TestSupportCardDebugSearch tests the debug search functionality
// // func TestSupportCardDebugSearch(t *testing.T) {
// // 	query := "10001-special-week"
// // 	cfg := createTestConfig()
// // 	client := uma.NewGametoraClient(cfg)

// // 	// Test debug search (this should not fail)
// // 	client.DebugSearchSupportCard(query)
// // 	t.Logf("🔍 Debug search completed for: %s", query)
// // }

// // TestSupportCardIntegration tests the complete support card functionality
// func TestSupportCardIntegration(t *testing.T) {
// 	// This test runs all the individual components to ensure they work together
// 	t.Run("basic search", TestSupportCardSearch)
// 	t.Run("multiple versions", TestSupportCardMultipleVersions)
// 	t.Run("support hints", TestSupportCardHints)
// 	t.Run("event skills", TestSupportCardEventSkills)
// 	t.Run("not found", TestSupportCardNotFound)
// 	// t.Run("debug search", TestSupportCardDebugSearch)

// 	t.Log("🎉 All support card functionality tests completed!")
// }
