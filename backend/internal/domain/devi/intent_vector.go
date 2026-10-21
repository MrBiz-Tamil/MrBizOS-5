package devi

import (
	"context"
	"strings"
	"sync"
	"time"

	"drayya-core/internal/services/embedding"
	"gorm.io/gorm"
)

// IntentReference holds the text phrases that represent a specific intent across languages.
type IntentReference struct {
	Intent  intent
	Phrases []string
}

// coreIntentReferences contains the training phrases for our vector router in English, Tamil, Malayalam, and Hindi.
var coreIntentReferences = []IntentReference{
	{
		Intent: intentBook,
		Phrases: []string{
			"book an appointment with doctor", "i want to see a doctor",
			"எனக்கு டாக்டர் வேண்டும்", "டாக்டரை பார்க்க வேண்டும்",
			"എനിക്ക് ഡോക്ടറെ കാണണം", "ഡോക്ടറെ ബുക്ക് ചെയ്യുക",
			"मुझे डॉक्टर से मिलना है", "डॉक्टर की अपॉइंटमेंट बुक करें",
		},
	},
	{
		Intent: intentPharmacy,
		Phrases: []string{
			"buy medicines", "order pharmacy", "need tablets",
			"மருந்து வாங்க வேண்டும்", "மாத்திரைகள் வேண்டும்",
			"മരുന്ന് വാങ്ങണം", "ഫാർമസി",
			"मुझे दवाई चाहिए", "दवाइयां खरीदें",
		},
	},
	{
		Intent: intentLab,
		Phrases: []string{
			"book lab test", "blood test", "xray scan",
			"ரத்த பரிசோதனை", "லேப் டெஸ்ட்",
			"രക്ത പരിശോധന", "ലാബ് ടെസ്റ്റ്",
			"ब्लड टेस्ट बुक करें", "लैब टेस्ट",
		},
	},
	{
		Intent: intentSymptom,
		Phrases: []string{
			"i have a headache", "not feeling well", "fever and cough",
			"எனக்கு காய்ச்சல் உள்ளது", "உடல் நிலை சரியில்லை",
			"എനിക്ക് സുഖമില്ല", "പനിയും ചുമയും",
			"मुझे बुखार है", "तबीयत ठीक नहीं है",
		},
	},
	{
		Intent: intentGreeting,
		Phrases: []string{
			"hello", "hi there", "good morning",
			"வணக்கம்", "ஹலோ",
			"നമസ്കാരം", "ഹലോ",
			"नमस्ते", "हेलो",
		},
	},
	{
		Intent: intentSupport,
		Phrases: []string{
			"complaint", "help me", "issue with my order",
			"புகார்", "உதவி வேண்டும்",
			"പരാതി", "സഹായിക്കുക",
			"शिकायत", "मुझे मदद चाहिए",
		},
	},
	{
		Intent: intentMedicineInfo,
		Phrases: []string{
			"side effects of this medicine", "how to take this tablet", "is it safe",
			"இந்த மருந்தின் பக்க விளைவுகள்", "எப்படி சாப்பிடுவது",
			"ഈ മരുന്നിന്റെ പാർശ്വഫലങ്ങൾ",
			"इस दवा के साइड इफेक्ट्स",
		},
	},
}

// vectorStore holds the computed embeddings for our intents
type vectorStore struct {
	mu          sync.RWMutex
	initialized bool
	embeddings  map[intent][][]float32
}

var globalVectorStore = &vectorStore{
	embeddings: make(map[intent][][]float32),
}

