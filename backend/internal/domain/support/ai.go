package support

// AI auto-reply and department classifier for support tickets.
//
// Flow (async goroutine, fires after ticket is saved):
//   1. Call Gemini 1.5 Flash with ticket subject + description
//   2. Receive JSON: { department, can_resolve, reply, reason }
//   3. Always save department onto ticket
//   4. If can_resolve:
//      - Save bot TicketMessage (IsAIReply=true)
//      - Set ticket.AIHandled = true, status = "in_progress"
//   5. If not can_resolve:
//      - Send department notification email to the relevant team inbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"drayya-core/internal/services/email"
	"net/http"
	"os"
	"strings"
	"time"
)

const supportSystemPrompt = `You are an intelligent support triage agent for Dr.Ayya / Dr.Ayya — an AYUSH healthcare platform in India.

Your job when receiving a support ticket:
1. Classify the department (pick exactly one): billing | medical | pharmacy | technical | general
2. Decide if you can auto-resolve it with a clear, helpful reply.
3. If you can resolve it, write a friendly reply in plain English (under 120 words). Be warm and specific.
4. If you cannot resolve it (requires account access, transaction verification, clinical decision, or escalation), set can_resolve to false and briefly state why.

Department guide:
- billing    → payment failed, refund, invoice, subscription charge
- medical    → clinical question, prescription, doctor feedback, health record
- pharmacy   → medicine order, delivery, stock, prescription upload
- technical  → app bug, login issue, page not loading, OTP not received
- general    → anything else, feedback, feature request

You MUST respond with ONLY valid JSON — no prose, no markdown fences:
{
  "department": "<billing|medical|pharmacy|technical|general>",
  "can_resolve": <true|false>,
  "reply": "<helpful reply if can_resolve is true, otherwise empty string>",
  "reason": "<why human is needed if can_resolve is false, otherwise empty string>"
}`

// aiClassification is what Gemini returns.
type aiClassification struct {
	Department string `json:"department"`
	CanResolve bool   `json:"can_resolve"`
	Reply      string `json:"reply"`
	Reason     string `json:"reason"`
}

// classifyAndReply calls Gemini and returns the classification result.
// Returns nil on any error (caller ignores gracefully).
func classifyAndReply(subject, description string) *aiClassification {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil
	}

	userMsg := fmt.Sprintf("Subject: %s\n\nDescription: %s", subject, description)

	body, _ := json.Marshal(map[string]interface{}{
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{{"text": supportSystemPrompt}},
		},
		"contents": []map[string]interface{}{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": userMsg}},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 300,
			"temperature":     0.2,
			"responseMimeType": "application/json",
		},
	})

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s",
		apiKey,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[support/ai] request build error: %v", err)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		log.Printf("[support/ai] Gemini call error: %v", err)
		return nil
	}
	defer resp.Body.Close()

	var raw struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct{ Message string `json:"message"` } `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		log.Printf("[support/ai] decode error: %v", err)
		return nil
	}
	if raw.Error != nil {
		log.Printf("[support/ai] Gemini error: %s", raw.Error.Message)
		return nil
	}
	if len(raw.Candidates) == 0 || len(raw.Candidates[0].Content.Parts) == 0 {
		log.Printf("[support/ai] empty response")
		return nil
	}

	text := strings.TrimSpace(raw.Candidates[0].Content.Parts[0].Text)
	// Strip markdown fences if present
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result aiClassification
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		log.Printf("[support/ai] JSON parse error: %v — raw: %s", err, text)
		return nil
	}

	// Ensure department is valid
	if !ValidDepartments[result.Department] {
		result.Department = "general"
	}

	return &result
}

// processTicketAsync runs after a ticket is created.
// It classifies, optionally auto-replies, and notifies the department team.
func processTicketAsync(repo Repository, ticket *Ticket) {
	go func() {
		result := classifyAndReply(ticket.Subject, ticket.Description)
		if result == nil {
			// AI unavailable — just send to general queue, no bot reply
			notifyDept(ticket, "general", "")
			return
		}

		// Always persist department
		dept := result.Department
		if err := repo.SetDepartment(ticket.ID, dept); err != nil {
			log.Printf("[support/ai] SetDepartment error: %v", err)
		}

		if result.CanResolve && result.Reply != "" {
			// Save bot reply
			botMsg := &TicketMessage{
				TicketID:   ticket.ID,
				SenderID:   0,
				SenderName: "Dr.Ayya Support Bot",
				Message:    result.Reply,
				IsStaff:    true,
				IsAIReply:  true,
			}
			if err := repo.CreateMessage(botMsg); err != nil {
				log.Printf("[support/ai] CreateMessage error: %v", err)
			}
			if err := repo.SetAIHandled(ticket.ID, true); err != nil {
				log.Printf("[support/ai] SetAIHandled error: %v", err)
			}
			// Move to in_progress so it appears active
			if err := repo.SetStatus(ticket.ID, "in_progress"); err != nil {
				log.Printf("[support/ai] SetStatus error: %v", err)
			}
		} else {
			// Needs human — notify dept team
			notifyDept(ticket, dept, result.Reason)
		}
	}()
}

// notifyDept emails the department inbox about a new unresolved ticket.
func notifyDept(ticket *Ticket, dept, reason string) {
	envKey := DeptEmail(dept)
	to := os.Getenv(envKey)
	if to == "" {
		to = os.Getenv("SUPPORT_GENERAL_EMAIL")
	}
	if to == "" {
		to = os.Getenv("SMTP_FROM")
	}
	if to == "" {
		return
	}

	body := email.SupportTicketNotificationEmail(
		ticket.ID,
		ticket.Subject,
		ticket.Category,
		ticket.Priority,
		ticket.Description,
		dept,
		reason,
	)

	email.SendSimple(to,
		fmt.Sprintf("[%s] New Ticket #%d — %s", strings.ToUpper(dept), ticket.ID, ticket.Subject),
		body,
	)
}
