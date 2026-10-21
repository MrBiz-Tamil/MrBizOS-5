// Package devi implements the Devi Virtual Health Assistant engine.
package devi

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"drayya-core/internal/domain/identity"
	"drayya-core/internal/resources"
	"drayya-core/internal/services/embedding"
)

// Service defines the interface for the Devi virtual health assistant.
type Service interface {
	Process(ctx context.Context, uc *UserContext, message string) FlowResult
	ProcessWhatsApp(ctx context.Context, phone, sessionID, message string) string
	ProcessLegacy(ctx context.Context, userID, sessionID, message string) string
	GetSymptomHistory(userID uint, limit int) ([]SymptomLog, error)
	GetFamilyMemberHistory(userID, familyMemberID uint, limit int) ([]SymptomLog, error)
	FindSimilarSymptoms(userID uint, query string, limit int) ([]SymptomLog, error)
	DeleteSymptomHistory(userID uint, familyMemberID uint) error
	AnalyzeSymptomsLLM(ctx context.Context, symptoms, duration, severity string) (*SymptomAnalysis, error)
	ApproveLearnedIntent(ctx context.Context, id uint) error
	SetServiceContainer(svc ServiceContainer)
}

// Engine is a type alias for service to maintain backward compatibility.
type Engine = service

type service struct {
	svc     ServiceContainer
	syms    *symptomDataset
	symRepo *Repository
}

// NewEngine constructs a Service (as Engine) from raw DB and Redis pointers.
// Backward-compatible signature — used by server/app.go.
func NewEngine(db *gorm.DB, rdb *redis.Client) Service {
	return NewService(ServiceContainer{DB: db, Redis: rdb})
}

// NewService constructs a Service with a fully-populated ServiceContainer.
func NewService(svc ServiceContainer) Service {
	s := &service{
		svc:  svc,
		syms: loadSymptoms(),
	}
	if svc.DB != nil {
		s.symRepo = NewRepository(svc.DB)
	}
	return s
}

// SetServiceContainer injects (or replaces) the service container after construction.
func (s *service) SetServiceContainer(svc ServiceContainer) {
	s.svc = svc
	if svc.DB != nil && s.symRepo == nil {
		s.symRepo = NewRepository(svc.DB)
	}
}

// ─── Session state helpers ────────────────────────────────────────────────────

func (s *service) loadState(ctx context.Context, uc *UserContext) *sessionState {
	return redisLoadState(ctx, s.svc.Redis, uc.SessionKey)
}

func (s *service) saveState(ctx context.Context, uc *UserContext, state *sessionState) {
	redisSaveState(ctx, s.svc.Redis, uc.SessionKey, state, sessionTTL(uc))
}

func (s *service) clearState(ctx context.Context, uc *UserContext) {
	redisClearState(ctx, s.svc.Redis, uc.SessionKey)
}

// ─── Entry points ─────────────────────────────────────────────────────────────

func (s *service) Process(ctx context.Context, uc *UserContext, message string) FlowResult {
	msg := strings.TrimSpace(message)
	lower := strings.ToLower(msg)

	// Global escape hatch — always clears state and shows menu
	if isExitCommand(lower) {
		s.clearState(ctx, uc)
		// Also clear LLM history for clean restart
		if s.svc.Redis != nil {
			s.svc.Redis.Del(ctx, uc.HistoryKey)
		}
		return s.greetingStart(uc)
	}

	state := s.loadState(ctx, uc)

	// Fresh guest session: clear any stale LLM history
	if uc.IsGuest() && state.Intent == "" {
		if s.svc.Redis != nil {
			s.svc.Redis.Del(ctx, uc.HistoryKey)
		}
	}

	// Active flow — continue it
	if state.Intent != "" {
		result := s.continueFlow(ctx, uc, state, msg)
		if result.Done {
			s.clearState(ctx, uc)
		} else {
			s.saveState(ctx, uc, state)
		}
		return result
	}

	// No active flow: detect intent using the Vector Router
	it := s.detectIntentVector(ctx, lower)

	// LLM fallback: trigger when intent is unknown, OR when the message is
	// long/conversational (> 50 chars) and not an emergency.
	apiKey := os.Getenv("GEMINI_API_KEY")
	shouldUseLLM := apiKey != "" && (it == intentUnknown ||
		(len(msg) > 50 && it != intentEmergency && it != intentBook && it != intentPharmacy))
	if shouldUseLLM {
		// Attach UserContext so LLM tool handlers can access user identity
		llmCtx := contextWithUC(ctx, uc)
		return done(s.processWithLLM(llmCtx, uc, msg))
	}

	return s.startFlow(ctx, uc, it, msg, state)
}

