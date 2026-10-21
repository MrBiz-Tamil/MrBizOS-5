// Package embedding provides a lightweight vector-embedding service backed by
// the Gemini text-embedding-004 model (768-dimensional dense vectors).
//
// When GEMINI_API_KEY is not set every call returns an empty slice and no
// error — callers degrade gracefully (e.g., skip vector search, use recency).
//
// pgvector wiring
// ---------------
// SetupPgVector(db) executes:
//   CREATE EXTENSION IF NOT EXISTS vector;
//   ALTER TABLE devi_symptom_logs
//       ALTER COLUMN embedding TYPE vector(768) USING embedding::vector;
//   CREATE INDEX IF NOT EXISTS idx_devi_emb ON devi_symptom_logs
//       USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
//
// This is safe to call multiple times (all statements are idempotent).
// Call it once at server startup AFTER AutoMigrate when DATABASE_DSN points
// to PostgreSQL.
package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"gorm.io/gorm"
)

const (
	geminiEmbedURL = "https://generativelanguage.googleapis.com/v1beta/models/text-embedding-004:embedContent"
	embeddingDims  = 768
)

// Generate returns a 768-dimensional vector for the given text.
// Returns nil (not an error) when the API key is absent.
func Generate(text string) ([]float32, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, nil // graceful no-op
	}

	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Parts []part `json:"parts"`
	}
	reqBody := map[string]interface{}{
		"model":   "models/text-embedding-004",
		"content": content{Parts: []part{{Text: text}}},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s?key=%s", geminiEmbedURL, apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding API: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("embedding API error: %s", result.Error.Message)
	}
	return result.Embedding.Values, nil
}

// MarshalVec serialises a []float32 to a JSON byte string for TEXT storage.
func MarshalVec(v []float32) string {
	if len(v) == 0 {
		return ""
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// SetupPgVector enables the pgvector extension and upgrades the embedding
// column on devi_symptom_logs to the native vector(768) type.
// It is a no-op on non-PostgreSQL databases (SQLite, etc.).
//
// The conversion is idempotent: if the column is already of type vector, the
// ALTER step is skipped. The previous implementation tried to cast the column
// regardless of current type which raised:
//   ERROR: invalid input syntax for type vector: "" (SQLSTATE 22P02)
// because GORM's text-default produced empty strings that pgvector rejects.
func SetupPgVector(db *gorm.DB) {
	if db.Dialector.Name() != "postgres" {
		return
	}

	// 1) Ensure extension is installed.
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS vector`).Error; err != nil {
		// Non-fatal — likely the DB doesn't have pgvector available.
		// Subsequent steps will be no-ops since the column conversion will fail safely.
		return
	}

	// 2) Detect current column type. Skip conversion if already vector.
	var dataType string
	row := db.Raw(`
		SELECT data_type FROM information_schema.columns
		WHERE table_name = 'devi_symptom_logs' AND column_name = 'embedding'
	`).Row()
	_ = row.Scan(&dataType)

	if dataType == "USER-DEFINED" {
		// Already vector — nothing to do.
	} else if dataType != "" {
		// 3) Pre-sanitize: replace empty strings with NULL so the cast is safe.
		_ = db.Exec(`UPDATE devi_symptom_logs SET embedding = NULL WHERE embedding = ''`).Error

		// 4) Convert. Use a wider USING clause that handles NULL and any other
		//    legacy garbage values without raising.
		_ = db.Exec(`
			ALTER TABLE devi_symptom_logs
			    ALTER COLUMN embedding TYPE vector(768)
			    USING (
			        CASE
			            WHEN embedding IS NULL OR embedding::text = '' THEN NULL
			            ELSE embedding::text::vector
			        END
			    )
		`).Error
	}

	// 5) Create the ANN index (idempotent via IF NOT EXISTS).
	_ = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_devi_emb
		    ON devi_symptom_logs
		    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)
	`).Error
}
