package main

import (
	"errors"
	"flag"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on environment variables")
	}

	direction := flag.String("direction", "up", "migration direction: up | down")
	steps := flag.Int("steps", 0, "number of steps for down (0 = all)")
	flag.Parse()

	dbURL := getEnv("DATABASE_URL", "postgres://postgres@localhost:5432/vocab?sslmode=disable")
	migrationsPath := getEnv("MIGRATIONS_PATH", "internal/db/migrations")

	m, err := migrate.New("file://"+migrationsPath, dbURL)
	if err != nil {
		log.Fatalf("failed to init migrate: %v", err)
	}

	switch *direction {
	case "up":
		err = m.Up()
	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
	default:
		log.Fatalf("unknown direction: %s (use up or down)", *direction)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("migrations applied successfully")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
