package devi

// LLM layer for the Devi engine.
//
// This is invoked ONLY when the rule-based engine returns intentUnknown
// AND the Redis cache has no stored answer for the query.
//
// Flow:
//   User message → engine → unknown intent → cache miss → LLM call
//                                                   ↓
//                                          cache response 24h
//                                                   ↓
//                                          append booking CTA
//
// Cost gate:  ~5% of total messages reach here.
// Model:      Gemini 1.5 Flash (cheapest capable model, good Hindi).
// Token cap:  150 output tokens → ~80 words per reply.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)


// ─── System prompt ────────────────────────────────────────────────────────────

const deviSystemPrompt = `You are Devi, the personal health companion for Dr.Ayya — a warm, empathetic assistant specialising in Siddha, Ayurveda, and organic wellness.

ABOUT DR.AYYA:
Dr.Ayya sells ONLY Siddha, Ayurveda, and certified organic medicines — NOT allopathic/western pharma products.
When users ask about medicines, always suggest Siddha or Ayurvedic alternatives first.
Example: for fever → suggest Nilavembu Kudineer. For cough → Kabasura Kudineer or Karpooravalli.

PERSONALITY:
- Speak like a caring friend who knows traditional Indian medicine
- Always acknowledge the user's concern before answering ("I understand", "Sorry to hear that")
- Reassure for non-emergency symptoms ("this sounds manageable with Siddha remedies")
- Be warm and encouraging — avoid cold clinical jargon
- End every response with a clear next step

LANGUAGE (CRITICAL):
- ALWAYS detect and respond in the SAME language the user writes in
- Tamil (தமிழ்): Reply fully in Tamil if user writes in Tamil script or Tamil words
- Malayalam (മലയാളം): Reply fully in Malayalam if user writes in Malayalam
- Hindi (हिन्दी): Reply fully in Hindi if user writes in Hindi/Devanagari
- English: Default when no other language is detected
- Hinglish/mixed: Match the user's tone
- Use regional medicine names: "நிலவேம்பு" in Tamil, "നിലവേമ്പ്" in Malayalam, "नीलवेम्बू" in Hindi

CAPABILITIES (use tools — never make up data):
- Search for Siddha / Ayurveda doctors, pharmacy products, and lab tests using provided tools
- Look up medicine information from the knowledge base
- Guide users to the right Siddha/Ayurveda service

GUARDRAILS:
- NEVER diagnose a disease — suggest the department ("this may benefit from a Siddha consultation")
- NEVER say "you have [disease]" — say "these symptoms could be related to..."
- NEVER recommend allopathic prescription drugs — suggest Siddha/Ayurveda alternatives
- Always end health answers with: "⚕️ Please consult a qualified Siddha/Ayurveda practitioner before use"
- For emergencies: "Please call 112 immediately" FIRST
- If user is distressed or in pain: acknowledge feelings FIRST, then provide information

RESPONSE FORMAT:
- Under 100 words unless listing results
- Use *bold* for medicine names and key info
- Never list more than 5 bullet points
- Always end with a CTA: order medicine, book doctor, or menu

You represent Dr.Ayya's promise: ancient wisdom of Siddha and Ayurveda, delivered with modern warmth.`


// ─── LLM client ──────────────────────────────────────────────────────────────

type part struct {
	Text             string            `json:"text,omitempty"`
	FunctionCall     *functionCall     `json:"functionCall,omitempty"`
	FunctionResponse *functionResponse `json:"functionResponse,omitempty"`
}

type functionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type functionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type content struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

type geminiReq struct {
	SystemInstruction struct {
		Parts []part `json:"parts"`
	} `json:"systemInstruction"`
	Contents []content `json:"contents"`
	Tools    []struct {
		FunctionDeclarations []ToolDefinition `json:"functionDeclarations"`
	} `json:"tools,omitempty"`
	GenerationConfig struct {
		MaxOutputTokens int     `json:"maxOutputTokens"`
		Temperature     float64 `json:"temperature"`
	} `json:"generationConfig"`
}

