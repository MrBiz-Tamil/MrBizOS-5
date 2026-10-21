package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/queue"
	"drayya-core/internal/utils/config"
	"drayya-core/internal/presence"


	"github.com/redis/go-redis/v9"
)

const maxConcurrentSessions = 3

// ProcessStreamItem handles detailed dispatch logic from Redis Stream
// ProcessStreamItem returns true if the item was successfully processed (or should be ACKed),
// false if dispatch failed and the item should be retried or dead-lettered.
func ProcessStreamItem(ctx context.Context, msg redis.XMessage) bool {
	val, ok := msg.Values["meta"].(string)
	if !ok {
		log.Printf("[Dispatch] Invalid meta format in stream msg %s — skipping", msg.ID)
		return true // bad message → ACK to prevent infinite retry
	}

	var meta queue.AppointmentMetadata
	if err := json.Unmarshal([]byte(val), &meta); err != nil {
		log.Printf("[Dispatch] JSON error in stream msg %s: %v — skipping", msg.ID, err)
		return true // unparseable → ACK
	}

	log.Printf("[Dispatch] Appointment %d (Type: %s)", meta.AppointmentID, meta.AppointmentType)
	return assignAppointment(ctx, meta.AppointmentID, meta)
}

// ProcessQueue pulls one item from the ZSET queue and attempts to assign it.
// Returns 1 if processed, 0 if queue empty or error.
func ProcessQueue() int {
	ctx := context.Background()
	meta, err := queue.Dequeue(ctx)
	if err != nil {
		log.Printf("Dequeue Error: %v", err)
		return 0
	}

	if meta == nil {
		return 0
	}

	log.Printf("[Manual] Manual/Polling Dispatch: Appointment %d", meta.AppointmentID)
	success := assignAppointment(ctx, meta.AppointmentID, *meta)
	if success {
		return 1
	}
	return 0
}

func assignAppointment(ctx context.Context, appointmentID int, meta queue.AppointmentMetadata) bool {
	var appt appointment.Appointment

	if err := database.DB.WithContext(ctx).First(&appt, appointmentID).Error; err != nil {
		log.Printf("Appointment %d not found (DB Error: %v)", appointmentID, err)
		return true // skip to ack
	}

	if appt.DoctorID != nil {
		log.Printf("Appointment %d already assigned", appointmentID)
		return true
	}

	// AI Autopilot: Find Best Doctor
	bestDoctorID := findBestDoctor(ctx, &appt, meta)

	if bestDoctorID == 0 {
		log.Printf("[Warning] No suitable doctor found for Appointment %d. Escalating...", appointmentID)
		queue.Escalate(ctx, appointmentID, "No suitable doctor found (Autopilot)")
		notifyChiefs(ctx, fmt.Sprintf("[Warning] Escalated Appointment #%d: No doctors available", appointmentID), map[string]interface{}{"appointment_id": appointmentID, "reason": "no_doctors"})
		return false
	}

	// Assign
	appt.DoctorID = &bestDoctorID
	appt.Status = appointment.StatusConfirmed

	if err := database.DB.WithContext(ctx).Save(&appt).Error; err != nil {
		log.Printf("Error updating appointment: %v", err)
		return false
	}

	log.Printf("[Success] Assigned Appointment %d to Doctor %d (Autopilot)", appointmentID, bestDoctorID)

	// Update real-time session load
	presence.IncrementSession(ctx, fmt.Sprintf("%d", bestDoctorID)) // ws helper still exists?
	// Check if IncrementSession uses ws.Rdb?
	// If yes, I should move it to database or ws package should use database.Rdb.
	// ws.IncrementSession depends on Rdb. I should fix it in ws/presence.go too.


	// Notify Doctor
	notifyDoctor(ctx, bestDoctorID, appointmentID)

	// Notify Chiefs (Audit Trail)
	notifyChiefs(ctx, fmt.Sprintf("[Success] Auto-Assigned Appointment #%d to Dr. %d", appointmentID, bestDoctorID), map[string]interface{}{
		"appointment_id": appointmentID,
		"doctor_id":      bestDoctorID,
		"type":           "auto_assign",
	})

	return true
}

