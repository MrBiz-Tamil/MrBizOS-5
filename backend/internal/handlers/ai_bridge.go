package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"drayya-core/internal/database"
	devdomain "drayya-core/internal/domain/devi"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/resources"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// deviMemoryEngine is the minimal interface the handlers need from Devi service.
type deviMemoryEngine interface {
	AnalyzeSymptomsLLM(ctx context.Context, symptoms, duration, severity string) (*devdomain.SymptomAnalysis, error)
}

// deviMemEng is set via SetDeviMemoryEngine at server startup.
var deviMemEng deviMemoryEngine

// SetDeviMemoryEngine wires the Devi service instance (called from server/app.go).
func SetDeviMemoryEngine(e deviMemoryEngine) { deviMemEng = e }

// DynamicSymptom structure for JSON loading matches the new dataset design
type DynamicSymptom struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Departments []string `json:"departments"`
	Urgency     int      `json:"urgency"`
	Aliases     []string `json:"aliases"`
}

type SymptomDataset struct {
	Symptoms []DynamicSymptom `json:"symptoms"`
}

// SymptomDataset singleton for in-memory caching
var (
	symptomCache *SymptomDataset
	cacheOnce    sync.Once
)

// LoadSymptomDataset - uses resources.SymptomMappingJSON (go:embed, compile-time safe)
func LoadSymptomDataset() *SymptomDataset {
	cacheOnce.Do(func() {
		var dataset SymptomDataset
		if err := json.Unmarshal(resources.SymptomMappingJSON, &dataset); err != nil {
			fmt.Printf("[AI Bridge] ERROR: failed to parse symptom_mapping.json: %v\n", err)
			return
		}
		if len(dataset.Symptoms) == 0 {
			fmt.Println("[AI Bridge] WARNING: symptom dataset loaded but contains 0 entries")
			return
		}
		fmt.Printf("[AI Bridge] Loaded %d symptoms into memory cache\n", len(dataset.Symptoms))
		symptomCache = &dataset
	})
	return symptomCache
}

type SymptomRequest struct {
	Symptoms         interface{} `json:"symptoms"`
	PrimarySymptom   string      `json:"primary_symptom"`
	Duration         string      `json:"duration"`
	Severity         string      `json:"severity"`
	Language         string      `json:"language"`
	GenderPreference string      `json:"gender_preference"`
	PatientID        uint        `json:"patient_id"`
}

// AIBridge acts as a local, rule-based dynamic symptom matcher.
func AIBridge(c *fiber.Ctx) error {
	targetPath := strings.TrimPrefix(c.Path(), "/api/v1/ai")
	if targetPath == "" {
		targetPath = "/"
	}

	if targetPath == "/analyze-symptoms" || targetPath == "/match-instant" {
		var req SymptomRequest
		if err := c.BodyParser(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_FORMAT", "Invalid request format")
		}

		var patientInputs []string
		isDetailedDescription := false

		if req.PrimarySymptom != "" {
			patientInputs = append(patientInputs, req.PrimarySymptom)
			if len(req.PrimarySymptom) > 20 {
				isDetailedDescription = true
			}
		}

		switch v := req.Symptoms.(type) {
		case string:
			if v != "" {
				patientInputs = append(patientInputs, strings.ToLower(v))
				if len(v) > 20 {
					isDetailedDescription = true
				}
			}
		case []interface{}:
			for _, val := range v {
				if str, ok := val.(string); ok {
					patientInputs = append(patientInputs, strings.ToLower(str))
				}
			}
		}

		// AI-DRIVEN BRANCH: If we have a detailed description and Gemini is active, use LLM Analyzer
		if isDetailedDescription && os.Getenv("GEMINI_API_KEY") != "" && deviMemEng != nil {
			fullText := strings.Join(patientInputs, ". ")
			analysis, err := deviMemEng.AnalyzeSymptomsLLM(c.Context(), fullText, req.Duration, req.Severity)
			if err == nil && analysis != nil {
				highestMatch := analysis.RecommendedSpecialization
				if highestMatch == "" && len(analysis.RecommendedSpecializations) > 0 {
					highestMatch = analysis.RecommendedSpecializations[0]
				}
				
				matchedProfile, matchedUser := matchDoctorLogic(highestMatch, req.Language, req.GenderPreference, req.PatientID)

				return c.JSON(fiber.Map{
					"urgency_level":               analysis.UrgencyLevel,
					"recommended_specializations": analysis.RecommendedSpecializations,
					"recommended_specialization":  highestMatch,
					"confidence":                  analysis.Confidence,
					"consultation_mode":           analysis.ConsultationMode,
					"action":                      analysis.Action,
					"ai_assessment":               analysis.AIAssessment,
					"severity_score":              analysis.SeverityScore,
					"matched_doctor": fiber.Map{
						"found":   matchedProfile != nil,
						"profile": matchedProfile,
						"user":    matchedUser,
					},
				})
			}
		}

		dataset := LoadSymptomDataset()
		if dataset == nil {
			return response.Error(c, fiber.StatusInternalServerError, "SYMPTOM_DATASET_LOAD_FAILED", "Failed to load symptom dataset")
		}

		departmentScores := make(map[string]int)
		maxUrgency := 0

		for _, input := range patientInputs {
			for _, symptom := range dataset.Symptoms {
				matchFound := false
				if strings.Contains(input, strings.ToLower(symptom.Name)) {
					matchFound = true
				} else {
					for _, alias := range symptom.Aliases {
						if strings.Contains(input, strings.ToLower(alias)) {
							matchFound = true
							break
						}
					}
				}

				if matchFound {
					for _, dept := range symptom.Departments {
						departmentScores[dept] += 1
					}
					if symptom.Urgency > maxUrgency {
						maxUrgency = symptom.Urgency
					}
				}
			}
		}

		highestMatch := "General Medicine"
		highestScore := 0
		var allMatched []string

		for dept, score := range departmentScores {
			allMatched = append(allMatched, dept)
			if score > highestScore {
				highestScore = score
				highestMatch = dept
			}
		}

		if highestScore == 0 {
			allMatched = []string{"General Medicine"}
			highestMatch = "General Medicine"
		}

		// CASE 1: True Medical Emergency (SOS Protocol)
		if maxUrgency >= 9 {
			return c.JSON(fiber.Map{
				"urgency_level":               "critical",
				"recommended_specializations": allMatched,
				"action":                      "trigger_sos",
				"consultation_mode":           "emergency",
				"ai_assessment":               "Critical symptoms detected. Bypassing video consultation and triggering SOS dispatch protocol.",
				"severity_score":              maxUrgency * 10,
			})
		}

		matchedProfile, matchedUser := matchDoctorLogic(highestMatch, req.Language, req.GenderPreference, req.PatientID)

		urgencyStr := "low"
		if maxUrgency >= 7 {
			urgencyStr = "high"
		} else if maxUrgency >= 4 {
			urgencyStr = "moderate"
		}

		return c.JSON(fiber.Map{
			"urgency_level":               urgencyStr,
			"recommended_specializations": allMatched,
			"recommended_specialization":  highestMatch,
			"confidence":                  0.95,
			"consultation_mode":           "video",
			"action":                      "match_doctor",
			"ai_assessment":               "Based on the selected symptoms, we recommend consulting a " + highestMatch + " specialist.",
			"severity_score":              maxUrgency * 10,
			"matched_doctor": fiber.Map{
				"found":   matchedProfile != nil,
				"profile": matchedProfile,
				"user":    matchedUser,
			},
		})
	}

	return response.Error(c, fiber.StatusNotFound, "AI_ENDPOINT_NOT_FOUND", "AI endpoint not found.")
}

