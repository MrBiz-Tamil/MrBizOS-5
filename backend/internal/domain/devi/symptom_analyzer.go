package devi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type SymptomAnalysis struct {
	UrgencyLevel             string   `json:"urgency_level"`
	RecommendedSpecialization string   `json:"recommended_specialization"`
	RecommendedSpecializations []string `json:"recommended_specializations"`
	Confidence               float64  `json:"confidence"`
	AIAssessment             string   `json:"ai_assessment"`
	SeverityScore            int      `json:"severity_score"`
	ConsultationMode         string   `json:"consultation_mode"`
	Action                   string   `json:"action"`
}

const symptomAnalysisPrompt = `You are the Dr.Ayya AI Symptom Analyzer.
Your task is to analyze patient symptoms and provide a structured medical assessment.

Rules:
1. Identify the urgency level: 'low', 'moderate', 'high', or 'critical'.
2. Recommend the best medical specialization (e.g., Cardiology, Dermatology, General Medicine).
3. Provide a brief, empathetic AI assessment (max 3 sentences).
4. Assign a severity score (1-100).
5. Suggest a consultation mode ('video' or 'emergency').
6. Suggest an action ('match_doctor' or 'trigger_sos').
7. ALWAYS return your response in PURE JSON format.

Input:
Symptoms: %s
Duration: %s
Severity: %s

Example Output:
{
  "urgency_level": "moderate",
  "recommended_specialization": "Gastroenterology",
  "recommended_specializations": ["Gastroenterology", "General Medicine"],
  "confidence": 0.9,
  "ai_assessment": "Your symptoms suggest a possible digestive issue. A consultation with a gastroenterologist is recommended for further evaluation.",
  "severity_score": 45,
  "consultation_mode": "video",
  "action": "match_doctor"
}`

func (e *Engine) AnalyzeSymptomsLLM(ctx context.Context, symptoms, duration, severity string) (*SymptomAnalysis, error) {
	prompt := fmt.Sprintf(symptomAnalysisPrompt, symptoms, duration, severity)
	
	history := []content{
		{
			Role:  "user",
			Parts: []part{{Text: prompt}},
		},
	}

	resp, err := e.callLLM(ctx, history)
	if err != nil {
		return nil, err
	}

	if len(resp.Parts) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	rawJSON := resp.Parts[0].Text

	// Strip all markdown code-fence variants the model might emit:
	//   ```json … ```  |  ``` … ```  |  `…`
	rawJSON = strings.TrimSpace(rawJSON)
	for _, fence := range []string{"```json", "```JSON", "```"} {
		if strings.HasPrefix(rawJSON, fence) {
			rawJSON = strings.TrimPrefix(rawJSON, fence)
			break
		}
	}
	if idx := strings.LastIndex(rawJSON, "```"); idx != -1 {
		rawJSON = rawJSON[:idx]
	}
	rawJSON = strings.TrimSpace(rawJSON)

	// Extract the first JSON object in case the model added preamble text
	if start := strings.Index(rawJSON, "{"); start > 0 {
		rawJSON = rawJSON[start:]
	}
	if end := strings.LastIndex(rawJSON, "}"); end != -1 && end < len(rawJSON)-1 {
		rawJSON = rawJSON[:end+1]
	}

	if rawJSON == "" || rawJSON[0] != '{' {
		log.Printf("[SymptomAnalyzer] Response is not JSON: %s", resp.Parts[0].Text)
		return nil, fmt.Errorf("LLM did not return valid JSON")
	}

	var analysis SymptomAnalysis
	if err := json.Unmarshal([]byte(rawJSON), &analysis); err != nil {
		log.Printf("[SymptomAnalyzer] Failed to parse LLM response: %v\nRaw: %s", err, rawJSON)
		return nil, err
	}

	// Wire trigger_sos action: if the LLM decides this is critical, surface it.
	// The caller (flow_symptom.go) checks analysis.Action == "trigger_sos".
	if analysis.UrgencyLevel == "critical" && analysis.Action == "" {
		analysis.Action = "trigger_sos"
	}

	return &analysis, nil
}
