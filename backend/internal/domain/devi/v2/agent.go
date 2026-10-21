package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// Agent is the per-process runtime that ties everything together. Construct
// once at startup; the same Agent serves every request concurrently.
//
// The Run() method is the only entry point. It implements the standard
// tool-using agent loop:
//
//   1. Safety check (rules)            — short-circuit if matched
//   2. Rate limit check                — reject if over quota
//   3. Append user message to memory
//   4. Loop up to MaxToolCallsPerTurn:
//        a. Call LLM with system prompt + history + tool schema
//        b. If reply is text only → append, return to user
//        c. If reply contains tool calls → execute each, append results
//   5. Persist final assistant turn to memory
//   6. Return ChatResponse
//
// The loop has a hard cap to prevent runaway agents.
type Agent struct {
	registry *Registry
	llm      *LLMClient
	memory   *Memory
	limiter  *Limiter
}

func NewAgent(registry *Registry, llm *LLMClient, memory *Memory, limiter *Limiter) *Agent {
	return &Agent{
		registry: registry,
		llm:      llm,
		memory:   memory,
		limiter:  limiter,
	}
}

// Run executes one user turn. Returns a ChatResponse ready to serialize.
func (a *Agent) Run(ctx context.Context, persona Persona, sessionID, userMessage string) ChatResponse {
	start := time.Now()

	// Make session ID available to tools via context (some need to mutate
	// session state, e.g. switch_family_context, confirmation pending store).
	ctx = WithSessionID(ctx, sessionID)

	// 1. Safety rails — rule-based, sub-millisecond.
	if action := SafetyCheck(userMessage, persona); action != nil {
		// Don't waste an LLM call on this. Memory is intentionally NOT
		// updated for safety-triggered responses; the conversation should
		// reset cleanly after a safety event.
		if action.Done {
			a.memory.Clear(ctx, persona, sessionID)
		}
		return ChatResponse{
			Reply:     action.Reply,
			SessionID: sessionID,
			Done:      action.Done,
			LatencyMS: int(time.Since(start).Milliseconds()),
		}
	}

	// 2a. Rate limits.
	if v := a.limiter.Check(ctx, persona); !v.Allowed {
		return ChatResponse{
			Reply:     friendlyLimitMessage(v.Reason, persona),
			SessionID: sessionID,
			LimitedBy: "rate",
			LatencyMS: int(time.Since(start).Milliseconds()),
		}
	}

	// 2b. Cost circuit breaker (Phase 8). Skip the LLM entirely when the
	// daily/per-user cap is spent — protects against runaway scrapers and
	// caps blast radius if our cost monitoring missed something.
	if CheckCircuit(ctx, persona) == CircuitOpen {
		return ChatResponse{
			Reply:     "I'm temporarily limited while traffic catches up. Try again in a few minutes — for urgent help, please use the SOS button.",
			SessionID: sessionID,
			LimitedBy: "cost",
			LatencyMS: int(time.Since(start).Milliseconds()),
		}
	}

	// 3. Confirmation gate — if a destructive action is pending and the
	// user's message looks like a YES/NO, settle it directly without the
	// LLM. The result is fed into memory as a function result, then we
	// fall through to the LLM loop so it can deliver the closing reply.
	if pending := loadPending(ctx, persona, sessionID); pending != nil {
		intent := DetectConfirm(userMessage)
		switch intent {
		case ConfirmIntentYes:
			toolsCalledForConfirm := []string{pending.ToolName}
			result := a.runDeferredTool(ctx, persona, pending)
			clearPending(ctx, persona, sessionID)
			a.memory.Append(ctx, persona, sessionID, Message{
				Role:        RoleUser,
				Content:     userMessage,
				TimestampMS: time.Now().UnixMilli(),
			})
			resJSON, _ := json.Marshal(result)
			a.memory.Append(ctx, persona, sessionID, Message{
				Role:        RoleFunction,
				ToolName:    pending.ToolName,
				ToolResult:  resJSON,
				TimestampMS: time.Now().UnixMilli(),
			})
			// Fall through into the LLM loop with toolsCalled seeded so
			// the final response includes the confirm action.
			_ = toolsCalledForConfirm
		case ConfirmIntentNo:
			clearPending(ctx, persona, sessionID)
			return ChatResponse{
				Reply:     "Cancelled. Anything else I can help with?",
				SessionID: sessionID,
				LatencyMS: int(time.Since(start).Milliseconds()),
			}
		}
	}

	// 4. Append user turn (skipped above if confirmation handled it).
	if last := a.lastUserMessage(ctx, persona, sessionID); last != userMessage {
		a.memory.Append(ctx, persona, sessionID, Message{
			Role:        RoleUser,
			Content:     userMessage,
			TimestampMS: time.Now().UnixMilli(),
		})
	}

	// 5. Tool-using loop.
	systemPrompt := BuildSystemPrompt(persona)
	tools := a.registry.ForPersona(persona)
	toolsCalled := []string{}
	maxIters := MaxToolCallsPerTurn + 1 // +1 for the final text turn

	for iter := 0; iter < maxIters; iter++ {
		history := a.memory.Load(ctx, persona, sessionID)
		callStart := time.Now()
		turn, err := a.llm.Call(ctx, systemPrompt, history, tools)
		callLatency := int(time.Since(callStart).Milliseconds())
		if err != nil {
			log.Printf("[devi v2] LLM call failed: %v", err)
			RecordUsage(ctx, persona, sessionID, geminiModel, 0, 0, callLatency, true)
			return ChatResponse{
				Reply:     "I'm having trouble thinking right now — please try again in a moment.",
				SessionID: sessionID,
				LatencyMS: int(time.Since(start).Milliseconds()),
			}
		}
		RecordUsage(ctx, persona, sessionID, geminiModel,
			turn.InputTokens, turn.OutputTokens, callLatency, false)

		// No tool calls — this is the final reply.
		if len(turn.ToolCalls) == 0 {
			a.memory.Append(ctx, persona, sessionID, Message{
				Role:        RoleModel,
				Content:     turn.Text,
				TimestampMS: time.Now().UnixMilli(),
			})
			latency := int(time.Since(start).Milliseconds())

			// Persist a SymptomLog row so admin stats include V2 traffic.
			// inferIntent is a cheap classification used only for the log;
			// the LLM does the real routing via tools.
			intent, urgency := inferIntent(userMessage, toolsCalled)
			writeSymptomLog(ctx, persona, sessionID, userMessage, intent, urgency, latency)

			return ChatResponse{
				Reply:       turn.Text,
				SessionID:   sessionID,
				ToolsCalled: toolsCalled,
				LatencyMS:   latency,
			}
		}

		// Mixed (prose + tools) — emit the prose, then execute tools.
		// Gemini sometimes prefaces a tool call with explanation. We keep
		// the model's text in memory so the next turn has context.
		if turn.Text != "" {
			a.memory.Append(ctx, persona, sessionID, Message{
				Role:        RoleModel,
				Content:     turn.Text,
				TimestampMS: time.Now().UnixMilli(),
			})
		}

		// Execute every requested tool. If any fails, we still feed an
		// error result back to the LLM so it can recover gracefully.
		for _, call := range turn.ToolCalls {
			toolsCalled = append(toolsCalled, call.Name)
			result := a.executeTool(ctx, persona, call)
			resJSON, _ := json.Marshal(result)
			a.memory.Append(ctx, persona, sessionID, Message{
				Role:        RoleFunction,
				ToolName:    call.Name,
				ToolResult:  resJSON,
				TimestampMS: time.Now().UnixMilli(),
			})
		}
		// Loop — LLM will see the tool results and either reply or call more tools.
	}

	// Hit the iteration cap — the LLM kept asking for tools without
	// settling. Return what we have.
	return ChatResponse{
		Reply: "I gathered some information but need to step back — could you rephrase what " +
			"you're looking for?",
		SessionID:   sessionID,
		ToolsCalled: toolsCalled,
		LatencyMS:   int(time.Since(start).Milliseconds()),
	}
}

