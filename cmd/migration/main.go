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
	flag.Parse()
	resetFlag := flag.Bool("reset", false, "Reset the database")

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

		db.Exec("SET session_replication_role = 'replica';")

		// Drop all tables
		err := db.Exec(`
			DO $$ DECLARE
			r RECORD;
			BEGIN
				FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
					EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
				END LOOP;
			END $$;
		`)

		// Reset to normal state
		db.Exec("SET session_replication_role = 'origin';")

		if err != nil {
			log.Fatalf("Failed to drop tables: %v", err)
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
