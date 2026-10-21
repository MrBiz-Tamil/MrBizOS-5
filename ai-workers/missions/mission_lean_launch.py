import asyncio
from services.corporate_engine import CorporateEngine
from agents.finance import FinanceAgent
from agents.sourcing import SourcingAgent
from agents.negotiation import NegotiationAgent
from agents.sales import SalesAgent
from core.models import OperationMode

async def simulate_lean_launch_discussion():
    print("🚀 STARTING AI TEAM STRATEGIC DISCUSSION: 'OPERATION BOOTSTRAP'\n")
    
    engine = CorporateEngine()
    finance = FinanceAgent(OperationMode.MANUAL)
    sourcing = SourcingAgent(OperationMode.MANUAL)
    negotiation = NegotiationAgent(OperationMode.MANUAL)
    sales = SalesAgent(OperationMode.MANUAL)

    # 1. Finance Agent: The Credit Play
    print("🏦 FINANCE AGENT: Analyzing capital efficiency...")
    credit_strategy = "Recommendation: Focus on 'Net-30' or 'Net-60' payment terms with suppliers. Leverage Blockchain records to show reliability even with low initial cash. Suggesting a 'Pre-Order' deposit model to cover 50% of COGS before shipping."

    # 2. Sourcing Agent: Finding the 'Entry' Product
    print("🔍 SOURCING AGENT: Searching for high-margin, small-batch goods...")
    product_pick = "Selected: Specialized IR Thermometers (South Korea). Low shipping volume per unit, 50%+ gross margin. Small MOQ (Minimum Order Quantity) of 100 units is acceptable."

    # 3. Negotiation Agent: The Margin Defense
    print("🤝 NEGOTIATION AGENT: Drafting the lean-entry script...")
    negotiation_tactic = "Strategy: 'The Partnership Anchor.' Offer the supplier a long-term 'Authorized Distributor' status on our Blockchain ledger in exchange for a 20% discount on the first 100-unit 'Market Entry' batch."

    # 4. Sales Agent: The Demand Generation
    print("💰 SALES AGENT: Designing low-cost outreach...")
    sales_plan = "Execution: Generate 10 AI Growth Videos targeting Tier-2 private clinics instead of Tier-1 hospitals. Tier-2 buyers make decisions 5x faster and are willing to pay a 15% premium for immediate local stock."

    # 5. Corporate Engine: The CEO Decision
    print("🧠 CORPORATE ENGINE: Synthesizing 'Operation Bootstrap'...")
    final_decision = {
        "Strategic_Goal": "Convert $5,000 into $15,000 in 45 days.",
        "Resource_Allocation": "90% of effort on Sales Pre-Orders, 10% on Sourcing.",
        "Risk_Rating": "LOW (Cap-at-Batch)",
        "CEO_Recommendation": "GREEN LIGHT. This lean model prioritizes cash-flow velocity over absolute volume. Start with South Korean IR Thermometers."
    }

    print("\n--- DISCUSSION REPORT: OPERATION BOOTSTRAP ---")
    print(f"Finance: {credit_strategy}")
    print(f"Sourcing: {product_pick}")
    print(f"Negotiation: {negotiation_tactic}")
    print(f"Sales: {sales_plan}")
    print(f"CEO DECISION: {final_decision['CEO_Recommendation']} - {final_decision['Strategic_Goal']}")

if __name__ == "__main__":
    asyncio.run(simulate_lean_launch_discussion())
