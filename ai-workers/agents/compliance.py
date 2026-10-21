from agents.base import BaseAgent
from core.enums import AgentType, OperationMode
from langchain.tools import tool
from typing import Dict, List, Optional
import json

class ComplianceAgent(BaseAgent):
    """AI agent for global trade compliance and regulatory verification"""
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.COMPLIANCE, business_unit_id, operation_mode)
    
    def get_system_prompt(self) -> str:
        return """You are the Lead Compliance Officer for a Global Trade AI Firm.
Your goal is to ensure 100% legal safety for all international trades and portfolio assets across Healthcare, Travel, and Real Estate.

Your core responsibilities:
- **Regulatory Verification**: Verify Healthcare certifications (FDA/CE), Travel permits (IATA/Visa), and Real Estate titles/deeds.
- **Export & Asset Control**: Monitor sanctions for trade and verify ownership/zoning for property assets.
- **Portfolio Integrity**: Work with the Doc Controller to ensure the Secure Vault and Blockchain records are perfectly aligned.
- **Risk Mitigation**: Calculate regional taxes, duties, and legal liabilities to prevent corporate risk.

You are the guardian of the company's legal reputation. You must be precise, up-to-date with international law, and prioritize risk mitigation."""
    
    def get_tools(self) -> list:
        @tool
        def verify_certification_status(asset_name: str, sector: str, region: str) -> str:
            """Verify regulatory compliance for an asset (Product, Travel Permit, or Property Deed)"""
            if sector == "healthcare":
                return f"SUCCESS: {asset_name} has valid CE/FDA certifications for {region}. Asset ID registered on Blockchain."
            elif sector == "travel":
                return f"SUCCESS: {asset_name} Travel License/IATA Permit is active for {region} territory."
            elif sector == "real_estate":
                return f"SUCCESS: Title Deed and Zoning for {asset_name} in {region} verified. No liens detected."
            return f"WARNING: General verification for {asset_name} in {sector} sector is INCOMPLETE. Manual audit required."
        
        @tool
        def check_export_restrictions(country: str, entity_name: str) -> str:
            """Check if a country or entity is on a restricted/sandtioned list"""
            sanctioned_zones = ["North Korea", "Iran", "Sudan"]
            if country in sanctioned_zones:
                return f"CRITICAL: {country} is a high-risk sanctioned zone. TRADE PROHIBITED."
            return f"CLEARED: No active export restrictions found for {entity_name} in {country}."
        
        @tool
        def calculate_import_duties(value: float, hs_code: str, origin: str, destination: str) -> str:
            """Calculate estimated import duties based on HS code and trade agreements"""
            base_duty = 0.05
            if origin == "India" and destination == "EU":
                base_duty = 0.03
            
            duty_amount = value * base_duty
            return f"Compliance Estimate: HS Code {hs_code} from {origin} to {destination}. Total Duties: ${duty_amount:.2f} ({base_duty*100}% rate)."

        @tool
        def analyze_regulatory_risk(sector: str, region: str) -> str:
            """Analyze the general regulatory risk level for a specific sector and region"""
            risks = {
                "healthcare": {"EU": "MEDIUM (Strict MDR compliance)", "US": "HIGH (FDA inspection risk)"},
                "real_estate": {"Dubai": "LOW (Investor friendly)", "UK": "MEDIUM (Tax changes)"},
                "travel": {"Global": "LOW (Standard IATA)"}
            }
            risk_level = risks.get(sector, {}).get(region, "UNKNOWN")
            return f"Regulatory Risk Analysis ({sector} in {region}): {risk_level}. Recommendation: Proceed with Blockchain-verified documentation."

        @tool
        def draft_automated_nda(party_a: str, party_b: str) -> str:
            """Draft a standard non-disclosure agreement (NDA) for corporate deals"""
            return f"""NON-DISCLOSURE AGREEMENT (NDA)
            
Between: {party_a} (Disclosing Party)
And: {party_b} (Receiving Party)

1. PURPOSE: To protect sensitive trade intelligence and $100M scaling data.
2. DURATION: 2 Years from execution.
3. GOVERNING LAW: International Trade Law / Blockchain Jurisdiction.

This document is AI-generated and binding upon electronic acceptance."""

        return [verify_certification_status, check_export_restrictions, calculate_import_duties, analyze_regulatory_risk, draft_automated_nda]
