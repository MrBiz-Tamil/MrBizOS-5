package ceo_ai

import (
	"time"
)

// ExecutiveQueryRequest represents an incoming natural language prompt from executive management.
type ExecutiveQueryRequest struct {
	Query       string   `json:"query"`
	UserID      uint     `json:"user_id"`
	Permissions []string `json:"permissions"`
}

// ExecutiveQueryResponse represents synthesized executive insights.
type ExecutiveQueryResponse struct {
	Query            string            `json:"query"`
	Answer           string            `json:"answer"`
	DataSources      []string          `json:"data_sources"`
	BusinessUnit     string            `json:"business_unit,omitempty"`
	MetricsSummary   map[string]any    `json:"metrics_summary,omitempty"`
	SuggestedActions []string          `json:"suggested_actions,omitempty"`
	GeneratedAt      time.Time         `json:"generated_at"`
}
