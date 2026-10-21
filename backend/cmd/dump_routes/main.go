package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"drayya-core/internal/database"
	"drayya-core/internal/server"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 1. Setup a dummy SQLite database so server.New() doesn't panic on nil DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to initialize dummy db: %v", err)
	}
	database.DB = db

	// Initialize the server (registers all routes)
	// We set SKIP_LIMITER to avoid any issues with rate limiters firing during init
	os.Setenv("SKIP_LIMITER", "true")
	app := server.New()

	// 2. Extract routes
	type RouteInfo struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	}

	var routes []RouteInfo
	for _, routeList := range app.Stack() {
		for _, route := range routeList {
			// Ignore middleware (use len(route.Handlers) to check if it's purely middleware?
			// Actually Fiber adds methods for middleware like "USE", we can filter standard methods
			if route.Method == "USE" || route.Method == "" {
				continue
			}
			routes = append(routes, RouteInfo{
				Method: route.Method,
				Path:   route.Path,
			})
		}
	}

	// 3. Write to stdout or file
	out, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal routes: %v", err)
	}

	if len(os.Args) > 1 {
		err = os.WriteFile(os.Args[1], out, 0644)
		if err != nil {
			log.Fatalf("Failed to write to file: %v", err)
		}
		fmt.Printf("Successfully exported %d routes to %s\n", len(routes), os.Args[1])
	} else {
		fmt.Println(string(out))
	}
}
