package main

import (
	"log"
	"drayya-core/internal/database"
	"os"
)

func main() {
	f, err := os.Create("debug_db.log")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)

	os.Setenv("USE_SQLITE", "true")
	os.Setenv("DB_URL", ":memory:")

	log.Println("Starting DB connection Debug")

	// This might exit the process if it fails
	database.Connect()

	log.Println("DB connection successful")
}
