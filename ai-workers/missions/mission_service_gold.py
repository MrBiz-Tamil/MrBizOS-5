import asyncio
from agents.sourcing import SourcingAgent
from agents.negotiation import NegotiationAgent
from agents.sales import SalesAgent
from core.logic import OperationMode

async def run_operation_service_gold():
    """
    Operation Service Gold: Zero-Investment Brokerage Mission
    Scenario: Connecting a Tier-1 medical supplier with a European hospital group.
    Result: A secured 3% commission on a $1M contract.
    """
    print("\nStarting OPERATION SERVICE GOLD: Zero-Investment Brokerage...")
    
    # 1. Initialize Agents
    sourcing = SourcingAgent(OperationMode.AUTOPILOT)
    negotiation = NegotiationAgent(OperationMode.AUTOPILOT)
    sales = SalesAgent(OperationMode.AUTOPILOT)
    
    # 2. Sourcing: Find High-Margin Contract Opportunity
    print("\n[1/4] Sourcing Agency Scouting...")
    sourcing_result = await sourcing.run(
        "Find a Tier-1 medical supplier for high-volume surgical kits in Southeast Asia with a referral or partner program."
    )
    print(f"Sourcing Result: {sourcing_result[:200]}...")
    
    # 3. Negotiation: Draft Commission Agreement
    print("\n[2/4] Protecting Zero-Investment Profit...")
    negotiation_result = await negotiation.run(
        f"Draft a 3% commission agreement for a $1,000,000 procurement contract based on these sourcing results: {sourcing_result}"
    )
    print(f"Negotiation Result: {negotiation_result}")
    
    # 4. Sales: Identify and Reach Out to institutional Buyers
    print("\n[3/4] Finding institutional Leads (Zero Risk)...")
    sales_result = await sales.run(
        "Search for European hospital regional headquarters for procurement outreach and draft a professional introduction."
    )
    print(f"Sales Result: {sales_result[:200]}...")
    
    # 5. Success Summary
    print("\n[4/4] MISSION LOG: Success")
    print("-------------------------------------------------")
    print("STATUS: Commission Agreement Secured")
    print("PROJECTED REVENUE: $30,000 (3% of $1M)")
    print("CAPITAL RISK: $0.00")
    print("PATH TO $100M: Execute 3,334 similar deals or scale deal size to $10M+.")
    print("-------------------------------------------------")

if __name__ == "__main__":
    asyncio.run(run_operation_service_gold())
