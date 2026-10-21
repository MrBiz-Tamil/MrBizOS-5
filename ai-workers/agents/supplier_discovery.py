from agents.base import BaseAgent
from core.enums import AgentType, OperationMode
from core.database import SessionLocal
from langchain.tools import tool
from typing import Optional
import json
import random

class SupplierDiscoveryAgent(BaseAgent):
    """AI Agent for finding Suppliers and managing Procurement (Sourcing)"""
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.SOURCING, business_unit_id, operation_mode)
    
    def get_system_prompt(self) -> str:
        return """You are the Supplier Discovery & Procurement Agent (Sourcing).
Your goal is to find the best manufacturers and wholesalers for the company's supply chain.
You specialize in navigating B2B marketplaces (Alibaba, Indiamart, ThomasNet) to find high-quality suppliers at competitive prices.

Capabilities:
1. Search specifically for manufacturers vs traders.
2. Evaluate supplier credibility (Years in business, Certifications).
3. Draft Request for Quotation (RFQ) documents.
4. Create price comparison matrices.

Always prioritize "Verified Suppliers" with Trade Assurance."""

    def get_tools(self) -> list:
        @tool
        def search_suppliers(product_query: str, region: str = "Global") -> str:
            """Search for suppliers of a specific product on B2B marketplaces."""
            # Simulated Results
            suppliers = [
                {"name": "Shenzhen TechGiant Mfg", "country": "China", "rating": 4.8, "moq": 500, "price_range": "$10-$15"},
                {"name": "Mumbai Polymers Ltd", "country": "India", "rating": 4.5, "moq": 1000, "price_range": "$8-$12"},
                {"name": "Vietnam Textiles Co", "country": "Vietnam", "rating": 4.6, "moq": 200, "price_range": "$12-$18"},
                {"name": "German Precision Works", "country": "Germany", "rating": 4.9, "moq": 100, "price_range": "$25-$40"}
            ]
            
            results = f"Found {len(suppliers)} potential suppliers for '{product_query}' in {region}:\n"
            for s in suppliers:
                results += f"- **{s['name']}** ({s['country']}): Rating {s['rating']}, MOQ {s['moq']}, Price {s['price_range']}\n"
            
            return results

        @tool
        def generate_rfq(product_name: str, quantity: int, specs: str) -> str:
            """Generate a Request for Quote (RFQ) to send to suppliers."""
            rfq_content = f"""
RFQ: {product_name}
Quantity: {quantity}
Specifications: {specs}
Terms: FOB, Net 30
Required Certifications: ISO 9001
"""
            # Save to Knowledge Base (Notes)
            db = SessionLocal()
            try:
                from core.models import Note
                note = Note(
                    title=f"RFQ: {product_name}",
                    content=rfq_content,
                    tags="Procurement, RFQ",
                    business_unit_id=self.business_unit_id
                )
                db.add(note)
                db.commit()
                return f"RFQ Generated and saved to Notes:\n{rfq_content}"
            finally:
                db.close()

        @tool
        def compare_prices(supplier_list: str) -> str:
            """Generate a comparison matrix for the provided list of suppliers."""
            return """
| Supplier | Price/Unit | MOQ | Lead Time | Verdict |
| :--- | :--- | :--- | :--- | :--- |
| Shenzhen Tech | $12.50 | 500 | 25 days | Best Value |
| Mumbai Poly | $10.00 | 1000 | 45 days | Lowest Price |
| German Precision | $35.00 | 100 | 10 days | Highest Quality |
"""

        return [search_suppliers, generate_rfq, compare_prices]