func (s *service) ProcessWhatsApp(ctx context.Context, phone, sessionID, message string) string {
	var uc *UserContext
	if s.svc.IdentityRepo != nil {
		user, err := s.svc.IdentityRepo.GetUserByPhone(phone)
		if err == nil && user != nil {
			uc = BuildUserContext(&user.ID, nil, "")
			uc.DisplayName = user.FullName
		}
	}
	if uc == nil {
		uc = BuildUserContext(nil, nil, "wa-"+phone)
		uc.DisplayName = "Guest (" + phone + ")"
	}
	return s.Process(ctx, uc, message).Message
}

func (s *service) ProcessLegacy(ctx context.Context, userID, sessionID, message string) string {
	var uid *uint
	if n, err := strconv.ParseUint(userID, 10, 64); err == nil && n > 0 {
		u := uint(n)
		uid = &u
	}
	uc := BuildUserContext(uid, nil, sessionID)
	return s.Process(ctx, uc, message).Message
}

// ─── Flow Router ──────────────────────────────────────────────────────────────

func (s *service) startFlow(ctx context.Context, uc *UserContext, it intent, msg string, state *sessionState) FlowResult {
	// Guest capability gate
	if uc.IsGuest() {
		if gateMsg, blocked := guestGate(it); blocked {
			return done(gateMsg)
		}
	}

	var result FlowResult
	switch it {
	case intentGreeting:
		result = s.greetingStart(uc)
	case intentRegister:
		result = s.registerStart()
	case intentEmergency:
		result = s.emergencyStart(ctx, uc)
	case intentSymptom:
		result = s.symptomStart(ctx, uc, msg, state)
	case intentBook:
		result = s.bookingStart(ctx, uc, msg, state)
	case intentPharmacy:
		result = s.pharmacyStart(ctx, uc, msg, state)
	case intentLab:
		result = s.labStart(ctx, uc, msg, state)
	case intentSupport:
		result = s.supportStart(ctx, uc, msg, state)
	case intentInsurance:
		result = s.insuranceStart(ctx, uc, msg, state)
	case intentPremium:
		result = s.premiumStart(uc)
	case intentMedicineInfo:
		result = s.medicineInfoStart(ctx, uc, msg, state)
	case intentPayment:
		result = s.paymentStart(ctx, uc, msg, state)
	case intentMyAccount:
		result = s.myAccountStart(ctx, uc)
	default:
		result = s.unknownStart(msg)
	}

	if !result.Done {
		s.saveState(ctx, uc, state)
	}
	return result
}

func (s *service) continueFlow(ctx context.Context, uc *UserContext, state *sessionState, msg string) FlowResult {
	switch intent(state.Intent) {
	case intentBook:
		return s.bookingContinue(ctx, uc, state, msg)
	case intentSymptom:
		return s.symptomContinue(ctx, uc, state, msg)
	case intentPharmacy:
		return s.pharmacyContinue(ctx, uc, state, msg)
	case intentLab:
		return s.labContinue(ctx, uc, state, msg)
	case intentSupport:
		return s.supportContinue(ctx, uc, state, msg)
	case intentMedicineInfo:
		return s.medicineInfoContinue(ctx, uc, state, msg)
	case intentPayment:
		return s.paymentContinue(ctx, uc, state, msg)
	case intentInsurance:
		return s.insuranceContinue(ctx, uc, state, msg)
	default:
		s.clearState(ctx, uc)
		return s.greetingStart(uc)
	}
}

// ─── Symptom matching engine ──────────────────────────────────────────────────

type symptomEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Departments []string `json:"departments"`
	Urgency     int      `json:"urgency"`
	Aliases     []string `json:"aliases"`
}

type symptomDataset struct {
	Symptoms []symptomEntry `json:"symptoms"`
}

type matchResult struct {
	department string
	urgency    int
}

func loadSymptoms() *symptomDataset {
	var ds symptomDataset
	if err := json.Unmarshal(resources.SymptomMappingJSON, &ds); err != nil {
		return &symptomDataset{}
	}
	return &ds
}

