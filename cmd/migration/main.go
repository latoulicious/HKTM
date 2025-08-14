package main

import (
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/latoulicious/HKTM/pkg/database"
	"github.com/latoulicious/HKTM/pkg/database/migration"
)

func main() {
	// Parse the command line arguments
	migrateFlag := flag.Bool("migrate", false, "Run the migrations")
	resetFlag := flag.Bool("reset", false, "Reset the database")
	flag.Parse()

	// Load the environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	db, err := database.NewGormDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get SQL database: %v", err)
	}

	defer sqlDB.Close()
	log.Println("Connected to database")

	// Reset Flag
	if *resetFlag {
		log.Println("Resetting database...")

		// Get all table names
		var tables []string
		db.Raw("SELECT tablename FROM pg_tables WHERE schemaname = current_schema()").Scan(&tables)

		log.Printf("Found %d tables to drop", len(tables))

		// Drop each table individually
		for _, table := range tables {
			log.Printf("Dropping table: %s", table)
			result := db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE")
			if result.Error != nil {
				log.Printf("Warning: Failed to drop table %s: %v", table, result.Error)
			}
		}

		log.Println("Database reset successfully")
	}

	log.Println("Running migrations...")

	if err := migration.RunMigration(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations completed successfully")

	// Schema Flag
	if *migrateFlag {
		log.Println("Running migrations...")

		if err := migration.RunMigration(db); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		log.Println("Migrations completed successfully")
	}
}
