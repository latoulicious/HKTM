package test

// import (
// 	"fmt"
// 	"log"
// 	"testing"
// 	"time"

// 	"github.com/latoulicious/HKTM/internal/config"
// 	"github.com/latoulicious/HKTM/pkg/database"
// 	"github.com/latoulicious/HKTM/pkg/uma"
// )

// func TestCache(t *testing.T) {
// 	// Initialize database
// 	db, err := database.NewDatabase("test_cache.db")
// 	if err != nil {
// 		log.Fatalf("Failed to initialize database: %v", err)
// 	}
// 	defer db.Close()

// 	// Initialize Gametora client with minimal config
// 	cfg := &config.Config{
// 		CronEnabled:  false, // Disable cron for testing
// 		CronSchedule: "0 0 */6 * * *",
// 	}
// 	gametoraClient := uma.NewGametoraClient(cfg)

// 	// Test character search caching
// 	fmt.Println("Testing character search caching...")

// 	// First search (should hit API)
// 	fmt.Println("1. First search for 'Oguri Cap'...")
// 	start := time.Now()
// 	result1 := uma.NewClient().SearchCharacter("Oguri Cap")
// 	duration1 := time.Since(start)
// 	fmt.Printf("   Found: %v, Duration: %v\n", result1.Found, duration1)

// 	// Cache the result
// 	if result1 != nil {
// 		if err := db.CacheCharacterSearch("Oguri Cap", result1, 24*time.Hour); err != nil {
// 			fmt.Printf("   Failed to cache: %v\n", err)
// 		} else {
// 			fmt.Println("   Cached successfully")
// 		}
// 	} else {
// 		fmt.Println("   No result to cache (result is nil)")
// 	}

// 	// Second search (should hit cache)
// 	fmt.Println("2. Second search for 'Oguri Cap'...")
// 	start = time.Now()
// 	cached, err := db.GetCachedCharacterSearch("Oguri Cap")
// 	duration2 := time.Since(start)
// 	if err != nil {
// 		fmt.Printf("   Cache error: %v\n", err)
// 	} else if cached != nil {
// 		fmt.Printf("   Found in cache: %v, Duration: %v\n", cached.Found, duration2)
// 	} else {
// 		fmt.Println("   Not found in cache")
// 	}

// 	// Test Gametora skills caching
// 	fmt.Println("\nTesting Gametora skills caching...")

// 	// First search (should hit API)
// 	fmt.Println("1. First search for 'daring tact'...")
// 	start = time.Now()
// 	result2 := gametoraClient.SearchSimplifiedSupportCard("daring tact")
// 	duration3 := time.Since(start)
// 	fmt.Printf("   Found: %v, Duration: %v\n", result2.Found, duration3)

// 	// Cache the result
// 	if result2 != nil {
// 		if err := db.CacheGametoraSkills("daring tact", result2, 24*time.Hour); err != nil {
// 			fmt.Printf("   Failed to cache: %v\n", err)
// 		} else {
// 			fmt.Println("   Cached successfully")
// 		}
// 	} else {
// 		fmt.Println("   No result to cache (result is nil)")
// 	}

// 	// Second search (should hit cache)
// 	fmt.Println("2. Second search for 'daring tact'...")
// 	start = time.Now()
// 	cached2, err := db.GetCachedGametoraSkills("daring tact")
// 	duration4 := time.Since(start)
// 	if err != nil {
// 		fmt.Printf("   Cache error: %v\n", err)
// 	} else if cached2 != nil {
// 		fmt.Printf("   Found in cache: %v, Duration: %v\n", cached2.Found, duration4)
// 	} else {
// 		fmt.Println("   Not found in cache")
// 	}

// 	// Show cache statistics
// 	fmt.Println("\nCache statistics:")
// 	stats, err := db.GetCacheStats()
// 	if err != nil {
// 		fmt.Printf("Failed to get stats: %v\n", err)
// 	} else {
// 		for name, count := range stats {
// 			fmt.Printf("  %s: %d\n", name, count)
// 		}
// 	}

// 	fmt.Println("\nTest completed!")
// }
