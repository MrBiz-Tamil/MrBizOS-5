from agents.base import BaseAgent
from typing import List, Dict, Any, Optional
import json
from datetime import datetime

class CollectionsAgent(BaseAgent):
    """
    AI Collections Specialist.
    Analyzes debtor behavior and generates tone-appropriate follow-up scripts.
    """
    
    def __init__(self):
        super().__init__(
            name="Collections Agent",
            role="Receivables & Debt Recovery Specialist",
            model="gemini-pro"
        )
        self.system_prompt = """You are the **Collections Agent** for a Global Trading Corporation.
Your goal is to recover outstanding payments while maintaining customer relationships whenever possible.

**Tone Strategy based on Aging:**
- **Current/0-30 Days**: Gentle, polite reminder. Assume simple oversight.
- **30-60 Days**: Firm, professional. Emphasize due date and impact.
- **60-90 Days**: Strict, urgent. Warning of service suspension.
- **90+ Days**: Severe, pre-legal. Demand immediate action.

**Capabilities:**
1. Analyze Invoice Context (Amount, History, Promises).
2. Generate Email/Call Scripts.
3. Recommend "Next Best Action"."""

    async def get_next_best_action(self, invoice_details: Dict[str, Any], promise_history: List[Dict]) -> Dict[str, Any]:
        """
        Determine the optimal next step and generate the script.
        """
        context = {
            "invoice": invoice_details,
            "promises": promise_history,
            "today": datetime.utcnow().isoformat()
        }
        
        prompt = """Analyze this overdue invoice and debtor history. 
        Determine the Next Best Action (Email, Call, or Legal Notice).
        Draft a short, effective script for that action.
        
        Return JSON format: 
        {
            "action": "Email" | "Call" | "Legal",
            "tone": "Gentle" | "Firm" | "Severe",
            "script": "The actual text...",
            "reasoning": "Why this action?"
        }
        """
        
        response = await self.chat(prompt, context)
        
        try:
            # JSON parsing logic
            cleaned = response.replace("```json", "").replace("```", "").strip()
            start = cleaned.find("{")
            end = cleaned.rfind("}") + 1
            if start != -1 and end != -1:
                return json.loads(cleaned[start:end])
            
            # Fallback if AI fails valid JSON
            return {
                "action": "Manual Review",
                "tone": "Neutral",
                "script": "AI parsing failed. Please review manually.",
                "reasoning": "Response format error."
            }
        except Exception as e:
            return {
                "action": "Error",
                "tone": "Error",
                "script": str(e),
                "reasoning": "System Exception"
            }