func (s *service) matchSymptoms(text string) matchResult {
	lower := strings.ToLower(text)
	deptScores := make(map[string]int)
	maxUrgency := 0

	for _, sym := range s.syms.Symptoms {
		matched := strings.Contains(lower, strings.ToLower(sym.Name))
		if !matched {
			for _, alias := range sym.Aliases {
				if strings.Contains(lower, strings.ToLower(alias)) {
					matched = true
					break
				}
			}
		}
		if matched {
			for _, d := range sym.Departments {
				deptScores[d]++
			}
			if sym.Urgency > maxUrgency {
				maxUrgency = sym.Urgency
			}
		}
	}

	best, bestScore := "General Medicine", 0
	for dept, score := range deptScores {
		if score > bestScore {
			bestScore = score
			best = dept
		}
	}
	return matchResult{department: best, urgency: maxUrgency}
}

// ─── Doctor matching ──────────────────────────────────────────────────────────

func (s *service) findAvailableDoctor(specialization, userID string) *identity.DoctorProfile {
	if s.svc.DB == nil {
		return nil
	}
	threshold := time.Now().Add(-5 * time.Minute)
	var profile identity.DoctorProfile

	// Try exact specialization first
	err := s.svc.DB.Model(&identity.DoctorProfile{}).
		Where("is_online = ? AND is_verified = ? AND current_status = ? AND last_ping > ?",
			true, true, "available", threshold).
		Where("specialization = ?", specialization).
		Order("((average_rating * 10) + admin_boost_score - total_consultations_today) DESC").
		Limit(1).First(&profile).Error
	if err == nil {
		return &profile
	}

	// Fall back to any available doctor
	err = s.svc.DB.Model(&identity.DoctorProfile{}).
		Where("is_online = ? AND is_verified = ? AND current_status = ? AND last_ping > ?",
			true, true, "available", threshold).
		Order("average_rating DESC").
		Limit(1).First(&profile).Error
	if err != nil {
		return nil
	}
	return &profile
}

// ─── Symptom history (public API for HTTP handlers) ───────────────────────────

func (s *service) GetSymptomHistory(userID uint, limit int) ([]SymptomLog, error) {
	if s.symRepo == nil {
		return nil, nil
	}
	return s.symRepo.GetUserHistory(userID, limit)
}

func (s *service) GetFamilyMemberHistory(userID, familyMemberID uint, limit int) ([]SymptomLog, error) {
	if s.symRepo == nil {
		return nil, nil
	}
	return s.symRepo.GetFamilyMemberHistory(userID, familyMemberID, limit)
}

func (s *service) FindSimilarSymptoms(userID uint, query string, limit int) ([]SymptomLog, error) {
	if s.symRepo == nil {
		return nil, nil
	}
	vec, _ := embedding.Generate(query)
	return s.symRepo.FindSimilar(userID, vec, limit)
}

func (s *service) DeleteSymptomHistory(userID uint, familyMemberID uint) error {
	if s.symRepo == nil {
		return nil
	}
	if familyMemberID > 0 {
		return s.symRepo.DeleteFamilyMemberHistory(userID, familyMemberID)
	}
	return s.symRepo.DeleteUserHistory(userID)
}

// ─── Symptom log persistence (used by flow_symptom.go) ───────────────────────

func (s *service) saveSymptomLog(ctx context.Context, uc *UserContext, sessionID, symptoms string, res matchResult) {
	if s.symRepo == nil {
		return
	}

	vec, err := embedding.Generate(symptoms)
	if err != nil {
		log.Printf("[Devi] Embedding failed: %v", err)
	}

	entry := &SymptomLog{
		SessionID:       sessionID,
		SymptomsText:    symptoms,
		DeptMatched:     res.department,
		Urgency:         res.urgency,
		Embedding:       embedding.MarshalVec(vec),
		ParticipantType: uc.ParticipantType,
	}

	switch uc.ParticipantType {
	case ParticipantPrimary:
		if uc.UserID != nil {
			entry.UserID = *uc.UserID
		}
	case ParticipantFamilyMember:
		if uc.UserID != nil {
			entry.UserID = *uc.UserID
		}
		entry.FamilyMemberID = uc.FamilyMemberID
	case ParticipantGuest:
		if uc.GuestBrowserID != "" {
			entry.GuestBrowserID = &uc.GuestBrowserID
		}
	}

	if err := s.symRepo.Save(entry); err != nil {
		return
	}

	// Retention: keep ≤ 100 rows per authenticated user
	if s.svc.DB != nil && uc.UserID != nil && *uc.UserID > 0 {
		userID := *uc.UserID
		go func() {
			const maxLogs = 100
			sub := s.svc.DB.Model(&SymptomLog{}).Select("id").
				Where("user_id = ?", userID).Order("created_at DESC").Limit(maxLogs)
			s.svc.DB.Where("user_id = ? AND id NOT IN (?)", userID, sub).Delete(&SymptomLog{})
		}()
	}
}
