//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"drayya-core/internal/server"
)

func main() {
	app := server.New()
	routes := app.Stack()

	type OpenAPIPath struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Responses   map[string]interface{} `json:"responses"`
	}

	paths := make(map[string]map[string]OpenAPIPath)

	for _, list := range routes {
		for _, r := range list {
			if r.Path == "/" || r.Path == "/metrics" || r.Path == "/health" {
				continue
			}

			// Convert Fiber wildcards/params to OpenAPI path parameters format
			path := r.Path
			path = filepath.ToSlash(path)

			if _, ok := paths[path]; !ok {
				paths[path] = make(map[string]OpenAPIPath)
			}

			method := r.Method
			paths[path][method] = OpenAPIPath{
				Summary:     fmt.Sprintf("Auto-generated stub for %s %s", method, path),
				Description: fmt.Sprintf("Handler: %s", r.Name),
				Responses: map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Successful operation",
					},
				},
			}
		}
	}

	swagger := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Dr.Ayya Automatically Discovered APIs",
			"description": "Auto-generated Swagger spec for undocumented REST paths in the project.",
			"version":     "1.0.0",
		},
		"paths": paths,
	}

	outputFile := filepath.Join(".", "docs", "undocumented_swagger.json")
	_ = os.MkdirAll(filepath.Dir(outputFile), 0755)

	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(swagger); err != nil {
		log.Fatalf("Failed to write JSON: %v", err)
	}

	fmt.Printf("Successfully exported %d path stubs to %s\n", len(paths), outputFile)
}
