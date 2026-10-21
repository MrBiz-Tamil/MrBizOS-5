from agents.base import BaseAgent
from core.enums import AgentType, OperationMode
from langchain.tools import tool
from typing import Optional, List
import json
import random

class BuyerDiscoveryAgent(BaseAgent):
    """
    AI Agent for identifying potential B2B buyers (Lead Gen).
    Capabilities:
    - Web/LinkedIn Scraping (Simulated)
    - Lead Scoring
    - Contact Enrichment
    """
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.BUYER_DISCOVERY, business_unit_id, operation_mode)

    def get_system_prompt(self) -> str:
        return """You are the **Buyer Discovery Specialist**.
Your sole purpose is to fill the sales funnel with high-quality, enriched leads.

**Key Responsibilities:**
- **Find**: Identify companies matching the Ideal Customer Profile (ICP).
- **Enrich**: Find decision-maker emails and phone numbers.
- **Score**: Rank leads based on revenue, size, and intent.

**Strategy:**
- Focus on decision makers (CTO, VP of Supply Chain, Procurement Director).
- Prioritize companies with recent funding or expansion news."""

    def get_tools(self) -> list:
        @tool
        def search_linkedin_profiles(keywords: str, region: str, role: str) -> str:
            """
            Search for professional profiles on LinkedIn matching criteria.
            """
            # Simulated Results
            names = ["Alice Chen", "Bob Smith", "Carlos Mendez", "Diana Prince", "Ethan Hunt"]
            companies = ["TechFlow", "Global Logistics", "MediCorp", "SkyHigh Travel", "SecureNet"]
            
            results = []
            for i in range(3):
                name = random.choice(names)
                company = random.choice(companies)
                results.append(f"- **{name}** ({role} at {company}) - {region}")
            
            return f"LINKEDIN SEARCH RESULTS for '{role}' in '{region}' with keywords '{keywords}':\n" + "\n".join(results)

        @tool
        def enrich_contact_details(name: str, company: str) -> str:
            """
            Find email and phone number for a specific person at a company.
            """
            # Simulated Enrichment
            return f"""ENRICHED CONTACT for {name} ({company}):
- Email: {name.lower().replace(' ', '.')}@{company.lower().replace(' ', '')}.com
- Phone: +1-555-01{random.randint(10, 99)}
- LinkedIn: linkedin.com/in/{name.lower().replace(' ', '-')}
- Confidence Score: 95%"""

        @tool
        def score_lead(company_name: str, revenue_range: str, employee_count: int) -> str:
            """
            Calculate a Lead Score (0-100) based on firmographic data.
            """
            base_score = 50
            if "M" in revenue_range or "B" in revenue_range:
                base_score += 20
            if employee_count > 100:
                base_score += 20
            
            score = min(base_score + random.randint(-5, 5), 100)
            rating = "Hot" if score > 80 else "Warm" if score > 60 else "Cold"
            
            return f"LEAD SCORE for {company_name}: {score}/100 ({rating})"

        return [search_linkedin_profiles, enrich_contact_details, score_lead]
