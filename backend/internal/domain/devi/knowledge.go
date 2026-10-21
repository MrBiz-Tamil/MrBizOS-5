package devi

import (
	"context"
	"fmt"
	"log"
	"drayya-core/internal/services/embedding"
	"strings"
)

// handleSearchKnowledgeBase performs a semantic search using pgvector
func handleSearchKnowledgeBase(ctx context.Context, e *Engine, args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "Please provide a query to search the knowledge base.", nil
	}

	// 1. Generate embedding for the query
	vec, err := embedding.Generate(query)
	if err != nil {
		return "", fmt.Errorf("failed to generate embedding: %w", err)
	}

	if vec == nil {
		// API key missing or graceful no-op — fall back to keyword search
		return handleKeywordSearch(e, query)
	}

	// 2. Perform vector similarity search (PostgreSQL only)
	if e.svc.DB == nil || e.svc.DB.Dialector.Name() != "postgres" {
		return handleKeywordSearch(e, query)
	}

	var results []struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	// Serialise vector for Postgres
	vecJSON := embedding.MarshalVec(vec)

	// cosine similarity search
	err = e.svc.DB.Raw(`
		SELECT title, content
		FROM knowledge_items
		ORDER BY embedding <=> ?::vector
		LIMIT 3
	`, vecJSON).Scan(&results).Error

	if err != nil {
		log.Printf("RAG Error: %v", err)
		return handleKeywordSearch(e, query)
	}

	if len(results) == 0 {
		return "I couldn't find any specific information about that in our knowledge base.", nil
	}

	var sb strings.Builder
	sb.WriteString("Found the following information:\n\n")
	for _, res := range results {
		sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", res.Title, res.Content))
	}
	return sb.String(), nil
}

func handleKeywordSearch(e *Engine, query string) (string, error) {
	var results []struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if e.svc.DB == nil {
		return "Knowledge base is currently unavailable.", nil
	}

	err := e.svc.DB.Table("knowledge_items").
		Where("title ILIKE ? OR content ILIKE ?", "%"+query+"%", "%"+query+"%").
		Limit(3).
		Scan(&results).Error

	if err != nil {
		return "I encountered an error searching the knowledge base.", nil
	}

	if len(results) == 0 {
		return "I couldn't find any information matching your query.", nil
	}

	var sb strings.Builder
	sb.WriteString("Keyword search results:\n\n")
	for _, res := range results {
		sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", res.Title, res.Content))
	}
	return sb.String(), nil
}