func matchDoctorLogic(highestMatch, language, genderPref string, patientID uint) (*identity.DoctorProfile, *identity.User) {
	// Business Rules & Patient Constraints
	var maxFee int64 = -1
	var minRating float64 = 0
	if patientID > 0 {
		var patient identity.User
		if err := database.DB.First(&patient, patientID).Error; err == nil {
			maxFee = patient.WalletBalance
			// Premium tier gets highly rated doctors
			if patient.Tier == "premium" || patient.IsVerifiedPremium {
				minRating = 4.5
			}
		}
	}

	// Finding Doctor Logic
	activeThreshold := time.Now().Add(-5 * time.Minute)

	findDoctor := func(specialty, lang, gender string, minR float64, maxF int64) *identity.DoctorProfile {
		var profile identity.DoctorProfile
		query := database.DB.Model(&identity.DoctorProfile{}).
			Where("is_online = ?", true).
			Where("is_verified = ?", true).
			Where("current_status = ?", "available").
			Where("last_ping > ?", activeThreshold)

		if specialty != "" {
			query = query.Where("specialization = ?", specialty)
		}
		if lang != "" {
			query = query.Where("languages_spoken LIKE ?", "%"+strings.ToLower(lang)+"%")
		}
		if gender != "" {
			query = query.Where("LOWER(gender) = ?", strings.ToLower(gender))
		}
		if minR > 0 {
			query = query.Where("average_rating >= ?", minR)
		}
		if maxF >= 0 {
			query = query.Where("consultation_fee <= ?", maxF)
		}

		// Load Balancing: Formula weights rating, admin boost, and daily load
		query = query.Order("((average_rating * 10) + admin_boost_score - total_consultations_today) DESC").Limit(1)

		if err := query.First(&profile).Error; err == nil {
			return &profile
		}
		return nil
	}

	// Attempt Matching with Cascading Fallbacks
	matchedProfile := findDoctor(highestMatch, language, genderPref, minRating, maxFee)

	if matchedProfile == nil && genderPref != "" {
		matchedProfile = findDoctor(highestMatch, language, "", minRating, maxFee)
	}
	if matchedProfile == nil && language != "" {
		matchedProfile = findDoctor(highestMatch, "", "", minRating, maxFee)
	}
	if matchedProfile == nil && minRating > 0 {
		matchedProfile = findDoctor(highestMatch, "", "", 0, maxFee)
	}
	// CASE 2: Conflicting/No specialist match -> General Medicine
	if matchedProfile == nil && highestMatch != "General Medicine" {
		matchedProfile = findDoctor("General Medicine", "", "", 0, maxFee)
	}

	var matchedUser *identity.User
	if matchedProfile != nil {
		var u identity.User
		if err := database.DB.First(&u, matchedProfile.UserID).Error; err == nil {
			u.PasswordHash = ""
			matchedUser = &u
		}
	}
	return matchedProfile, matchedUser
}

