package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("ℹ️  No .env file found, continuing...")
	}
}

func GetDatabaseURL() string {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatalln("❌ DATABASE_URL not set (in .env or environment)")
	}
	return url
}