// calculateDistance returns distance between two points in km using Haversine
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km

	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	dlon := lon2Rad - lon1Rad
	dlat := lat2Rad - lat1Rad

	a := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Pow(math.Sin(dlon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func findBestDoctor(ctx context.Context, appt *appointment.Appointment, meta queue.AppointmentMetadata) uint {
	// 1. Fetch Candidates (Online & Verified)
	var doctors []identity.DoctorProfile
	if err := database.DB.WithContext(ctx).
		Where("is_online = ? AND is_verified = ?", true, true).
		Find(&doctors).Error; err != nil {
		return 0
	}

	// 1.1 Filter by Redis Presence (Real-time source of truth)
	var onlineCandidates []identity.DoctorProfile
	for _, doc := range doctors {
		if presence.IsDoctorOnline(ctx, fmt.Sprintf("%d", doc.UserID)) {
			onlineCandidates = append(onlineCandidates, doc)
		}
	}


	// Fallback: If no doctors are found in Redis but DB says online,
	// we use DB results as a backup (lower priority)
	if len(onlineCandidates) > 0 {
		doctors = onlineCandidates
	}

	if len(doctors) == 0 {
		return 0
	}

	// Use patient coordinates stored on the appointment (set by SOS trigger or Devi emergency handler)
	var patientLat, patientLng float64 = appt.Latitude, appt.Longitude

	// 2. Score Candidates
	type ScoredDoctor struct {
		Doctor identity.DoctorProfile
		Score  float64
	}

	var ranked []ScoredDoctor

	for _, doc := range doctors {
		// Hard session cap: never assign to an overloaded doctor.
		if presence.GetActiveSessions(ctx, fmt.Sprintf("%d", doc.UserID)) >= maxConcurrentSessions {
			continue
		}


		// GEO-FENCING LOGIC FOR EMERGENCIES:
		if meta.AppointmentType == "emergency" {
			// Require doctors within 5km radius
			dist := calculateDistance(patientLat, patientLng, doc.Latitude, doc.Longitude)
			if dist > 5.0 {
				continue // Skip doctors outside the 5km SOS radius
			}
		}

		score := calculateDoctorScore(ctx, doc, meta)
		ranked = append(ranked, ScoredDoctor{Doctor: doc, Score: score})
	}

	// 3. Sort (Highest Score First)
	// Simple bubble sort for small list, or use sort.Slice
	for i := 0; i < len(ranked); i++ {
		for j := 0; j < len(ranked)-i-1; j++ {
			if ranked[j].Score < ranked[j+1].Score {
				ranked[j], ranked[j+1] = ranked[j+1], ranked[j]
			}
		}
	}

	// 4. Return Best
	if len(ranked) > 0 {
		log.Printf("[Match] Best Match: Dr %d (Score: %.2f)", ranked[0].Doctor.UserID, ranked[0].Score)
		return ranked[0].Doctor.UserID
	}

	return 0
}

func calculateDoctorScore(ctx context.Context, doc identity.DoctorProfile, meta queue.AppointmentMetadata) float64 {
	// Formula: (Performance * 0.5) + (TierBonus) - (Load * 20)

	score := 0.0

	// 1. Base Performance (Use Rating as proxy for now if PerfService missing)
	// Valid range 0-100. Rating 0-5 -> 5*20 = 100.
	// 1. Base Performance (Use Rating as proxy for now if PerfService missing)
	// Valid range 0-100. Rating 0-5 -> 5*20 = 100.
	perfScore := doc.AverageRating * 20.0
	if perfScore == 0 {
		perfScore = 70.0 // Default neutral
	}
	score += perfScore * 0.5

	// 1b. Growth Boost (Smart Growth Engine)
	if config.GetGrowthSetting("enableAlgorithmBoost", true).(bool) {
		dailyLimit := int64(config.GetGrowthSetting("dailyFreeLimit", float64(2)).(float64))
		if doc.FreeConsultsGivenToday < int(dailyLimit) {
			score *= 1.10 // 10% Visibility Boost
		}
	}

	// 2. Load Factor (Active Sessions from Redis)
	load := presence.GetActiveSessions(ctx, fmt.Sprintf("%d", doc.UserID))
	score -= float64(load) * 20.0


	// 3. Tier Bonus (Matching Premium Patients to Top Docs)
	if meta.SubscriptionTier == "platinum" || meta.SubscriptionTier == "diamond" {
		if doc.AverageRating > 4.5 {
			score += 30.0
		}
	}

	// 4. Language Match — a shared language materially improves consultation
	// quality (especially for AYUSH). Strong boost when the doctor speaks the
	// patient's preferred language.
	if meta.Language != "" && doc.LanguagesSpoken != "" {
		if strings.Contains(strings.ToLower(doc.LanguagesSpoken), strings.ToLower(meta.Language)) {
			score += 40.0
		}
	}

	return score
}

func notifyDoctor(ctx context.Context, doctorID uint, appointmentID int) {
	// Fetch appointment to get patient name and consultation mode
	var appt appointment.Appointment
	if err := database.DB.WithContext(ctx).First(&appt, appointmentID).Error; err != nil {
		log.Printf("[Dispatch] notifyDoctor: could not fetch appointment %d: %v", appointmentID, err)
		return
	}

	// Fetch the patient's display name from the users table
	// (Appointment.PatientID == identity.User.ID)
	patientName := "Patient"
	var patientUser identity.User
	if err := database.DB.WithContext(ctx).
		Select("full_name").
		First(&patientUser, appt.PatientID).Error; err == nil && patientUser.FullName != "" {
		patientName = patientUser.FullName
	}

	consultationMode := appt.ConsultationMode
	if consultationMode == "" {
		consultationMode = "video"
	}

	apptType := appt.AppointmentType
	if apptType == "" {
		apptType = "instant"
	}

	msg := map[string]interface{}{
		// "instant_call_request" is handled by DoctorDashboard.tsx WebSocket listener.
		// The previous type "system_notification" was silently dropped by the frontend.
		"type":    "instant_call_request",
		"room_id": fmt.Sprintf("room:user:%d", doctorID),
		"metadata": map[string]interface{}{
			"appointment_id":    appointmentID,
			"patient_name":      patientName,
			"consultation_mode": consultationMode,
			"type":              apptType,
		},
	}

	data, _ := json.Marshal(msg)
	if database.Rdb != nil {
		database.Rdb.Publish(ctx, "chat:global", data)
	}

	log.Printf("[Dispatch] Notified doctor %d of instant call request #%d (%s %s)", doctorID, appointmentID, apptType, consultationMode)
}

func notifyChiefs(ctx context.Context, message string, meta map[string]interface{}) {
	msg := map[string]interface{}{
		"type":     "system_notification",
		"room_id":  "room:system:dispatch",
		"content":  message,
		"metadata": meta,
	}
	data, _ := json.Marshal(msg)
	if database.Rdb != nil {
		database.Rdb.Publish(ctx, "chat:global", data)
	}
}

// RunDispatchCycle processes the queue until empty.
// This is used by the manual trigger endpoint.
func RunDispatchCycle() {
	count := 0
	for {
		processed := ProcessQueue()
		if processed == 0 {
			break
		}
		count += processed
	}
	log.Printf("Manual Dispatch Cycle Completed. Processed %d items.", count)
}
