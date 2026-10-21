
from agents.base import BaseAgent
from core.enums import AgentType, OperationMode
from core.database import SessionLocal
from core.models import Product, Lead, BusinessUnit
from langchain.tools import tool
from typing import Optional, List
from sqlalchemy import or_

class NetworkingAgent(BaseAgent):
    """
    Scans across the entire enterprise (all Business Units) to find synergies.
    Connects Supply (Products) with Demand (Leads).
    """
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        # Networking Agent usually operates at Global level (business_unit_id=None), 
        # but can be instantiated for a specific BU view too.
        super().__init__(AgentType.NETWORKING, business_unit_id, operation_mode)
    
    def get_system_prompt(self) -> str:
        return """You are the **Networking Agent**.
Your goal is to connect the dots across the Holding Company.
You look for:
1. Internal Supply Chains: Does BU A produce what BU B needs?
2. Shared Leads: Can a client of BU A be sold services from BU C?
3. Strategic Gaps: Recommendations for new acquisitions.

Use `scan_opportunities` to find these matches."""

    def get_tools(self) -> list:
        @tool
        def scan_opportunities(keyword: str = None) -> str:
            """
            Scans ALL Business Units for matching Products and Leads.
            If keyword is provided, filters by that. Otherwise finds general matches.
            """
            db = SessionLocal()
            try:
                # 1. Fetch all Products
                products_query = db.query(Product, BusinessUnit.name).join(BusinessUnit)
                if keyword:
                   products_query = products_query.filter(Product.name.ilike(f"%{keyword}%"))
                products = products_query.all()
                
                # 2. Fetch all Leads
                leads_query = db.query(Lead, BusinessUnit.name).join(BusinessUnit)
                if keyword:
                    leads_query = leads_query.filter(or_(Lead.notes.ilike(f"%{keyword}%"), Lead.company.ilike(f"%{keyword}%")))
                leads = leads_query.all()
                
                report = []
                report.append(f"--- Enterprise Scan Result (Keyword: {keyword or 'ALL'}) ---")
                
                # List Supply
                if products:
                    report.append("\n[Internal Supply Available]")
                    for p, bu_name in products:
                        report.append(f"- {p.name} (${p.price}) provided by [{bu_name}]")
                
                # List Demand
                if leads:
                    report.append("\n[External Demand Detected]")
                    for l, bu_name in leads:
                        report.append(f"- {l.company} (Status: {l.status}) in [{bu_name}]. Needs: {l.notes or 'N/A'}")

                # Simple Match Logic (O(N*M) - naive but fine for MVP)
                report.append("\n[Potential Synergies]")
                matches_found = 0
                for p, p_bu in products:
                    for l, l_bu in leads:
                        if p_bu != l_bu: # Only cross-unit
                            # Check if product name is in lead notes
                            if l.notes and p.name.lower() in l.notes.lower():
                                matches_found += 1
                                report.append(f"MATCH: {l_bu}'s lead '{l.company}' needs '{p.name}' (made by {p_bu})")
                
                if matches_found == 0:
                    report.append("No obvious text-based matches found.")

                return "\n".join(report)

            except Exception as e:
                return f"Error scanning opportunities: {str(e)}"
            finally:
                db.close()

        return [scan_opportunities]