// executeTool runs one tool with full guard sequence:
//   - Tool exists?
//   - Persona allowed?
//   - Confirmation required? (Phase 4; not enforced here yet)
//   - Tool's own counters within cap (Phase 4)
//
// Returns a JSON-serialisable result for the LLM to consume.
func (a *Agent) executeTool(ctx context.Context, persona Persona, call ToolCall) any {
	tool, ok := a.registry.Get(call.Name)
	if !ok {
		return map[string]any{"error": fmt.Sprintf("unknown tool: %s", call.Name)}
	}
	if !tool.AllowedFor(persona.Role) {
		return map[string]any{"error": "permission denied"}
	}

	// Confirmation gate (Phase 4) — destructive tools cannot fire on first
	// request. Stash the proposed action in Redis, return a structured
	// "needs confirmation" envelope so the LLM asks the user to confirm.
	if tool.Confirm {
		summary := tool.Name + " — confirm before executing"
		savePending(ctx, persona, SessionIDFromContext(ctx), PendingAction{
			ToolName: call.Name,
			Args:     call.Args,
			Summary:  summary,
		})
		return map[string]any{
			"_confirm_required": true,
			"tool":              call.Name,
			"args":              call.Args,
			"message":           "This action will be performed once the user explicitly confirms.",
		}
	}

	// Inject persona + session ID so tools can scope by user_id and mutate
	// session state without the LLM ever seeing those identifiers.
	toolCtx := WithPersona(ctx, persona)
	toolCtx = WithSessionID(toolCtx, SessionIDFromContext(ctx))

	result, err := tool.Execute(toolCtx, call.Args)
	if err != nil {
		log.Printf("[devi v2] tool %s failed: %v", call.Name, err)
		return map[string]any{"error": err.Error()}
	}
	return result
}