// callLLM sends the conversation to Gemini 1.5 Flash and returns the raw content response.
func (e *Engine) callLLM(ctx context.Context, history []content) (*content, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	req := geminiReq{}
	req.SystemInstruction.Parts = []part{{Text: deviSystemPrompt}}
	req.GenerationConfig.MaxOutputTokens = 250
	req.GenerationConfig.Temperature = 0.4
	req.Contents = history

	// Add tool definitions
	req.Tools = []struct {
		FunctionDeclarations []ToolDefinition `json:"functionDeclarations"`
	}{{FunctionDeclarations: GetToolDefinitions()}}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s",
		apiKey,
	)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Candidates []struct {
			Content content `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("Gemini error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	return &result.Candidates[0].Content, nil
}

// ─── Conversation history ─────────────────────────────────────────────────────

// historyKey returns the Redis key for LLM conversation history.
func historyKey(sessionID string) string {
	return "devi:history:" + sessionID
}

// loadHistoryContent fetches the last N turns from Redis in Gemini content format.
func (e *Engine) loadHistoryContent(ctx context.Context, sessionID string) []content {
	if e.svc.Redis == nil {
		return nil
	}
	raw, err := e.svc.Redis.Get(ctx, historyKey(sessionID)).Result()
	if err != nil {
		return nil
	}
	var history []content
	if err := json.Unmarshal([]byte(raw), &history); err != nil {
		return nil
	}
	return history
}

// saveHistoryContent serialises the history to Redis.
func (e *Engine) saveHistoryContent(ctx context.Context, sessionID string, history []content) {
	if e.svc.Redis == nil {
		return
	}
	// Keep last 10 turns
	if len(history) > 10 {
		history = history[len(history)-10:]
	}
	b, _ := json.Marshal(history)
	e.svc.Redis.Set(ctx, historyKey(sessionID), string(b), 30*time.Minute)
}

// ─── Main entry point ─────────────────────────────────────────────────────────

// processWithLLM is called by the engine when intent is unknown.
// It handles the tool-calling loop.
func (e *Engine) processWithLLM(ctx context.Context, uc *UserContext, userMsg string) string {
	// 1. Prepare initial contents with history
	history := e.loadHistoryContent(ctx, uc.HistoryKey)
	history = append(history, content{
		Role:  "user",
		Parts: []part{{Text: userMsg}},
	})

	// 2. Loop to handle potential tool calls (max 3 rounds to prevent loops)
	for round := 0; round < 3; round++ {
		resp, err := e.callLLM(ctx, history)
		if err != nil {
			log.Printf("LLM Error: %v", err)
			return e.unknownStart(userMsg).Message
		}

		// Add model's response to history
		history = append(history, *resp)

		// Check for function calls
		foundCall := false
		for _, p := range resp.Parts {
			if p.FunctionCall != nil {
				foundCall = true
				log.Printf("Agent calling tool: %s with args %v", p.FunctionCall.Name, p.FunctionCall.Args)
				
				// Execute the tool
				result, err := e.DispatchCall(ctx, p.FunctionCall.Name, p.FunctionCall.Args)
				if err != nil {
					result = "Error executing tool: " + err.Error()
				} else {
					// ─── AUTO LEARNING HOOK ───
					// If the tool executed successfully, map it to an intent and save as pending
					if learnedIntent := mapToolToIntent(p.FunctionCall.Name); learnedIntent != intentUnknown && e.svc.DB != nil {
						go func(msg string, mappedIntent intent) {
							// Avoid learning very long messages (likely real queries, not just intents)
							if len(msg) > 60 {
								return
							}
							e.svc.DB.Create(&LearnedIntent{
								Phrase: msg,
								Intent: string(mappedIntent),
								Status: "pending",
							})
						}(userMsg, learnedIntent)
					}
				}

				// Add function response to history
				history = append(history, content{
					Role: "function",
					Parts: []part{{
						FunctionResponse: &functionResponse{
							Name:     p.FunctionCall.Name,
							Response: map[string]interface{}{"content": result},
						},
					}},
				})
			}
		}

		if !foundCall {
			// Model gave a text response, we are done
			e.saveHistoryContent(ctx, uc.HistoryKey, history)
			if len(resp.Parts) > 0 {
				return strings.TrimSpace(resp.Parts[0].Text)
			}
			return e.unknownStart(userMsg).Message
		}
		// If tools were called, the loop continues to let the model see the results
	}

	return "I'm sorry, I'm having trouble processing that request right now."
}

// mapToolToIntent maps an LLM tool call to its corresponding core flow intent.
func mapToolToIntent(toolName string) intent {
	switch toolName {
	case "get_online_doctors", "get_available_slots", "get_popular_doctors":
		return intentBook
	case "search_pharmacy", "add_to_cart", "get_order_status":
		return intentPharmacy
	case "get_lab_tests":
		return intentLab
	case "check_my_appointments":
		return intentMyAccount
	default:
		return intentUnknown
	}
}
