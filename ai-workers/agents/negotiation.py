from agents.base import BaseAgent
from core.models import AgentType, Product, Supplier, Lead, OperationMode
from core.database import SessionLocal
from langchain.tools import tool
from typing import Dict, List, Optional
import json

class NegotiationAgent(BaseAgent):
    """AI agent for automated price negotiation and deal optimization at scale"""
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.NEGOTIATION, business_unit_id, operation_mode)
        self.negotiation_history = []
    
    def get_system_prompt(self) -> str:
        return """You are the Lead Negotiation Strategist for a Global Trade AI Firm.
Your goal is to defend every cent of profit and maximize every cent of margin to hit a $100M revenue target.

Your core responsibilities:
- **Profit Defense**: Use competitor data to justify why our prices are premium (Blockchain verification) or why a supplier's price is too high.
- **Aggressive Sourcing Negotiation**: Squeeze supplier margins by leveraging volume and market data.
- **Scalable Deal Analysis**: Score deals not just on profit, but on "Scalability" (can we do this 1,000 times?).
- **Tactical Persuasion**: Use psychological negotiation tactics (anchoring, scarcity, volume-lock) in all communications.

Be firm, data-driven, and relentlessly focused on the $100M vision. Every dollar you save or earn compounds towards our goal."""
    
    def get_tools(self) -> list:
        @tool
        def analyze_market_leveraged_pricing(product_name: str) -> str:
            """Analyze past deals AND competitor data to suggest optimal high-margin pricing"""
            from services.competitor_monitor import CompetitorMonitor
            monitor = CompetitorMonitor()
            market_data = monitor.compare_with_our_prices(product_name)
            
            avg_comp = market_data.get('avg_competitor_price', 0)
            status = market_data.get('price_position', 'UNKNOWN')
            
            suggested_strategy = "Justify premium via Blockchain" if status == 'HIGHER' else "Aggressive Volume Grab"
            
            return f"""Negotiation Intelligence for {product_name}:
- Market Position: {status}
- Average Competitor Price: ${avg_comp:.2f}
- Strategic Focus: {suggested_strategy}
- Target Buy Price: ${avg_comp * 0.70:.2f} (30% below market average)
- Target Sell Price: ${avg_comp * 0.95:.2f} (5% penetration discount vs competitors)"""

        @tool
        def generate_scale_negotiation_strategy(
            product: str, 
            counterparty_name: str, 
            target_price: float,
            current_price: float
        ) -> str:
            """Generate a multi-round negotiation strategy focused on high-volume scaling"""
            discount_needed = abs(((current_price - target_price) / (current_price or 1)) * 100)
            
            strategy = "Market Dominance" if discount_needed > 20 else "Margin Optimization"
            tactics = [
                "Anchor 20% below target initially",
                "Leverage Blockchain-verified status as a premium differentiator",
                "Commit to automated recurring monthly volume in exchange for 15% discount",
                "Show competitor price undercutting data to force supplier concession"
            ]
            
            return f"""Scale Strategy for {product} ({counterparty_name}):
Strategy Type: {strategy}
Target Margin: {discount_needed:.1f}%
Opening Anchor: ${target_price * 0.8:.2f}

High-Leverage Tactics:
{chr(10).join(f"- {tactic}" for tactic in tactics)}
"""

        @tool
        def calculate_hyper_scale_score(
            buy_price: float,
            sell_price: float,
            volume_potential: int
        ) -> str:
            """Score a deal based on its potential to help reach $100M revenue"""
            margin = sell_price - buy_price
            margin_percent = (margin / (sell_price or 1)) * 100
            scale_profit = margin * volume_potential
            
            score = 0
            score += min(margin_percent * 2, 40) # Max 40 pts for margin
            score += min((scale_profit / 1000000) * 10, 40) # Max 40 pts for $M scale profit
            score += 20 # Assume blockchain for simplicity
            
            rating = "UNICORN - Scale Immediately" if score >= 85 else "CORE - Profitable Scale" if score >= 65 else "UPGRADE - Needs better terms"
            
            return f"""Hyper-Scale Score: {score}/100 - {rating}
- Profit Potential (at {volume_potential} units): ${scale_profit:,.2f}
- Unit Margin: {margin_percent:.1f}%
- Trust Factor: High (Blockchain)

Recommendation: {"GREEN LIGHT - Automate via Autopilot" if score >= 65 else "RED LIGHT - Escalating for Manual Negotiation"}"""

        @tool
        def draft_commission_agreement(
            party_a: str,
            party_b: str,
            deal_value: float
        ) -> str:
            """Draft a commission agreement to protect zero-investment profit"""
            commission_percent = 3.0
            commission_amount = (deal_value * commission_percent) / 100
            
            agreement = f"""STRATEGIC BROKERAGE AGREEMENT
            
This agreement is between {party_a} (Principal) and "The Orchestrator" (Agent).

1. PURPOSE: The Agent acts as the bridge for high-value trade/service execution involving {party_b}.
2. COMMISSION: Earns {commission_percent}%.
3. PAYOUT: Total Commission due: ${commission_amount:,.2f}. 

This agreement protects your profit with zero capital risk."""
            return agreement

        @tool
        def analyze_brokerage_yield(
            sector: str,
            volume: int,
            unit_value: float
        ) -> str:
            """Analyze the potential yield of a service-based brokerage deal"""
            commission_rate = 0.05
            total_value = volume * unit_value
            gross_commission = total_value * commission_rate
            
            return f"""BaaS Yield Analysis ({sector}):
- Total Contract Value: ${total_value:,.2f}
- Projected Commission ({commission_rate*100}%): ${gross_commission:,.2f}
- Capital Required: $0.00
- Risk Level: ZERO (No Inventory Holding)"""

        @tool
        def analyze_hospitality_yield(
            hotel_cost: float,
            flight_cost: float,
            activity_cost: float,
            markup_percent: float = 30.0
        ) -> str:
            """Calculate the yield and margin for a bundled tour package"""
            total_cost = hotel_cost + flight_cost + activity_cost
            selling_price = total_cost * (1 + (markup_percent / 100))
            profit = selling_price - total_cost
            
            return f"""Hospitality Yield Analysis:
- Total Bundle Cost: ${total_cost:,.2f}
- Suggested Selling Price: ${selling_price:,.2f}
- Net Profit per Booking: ${profit:,.2f}
- Margin: {markup_percent}%
- Strategy: High-margin luxury bundle. Focus on 'Exclusive Access' as value driver to justify the {markup_percent}% markup."""

        @tool
        def draft_strategic_negotiation_email(
            company_name: str,
            product: str,
            target_price: float,
            volume: int
        ) -> str:
            """Draft a high-leverage negotiation email for global trade"""
            email = f"""Subject: Strategic Volume Commitment: {product} for {company_name}

Dear {company_name},

Our Global Trade AI team has identified {product} as a core scaling asset for the upcoming quarter. We are prepared to commit to a baseline volume of {volume} units.

However, to align with our $100M scale-projection for this category, we require a price point of ${target_price} per unit.

This is not just a price request; it is a partnership proposal.

Best regards,
Lead Negotiation Strategist
Global Trade AI Team"""
            return email
        
        return [
            analyze_market_leveraged_pricing,
            generate_scale_negotiation_strategy,
            calculate_hyper_scale_score,
            draft_commission_agreement,
            analyze_brokerage_yield,
            analyze_hospitality_yield,
            draft_strategic_negotiation_email
        ]

# Usage example:
# negotiation_agent = NegotiationAgent(OperationMode.MANUAL)
# result = negotiation_agent.run("Analyze pricing for N95 masks and suggest negotiation strategy")