// runDeferredTool actually executes a previously-pending tool after the user
// confirmed. It bypasses the Confirm gate (since we're past it) by calling
// Execute directly with persona+session injected via context.
func (a *Agent) runDeferredTool(ctx context.Context, persona Persona, p *PendingAction) any {
	tool, ok := a.registry.Get(p.ToolName)
	if !ok {
		return map[string]any{"error": "tool no longer registered"}
	}
	if !tool.AllowedFor(persona.Role) {
		return map[string]any{"error": "permission denied"}
	}
	toolCtx := WithPersona(ctx, persona)
	toolCtx = WithSessionID(toolCtx, SessionIDFromContext(ctx))
	result, err := tool.Execute(toolCtx, p.Args)
	if err != nil {
		log.Printf("[devi v2] deferred tool %s failed: %v", p.ToolName, err)
		return map[string]any{"error": err.Error()}
	}
	return result
}

// lastUserMessage returns the content of the most recent user message in
// memory, or empty string. Used to avoid duplicate appends when the
// confirmation gate already wrote the user turn.
func (a *Agent) lastUserMessage(ctx context.Context, persona Persona, sessionID string) string {
	hist := a.memory.Load(ctx, persona, sessionID)
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Role == RoleUser {
			return strings.TrimSpace(hist[i].Content)
		}
	}
	return ""
}

// inferIntent maps the user's message and the tools the LLM ended up
// calling into a coarse intent label for the symptom log. This is purely
// for analytics — the actual routing is done by Gemini via the tool
// schema. We back-derive the intent from the tools the LLM chose.
func inferIntent(_ string, toolsCalled []string) (intent string, urgency int) {
	intent = "general"
	for _, t := range toolsCalled {
		switch t {
		case "search_doctors", "get_doctor_availability", "create_booking":
			intent = "book_appointment"
		case "search_products", "add_to_cart":
			intent = "pharmacy"
		case "get_my_prescriptions":
			intent = "prescription_info"
		case "find_similar_symptoms":
			intent = "symptom_check"
		case "lookup_knowledge":
			intent = "knowledge_question"
		case "update_patient_profile":
			intent = "profile_update"
		}
	}
	// Urgency is set by safety rails (emergency = 5). For non-safety
	// turns it's 1 (low) — symptom_check turns may later be re-stamped
	// by an LLM-based severity classifier in Phase 3.
	urgency = 1
	return
}

// friendlyLimitMessage returns the user-facing message when a rate limit
// fires. Each window has a tailored tone so the user knows what happened
// and what they can do.
func friendlyLimitMessage(window string, persona Persona) string {
	switch window {
	case "minute":
		return "You're going fast! Give me 30 seconds and try again."
	case "hour":
		if persona.Role == RoleGuest {
			return "You've hit the guest hourly limit. Sign up for a free account to chat more."
		}
		return "You've reached this hour's chat limit. Try again in a little while!"
	case "day":
		if persona.Tier != "premium" {
			return "You've used today's free chat quota. Resets at midnight, or upgrade to premium for unlimited."
		}
		return "Daily limit reached. Resets at midnight."
	}
	return "Slow down a little and try again."
}
