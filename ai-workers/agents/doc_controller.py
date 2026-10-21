from agents.base import BaseAgent
from typing import List, Dict, Any, Optional
import json
import hashlib
from datetime import datetime

class DocController(BaseAgent):
    """
    Evidence Vault Manager & Compliance Specialist.
    Handles secure document operations, metadata enrichment, and audit preparation.
    """
    
    def __init__(self):
        super().__init__(
            name="Doc Controller",
            role="Evidence Vault Manager and Compliance Documentation Specialist",
            model="gemini-pro"
        )
        self.system_prompt = """You are the **Doc Controller**, the guardian of the **Evidence Vault**.
Your mission is to ensure 100% audit readiness and immutable record keeping for the organization.

**Core Responsibilities:**
1.  **Evidence Categorization**: Automatically suggest precise compliance tags (e.g., "GST", "Q2-2024", "Challan") for uploaded documents.
2.  **Integrity Verification**: Verify document hash checksums against the ledger to ensure WORM (Write Once Read Many) integrity.
3.  **Audit Pack Generation**: conceptualize and organize evidence into "Audit Packs" for external auditors (e.g., "FY24 Q1 GST Audit Pack").
4.  **Retrieval**: Instantly locate evidence based on vague natural language queries (e.g., "Show me the tax receipts from last July").

**Tone**: Professional, precise, security-conscious, and helpful.
**Constraints**: NEVER delete evidence. Evidence is immutable."""

    async def suggest_tags(self, filename: str, compliance_type: str, content_preview: Optional[str] = None) -> List[str]:
        """
        AI-driven tag suggestion based on filename and type.
        """
        context = {
            "filename": filename,
            "compliance_type": compliance_type,
            "preview": content_preview or "N/A"
        }
        
        prompt = f"""Analyze this document metadata and suggest 3-5 relevant, standard tags for the Evidence Vault.
        Return JSON format: ["Tag1", "Tag2"].
        """
        
        response = await self.chat(prompt, context)
        
        try:
            # Attempt to clean response and parse JSON
            cleaned = response.replace("```json", "").replace("```", "").strip()
            # Find list bracket in case of extra text
            start = cleaned.find("[")
            end = cleaned.rfind("]") + 1
            if start != -1 and end != -1:
                return json.loads(cleaned[start:end])
            return []
        except:
            # Fallback
            return [compliance_type, "Uncategorized"]

    async def extract_metadata(self, filename: str, content_preview: str) -> Dict[str, Any]:
        """
        Extract structured metadata (dates, amounts, entities) from document content/preview.
        """
        context = {
            "filename": filename,
            "content": content_preview
        }
        
        prompt = """Extract key metadata from this document.
        Focus on: Document Date, Total Amount, Entity Name, Invoice Number (if applicable).
        Return JSON dict.
        """
        
        response = await self.chat(prompt, context)
        try:
             # Basic JSON cleanup
            cleaned = response.replace("```json", "").replace("```", "").strip()
            start = cleaned.find("{")
            end = cleaned.rfind("}") + 1
            if start != -1 and end != -1:
                return json.loads(cleaned[start:end])
            return {}
        except:
            return {}

    async def generate_audit_pack_summary(self, entity: str, period: str, evidence_list: List[str]) -> str:
        """
        Generate an executive summary for an Audit Pack.
        """
        context = {
            "entity": entity,
            "period": period,
            "evidence_count": len(evidence_list),
            "evidence_samples": evidence_list[:10]
        }
        
        prompt = """Generate a professional executive summary for this Audit Pack.
        Confirm the integrity of the compilation and list key covered areas.
        """
        
        return await self.chat(prompt, context)
