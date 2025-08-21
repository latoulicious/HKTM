package tools

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/latoulicious/HKTM/pkg/database"
	"github.com/latoulicious/HKTM/pkg/database/models"
	"gorm.io/gorm"
)

func DBcheck() {
	fmt.Println("=== PostgreSQL Database Connectivity Check ===")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
		// Continue as .env might not exist in production
	}

	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Println("❌ DATABASE_URL environment variable not set")
		os.Exit(1)
	}

	fmt.Printf("📡 Connecting to database...\n")

	// Test database connection
	db, err := database.NewGormDBFromConfig(databaseURL)
	if err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Get underlying SQL DB for connection testing
	sqlDB, err := db.DB()
	if err != nil {
		fmt.Printf("❌ Failed to get underlying database connection: %v\n", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	fmt.Println("✅ Database connection established")

	// Test database ping
	fmt.Printf("🏓 Testing database ping...\n")
	if err := sqlDB.Ping(); err != nil {
		fmt.Printf("❌ Database ping failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Database ping successful")

	// Check database version
	fmt.Printf("🔍 Checking PostgreSQL version...\n")
	var version string
	if err := db.Raw("SELECT version()").Scan(&version).Error; err != nil {
		fmt.Printf("❌ Failed to get database version: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ PostgreSQL version: %s\n", version)

	// Check if uuid-ossp extension exists
	fmt.Printf("🔧 Checking uuid-ossp extension...\n")
	var extensionExists bool
	if err := db.Raw("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp')").Scan(&extensionExists).Error; err != nil {
		fmt.Printf("❌ Failed to check uuid-ossp extension: %v\n", err)
		os.Exit(1)
	}

	if extensionExists {
		fmt.Println("✅ uuid-ossp extension is available")
	} else {
		fmt.Println("⚠️  uuid-ossp extension not found - will be created during migration")
	}

	// Check connection pool stats
	fmt.Printf("📊 Checking connection pool stats...\n")
	stats := sqlDB.Stats()
	fmt.Printf("   - Open connections: %d\n", stats.OpenConnections)
	fmt.Printf("   - In use: %d\n", stats.InUse)
	fmt.Printf("   - Idle: %d\n", stats.Idle)

	// Test basic table operations (if tables exist)
	fmt.Printf("🗃️  Checking existing tables...\n")
	if err := checkExistingTables(db); err != nil {
		fmt.Printf("⚠️  Table check warning: %v\n", err)
	}

	// Test transaction capability
	fmt.Printf("🔄 Testing transaction capability...\n")
	if err := testTransactionCapability(db); err != nil {
		fmt.Printf("❌ Transaction test failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Transaction capability verified")

	// Performance test - simple query timing
	fmt.Printf("⚡ Running performance test...\n")
	start := time.Now()
	var result int
	if err := db.Raw("SELECT 1").Scan(&result).Error; err != nil {
		fmt.Printf("❌ Performance test failed: %v\n", err)
		os.Exit(1)
	}
	duration := time.Since(start)

	fmt.Printf("✅ Simple query completed in %v\n", duration)

	if duration > 5*time.Second {
		fmt.Println("⚠️  Query took longer than 5 seconds - check network latency")
	}

	fmt.Println("\n=== Database Connectivity Check Complete ===")
	fmt.Println("✅ PostgreSQL database is accessible and ready for use")
}

// checkExistingTables checks if the expected tables exist and are accessible
func checkExistingTables(db *gorm.DB) error {
	expectedTables := []string{
		"characters",
		"character_images",
		"support_cards",
		"audio_errors",
		"audio_metrics",
		"audio_logs",
		"queue_timeouts",
	}

	var existingTables []string
	if err := db.Raw(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = current_schema()
		AND table_type = 'BASE TABLE'
	`).Scan(&existingTables).Error; err != nil {
		return fmt.Errorf("failed to query existing tables: %w", err)
	}

	fmt.Printf("   Found %d existing tables\n", len(existingTables))

	// Check if any expected tables exist
	tableMap := make(map[string]bool)
	for _, table := range existingTables {
		tableMap[table] = true
	}

	missingTables := []string{}
	for _, expected := range expectedTables {
		if !tableMap[expected] {
			missingTables = append(missingTables, expected)
		}
	}

	if len(missingTables) > 0 {
		fmt.Printf("   ⚠️  Missing tables (will be created during migration): %v\n", missingTables)
	} else {
		fmt.Println("   ✅ All expected tables exist")

		// Test basic read access on audio_logs table if it exists
		if tableMap["audio_logs"] {
			var count int64
			if err := db.Model(&models.AudioLog{}).Count(&count).Error; err != nil {
				return fmt.Errorf("failed to count audio_logs: %w", err)
			}
			fmt.Printf("   📊 audio_logs table has %d records\n", count)
		}
	}

	return nil
}

// testTransactionCapability tests if the database supports transactions properly
func testTransactionCapability(db *gorm.DB) error {
	// Start a transaction
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	// Try to create a temporary table in the transaction
	if err := tx.Exec("CREATE TEMPORARY TABLE test_transaction (id SERIAL PRIMARY KEY, test_data TEXT)").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create temporary table: %w", err)
	}

	// Insert test data
	if err := tx.Exec("INSERT INTO test_transaction (test_data) VALUES ('test')").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert test data: %w", err)
	}

	// Verify data exists
	var count int64
	if err := tx.Raw("SELECT COUNT(*) FROM test_transaction").Scan(&count).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to count test data: %w", err)
	}

	if count != 1 {
		tx.Rollback()
		return fmt.Errorf("unexpected count in transaction: expected 1, got %d", count)
	}

	// Rollback the transaction (cleanup)
	if err := tx.Rollback().Error; err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	return nil
}
