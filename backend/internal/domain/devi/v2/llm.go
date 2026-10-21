package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Gemini wire format. Mirrors the v1beta API. The struct shapes look verbose
// because Gemini nests `tools.functionDeclarations` and roles a couple of
// layers deep, but each piece has a one-line meaning:
//
//	systemInstruction → set once per call, holds Devi's persona prompt
//	contents          → the conversation transcript (one entry per turn)
//	tools             → the tool catalog the LLM may call
//	generationConfig  → temperature + token caps
//
// The response either contains a text part (final reply) OR a functionCall
// part (request to invoke a tool). Mixed parts ARE possible — Gemini can
// return prose + a tool call in one turn. We handle that.

const (
	geminiModel      = "gemini-2.0-flash"
	geminiURLPattern = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
	geminiTimeout    = 20 * time.Second
)

// ─── Request shapes ───────────────────────────────────────────────────────

type geminiPart struct {
	Text             string             `json:"text,omitempty"`
	FunctionCall     *geminiFnCall      `json:"functionCall,omitempty"`
	FunctionResponse *geminiFnResponse  `json:"functionResponse,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"` // "user" | "model" | "function"
	Parts []geminiPart `json:"parts"`
}

type geminiFnCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiFnResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFnDecl `json:"functionDeclarations"`
}

type geminiFnDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type geminiRequest struct {
	SystemInstruction struct {
		Parts []geminiPart `json:"parts"`
	} `json:"systemInstruction"`
	Contents []geminiContent `json:"contents"`
	Tools    []geminiTool    `json:"tools,omitempty"`
	Config   struct {
		MaxOutputTokens int     `json:"maxOutputTokens"`
		Temperature     float64 `json:"temperature"`
	} `json:"generationConfig"`
}

// ─── Response shapes ──────────────────────────────────────────────────────

type geminiResponse struct {
	Candidates []struct {
		Content      geminiContent `json:"content"`
		FinishReason string        `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// LLMTurn is one round of LLM interaction. Either Text is non-empty (final
// reply for this turn) OR ToolCalls is non-empty (LLM wants to call tools),
// or both — Gemini can mix prose with a tool call.
type LLMTurn struct {
	Text         string
	ToolCalls    []ToolCall
	InputTokens  int
	OutputTokens int
}

// ToolCall is a single tool invocation request from the LLM.
type ToolCall struct {
	Name string
	Args map[string]any
}

// ─── Client ───────────────────────────────────────────────────────────────

// LLMClient wraps the Gemini HTTP call. Stateless — safe to reuse across
// goroutines. The agent loop creates one per process; conversation state is
// passed in via Call().
type LLMClient struct {
	httpClient *http.Client
}

func NewLLMClient() *LLMClient {
	return &LLMClient{
		httpClient: &http.Client{Timeout: geminiTimeout},
	}
}

// Call runs one LLM round trip: system prompt + history + available tools →
// next turn. Returns the LLM's response (text or tool calls), or an error.
func (c *LLMClient) Call(ctx context.Context, systemPrompt string, history []Message, tools []Tool) (*LLMTurn, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	req := geminiRequest{}
	req.SystemInstruction.Parts = []geminiPart{{Text: systemPrompt}}
	req.Config.MaxOutputTokens = MaxOutputTokens
	req.Config.Temperature = 0.4
	req.Contents = messagesToContents(history)
	if len(tools) > 0 {
		req.Tools = []geminiTool{{FunctionDeclarations: toolsToDecls(tools)}}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(geminiURLPattern, geminiModel, apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	var out geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("gemini decode failed: %w", err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("gemini error %d: %s", out.Error.Code, out.Error.Message)
	}
	if len(out.Candidates) == 0 {
		return nil, fmt.Errorf("gemini returned no candidates")
	}

	turn := &LLMTurn{
		InputTokens:  out.UsageMetadata.PromptTokenCount,
		OutputTokens: out.UsageMetadata.CandidatesTokenCount,
	}
	for _, p := range out.Candidates[0].Content.Parts {
		if p.Text != "" {
			turn.Text += p.Text
		}
		if p.FunctionCall != nil {
			turn.ToolCalls = append(turn.ToolCalls, ToolCall{
				Name: p.FunctionCall.Name,
				Args: p.FunctionCall.Args,
			})
		}
	}
	return turn, nil
}

// ─── Conversion helpers ───────────────────────────────────────────────────

// messagesToContents converts our Message log to Gemini's wire format. Two
// non-obvious bits:
//
//  - Gemini uses "model" not "assistant" for AI replies.
//  - Tool results go in their own message with role="function".
func messagesToContents(msgs []Message) []geminiContent {
	out := make([]geminiContent, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case RoleUser:
			out = append(out, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: m.Content}},
			})
		case RoleModel:
			out = append(out, geminiContent{
				Role:  "model",
				Parts: []geminiPart{{Text: m.Content}},
			})
		case RoleFunction:
			// Tool result. Gemini wants the result as a JSON object under
			// the function name. We stored arbitrary JSON; wrap it.
			var resp map[string]any
			if len(m.ToolResult) > 0 {
				_ = json.Unmarshal(m.ToolResult, &resp)
			}
			if resp == nil {
				resp = map[string]any{}
			}
			out = append(out, geminiContent{
				Role: "function",
				Parts: []geminiPart{{
					FunctionResponse: &geminiFnResponse{
						Name:     m.ToolName,
						Response: resp,
					},
				}},
			})
		}
	}
	return out
}

// toolsToDecls turns our Tool registry into Gemini's functionDeclaration list.
func toolsToDecls(tools []Tool) []geminiFnDecl {
	out := make([]geminiFnDecl, 0, len(tools))
	for _, t := range tools {
		out = append(out, geminiFnDecl{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		})
	}
	return out
}
