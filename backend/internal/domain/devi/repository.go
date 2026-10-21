package devi

import (
	"encoding/json"
	"math"
	"sort"

	"gorm.io/gorm"
)

// Repository handles persistence for Devi symptom logs.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Save persists a SymptomLog entry.
func (r *Repository) Save(log *SymptomLog) error {
	return r.db.Create(log).Error
}

// GetUserHistory returns the most recent N logs for a user, newest first.
func (r *Repository) GetUserHistory(userID uint, limit int) ([]SymptomLog, error) {
	var logs []SymptomLog
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// DeleteUserHistory removes all logs for a user (GDPR right-to-erasure).
func (r *Repository) DeleteUserHistory(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&SymptomLog{}).Error
}

// GetFamilyMemberHistory returns the most recent N symptom logs for a specific
// family member under a given parent user account, newest first.
func (r *Repository) GetFamilyMemberHistory(userID, familyMemberID uint, limit int) ([]SymptomLog, error) {
	var logs []SymptomLog
	err := r.db.
		Where("user_id = ? AND family_member_id = ?", userID, familyMemberID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// DeleteFamilyMemberHistory removes all symptom logs for a specific family member
// under a parent user account (GDPR right-to-erasure for dependents).
func (r *Repository) DeleteFamilyMemberHistory(userID, familyMemberID uint) error {
	return r.db.
		Where("user_id = ? AND family_member_id = ?", userID, familyMemberID).
		Delete(&SymptomLog{}).Error
}

// FindSimilar finds the top-k most semantically similar past symptom logs
// for a user. On PostgreSQL with pgvector it uses the <=> (cosine distance)
// operator; on SQLite/test environments it falls back to in-process cosine
// similarity computed across the most recent 200 logs.
func (r *Repository) FindSimilar(userID uint, embedding []float32, limit int) ([]SymptomLog, error) {
	if len(embedding) == 0 {
		return r.GetUserHistory(userID, limit)
	}

	// Check whether the column is a real pgvector type.
	// We detect this by checking the dialect name.
	dialect := r.db.Dialector.Name()
	if dialect == "postgres" {
		return r.findSimilarPgVector(userID, embedding, limit)
	}
	return r.findSimilarInProcess(userID, embedding, limit)
}

// findSimilarPgVector uses the pgvector <=> cosine-distance operator.
func (r *Repository) findSimilarPgVector(userID uint, embedding []float32, limit int) ([]SymptomLog, error) {
	vecJSON, err := json.Marshal(embedding)
	if err != nil {
		return nil, err
	}
	// Cast the JSON array literal to vector for comparison.
	var logs []SymptomLog
	err = r.db.Raw(`
		SELECT * FROM devi_symptom_logs
		WHERE user_id = ?
		  AND embedding IS NOT NULL
		  AND embedding != ''
		ORDER BY embedding::vector <=> ?::vector
		LIMIT ?
	`, userID, string(vecJSON), limit).Scan(&logs).Error
	return logs, err
}

// findSimilarInProcess deserialises all embeddings and computes cosine
// similarity in memory (used for SQLite / tests).
func (r *Repository) findSimilarInProcess(userID uint, queryVec []float32, limit int) ([]SymptomLog, error) {
	var logs []SymptomLog
	if err := r.db.
		Where("user_id = ? AND embedding != ''", userID).
		Order("created_at DESC").
		Limit(200).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	type scored struct {
		log   SymptomLog
		score float32
	}
	var results []scored
	for _, l := range logs {
		var vec []float32
		if err := json.Unmarshal([]byte(l.Embedding), &vec); err != nil {
			continue
		}
		results = append(results, scored{log: l, score: cosineSimilarity(queryVec, vec)})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	out := make([]SymptomLog, len(results))
	for i, r := range results {
		out[i] = r.log
	}
	return out, nil
}

// cosineSimilarity computes the cosine similarity between two float32 vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return float32(dot / denom)
}
