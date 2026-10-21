package v2

import (
	"context"
	"errors"
	"strings"

	"drayya-core/internal/database"
)

// lookupKnowledgeTool searches the KnowledgeItem table — the curated FAQ /
// platform manual / AYUSH protocol content. Read-only; allowed for
// guests and patients.
//
// V1 of this tool is keyword search. Phase 2 will swap to pgvector semantic
// search (the column + embeddings already exist; we just need to call the
// embedding service and compare cosine similarity).
var lookupKnowledgeTool = Tool{
	Name: "lookup_knowledge",
	Description: "Search the Dr.Ayya knowledge base for protocols, FAQ answers, " +
		"and platform how-tos. Use this when the user asks how to do something on the " +
		"platform, what an AYUSH term means, or for general health information. " +
		"Returns up to 3 most relevant articles with title and a short summary.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The user's question or topic. Pass the natural-language form, not keywords.",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Optional category filter: 'protocol', 'faq', 'service', or empty.",
			},
		},
		"required": []string{"query"},
	},
	Allowed:       []PersonaRole{RoleGuest, RolePatient},
	Confirm:       false,
	MaxPerSession: 20,
	MaxPerMinute:  10,
	Execute:       executeLookupKnowledge,
}

type knowledgeResult struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Source  string `json:"source,omitempty"`
}

func executeLookupKnowledge(ctx context.Context, args map[string]any) (any, error) {
	if database.DB == nil {
		return nil, errors.New("database unavailable")
	}
	queryRaw, _ := args["query"].(string)
	query := strings.TrimSpace(queryRaw)
	if query == "" {
		return map[string]any{"articles": []any{}, "total_found": 0}, nil
	}
	category, _ := args["category"].(string)

	// Keyword search across title + content. We use ILIKE-equivalent via
	// LOWER() so this query is portable to SQLite too.
	type row struct {
		ID      uint
		Title   string
		Content string
		Source  string
	}
	var rows []row
	q := database.DB.Table("knowledge_items").
		Select("id, title, content, source").
		Where(`LOWER(title) LIKE ? OR LOWER(content) LIKE ?`,
			"%"+strings.ToLower(query)+"%",
			"%"+strings.ToLower(query)+"%")
	if category != "" {
		q = q.Where("category = ?", category)
	}
	q.Order("updated_at DESC").Limit(3).Scan(&rows)

	out := make([]knowledgeResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, knowledgeResult{
			ID:      r.ID,
			Title:   r.Title,
			Snippet: snippet(r.Content, 240),
			Source:  r.Source,
		})
	}
	return map[string]any{
		"articles":    out,
		"total_found": len(out),
	}, nil
}

// snippet returns the first n characters of s, trimmed at a word boundary
// when possible. Used to keep tool results compact (token budget matters).
func snippet(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	cut := s[:n]
	if idx := strings.LastIndex(cut, " "); idx > n/2 {
		cut = cut[:idx]
	}
	return cut + "…"
}