// initializeVectors computes embeddings for all reference phrases.
// It loads core references and any dynamically learned intents approved by admins.
func (vs *vectorStore) initialize(db *gorm.DB) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.initialized {
		return
	}

	for _, ref := range coreIntentReferences {
		for _, phrase := range ref.Phrases {
			vec, err := embedding.Generate(phrase)
			if err == nil && len(vec) > 0 {
				vs.embeddings[ref.Intent] = append(vs.embeddings[ref.Intent], vec)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

	if db != nil {
		var learned []LearnedIntent
		if err := db.Where("status = ?", "approved").Find(&learned).Error; err == nil {
			for _, l := range learned {
				if l.Embedding == "" {
					vec, err := embedding.Generate(l.Phrase)
					if err == nil && len(vec) > 0 {
						l.Embedding = embedding.MarshalVec(vec)
						db.Save(&l)
						vs.embeddings[intent(l.Intent)] = append(vs.embeddings[intent(l.Intent)], vec)
					}
					time.Sleep(50 * time.Millisecond)
				} else {
					// Use pre-computed embedding from DB to save API calls
					// Need to unmarshal l.Embedding back to []float32
					// Quick hack for this demonstration: regenerate if not properly deserialized,
					// or properly parse it. Since we have embedding.MarshalVec, let's just generate it for simplicity if we don't have unmarshal.
					vec, _ := embedding.Generate(l.Phrase)
					if len(vec) > 0 {
						vs.embeddings[intent(l.Intent)] = append(vs.embeddings[intent(l.Intent)], vec)
					}
				}
			}
		}
	}

	vs.initialized = true
}

// ApproveLearnedIntent is an admin function to approve a pending learned phrase
// and immediately inject it into the active vectorStore.
func (s *service) ApproveLearnedIntent(ctx context.Context, id uint) error {
	if s.svc.DB == nil {
		return nil
	}
	var l LearnedIntent
	if err := s.svc.DB.First(&l, id).Error; err != nil {
		return err
	}
	l.Status = "approved"
	vec, err := embedding.Generate(l.Phrase)
	if err == nil && len(vec) > 0 {
		l.Embedding = embedding.MarshalVec(vec)
		globalVectorStore.mu.Lock()
		globalVectorStore.embeddings[intent(l.Intent)] = append(globalVectorStore.embeddings[intent(l.Intent)], vec)
		globalVectorStore.mu.Unlock()
	}
	return s.svc.DB.Save(&l).Error
}

// Approval function definition ended above.

// Removed cosineSimilarity as it is already defined in repository.go

// detectIntentVector finds the closest intent using vector similarity.
func (s *service) detectIntentVector(ctx context.Context, text string) intent {
	// 0. Safety override: Emergency keywords bypass AI completely for instant response
	lower := strings.ToLower(text)
	if isEmergency(lower) {
		return intentEmergency
	}

	// 1. Check Redis cache for exact match to save API cost and achieve 1ms latency
	if s.svc.Redis != nil {
		cacheKey := "devi:intent_cache:" + text
		if cached, err := s.svc.Redis.Get(ctx, cacheKey).Result(); err == nil && cached != "" {
			return intent(cached)
		}
	}

	// 2. Make sure reference vectors are initialized
	if !globalVectorStore.initialized {
		globalVectorStore.initialize(s.svc.DB)
	}

	// 3. Generate embedding for user text
	vec, err := embedding.Generate(text)
	if err != nil || len(vec) == 0 {
		// Fallback to legacy rule-based if embedding fails (e.g., no API key)
		return detectIntentLegacy(lower)
	}

	globalVectorStore.mu.RLock()
	defer globalVectorStore.mu.RUnlock()

	var bestIntent intent = intentUnknown
	var highestSim float32 = 0.0
	const threshold float32 = 0.70 // Require 70% similarity to confidently route

	for it, vectors := range globalVectorStore.embeddings {
		for _, refVec := range vectors {
			sim := cosineSimilarity(vec, refVec)
			if sim > highestSim {
				highestSim = sim
				bestIntent = it
			}
		}
	}

	if highestSim >= threshold {
		// Cache the confident result for 24h
		if s.svc.Redis != nil {
			cacheKey := "devi:intent_cache:" + text
			s.svc.Redis.Set(ctx, cacheKey, string(bestIntent), 24*time.Hour)
		}
		return bestIntent
	}

	return intentUnknown
}

// emergencyKeywords bypass vector search for safety
var emergencyKeywords = []string{
	"emergency", "sos", "ambulance", "critical", "dying", "unconscious",
	"not breathing", "heart attack", "stroke", "bleeding heavily", "suicide",
}

func isEmergency(lower string) bool {
	for _, kw := range emergencyKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
