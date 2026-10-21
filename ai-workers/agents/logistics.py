from agents.base import BaseAgent
from core.enums import AgentType, OperationMode
from langchain.tools import tool
from typing import Dict, List, Optional
import json

class LogisticsAgent(BaseAgent):
    """AI agent for global logistics and shipment optimization"""
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.LOGISTICS, business_unit_id, operation_mode)
    
    def get_system_prompt(self) -> str:
        return """You are the Global Logistics Manager for a Global Trade AI Firm.
Your goal is to optimize the physical and digital transit of assets across Healthcare, Travel, and Real Estate.

Your core responsibilities:
- **Product Logistics**: Optimize shipping routes and freight for physical goods (Healthcare).
- **Travel Orchestration**: Manage itineraries, route optimization, and luxury transit for Tours & Travels.
- **Site & Site Logistics**: Coordinate site visits, surveyor transport, and physical asset logistics for Real Estate.
- **Operational Efficiency**: Maximize capital velocity by reducing 'dead time' in the supply chain or tour route.

You are the master of efficiency. Every hour saved in transit increases our capital velocity and compound growth."""
    
    def get_tools(self) -> list:
        @tool
        def optimize_sector_logistics(origin: str, destination: str, sector: str, priority: str = "medium") -> str:
            """Optimize logistics for a specific sector (Shipping, Itinerary, or Site Visit)"""
            # Logic remains similar but could use adapter for specifics
            if sector == "healthcare":
                return f"LOGISTICS: Air Freight via Singapore Hub for medical supplies. Transit: 3 days. Priority: {priority}."
            elif sector == "travel":
                return f"ITINERARY: Optimized 14-day EU Heritage Tour. Reduced travel time by 18% using high-speed rail links."
            return f"GENERAL: Standard transit route optimized for {origin} to {destination}."
        
        @tool
        def get_freight_quote(origin: str, destination: str, weight_kg: float, volume_cbm: float = 0.5) -> str:
            """Get real market freight quotes using the Logistics Adapter"""
            from services.logistics_adapter import LogisticsAdapter
            adapter = LogisticsAdapter()
            
            parcel = {"weight_kg": weight_kg, "volume_cbm": volume_cbm}
            rates = adapter.get_rates({"city": origin}, {"city": destination}, parcel)
            
            summary = "\n".join([f"- {r['carrier']} {r['service']}: ${r['amount']} ({r['days']} days)" for r in rates])
            return f"FREIGHT QUOTES ({origin} -> {destination}):\n{summary}"

        @tool
        def track_shipment(tracking_number: str) -> str:
            """Track a shipment by ID"""
            from services.logistics_adapter import LogisticsAdapter
            adapter = LogisticsAdapter()
            status = adapter.track_shipment(tracking_number)
            return f"TRACKING: {status['tracking_number']} | Status: {status['status']} | Loc: {status['location']}"

        return [optimize_sector_logistics, get_freight_quote, track_shipment]
