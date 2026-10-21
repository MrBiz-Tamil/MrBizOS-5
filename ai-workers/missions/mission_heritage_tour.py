import asyncio
from agents.sourcing import SourcingAgent
from agents.negotiation import NegotiationAgent
from agents.sales import SalesAgent
from core.logic import OperationMode

async def run_operation_heritage_tour():
    """
    Operation Heritage Tour: Luxury Dubai Tour Orchestration
    Scenario: Bundling ultra-luxury hotels, private jets, and exclusive desert experiences.
    Result: A high-margin luxury tour package ready for outreach.
    """
    print("\nStarting OPERATION HERITAGE TOUR: Dubai Luxury Launch...")
    
    # 1. Initialize Agents
    sourcing = SourcingAgent(OperationMode.AUTOPILOT)
    negotiation = NegotiationAgent(OperationMode.AUTOPILOT)
    sales = SalesAgent(OperationMode.AUTOPILOT)
    
    # 2. Sourcing: Bundle Luxury Components
    print("\n[1/4] Travel Sourcing: Assembling Luxury Components...")
    sourcing_result = await sourcing.run(
        "Source ultra-luxury components for a 'Dubai Heritage' package: Burj Al Arab, Private Desert Majlis, and Emirates First-Class blocks."
    )
    print(f"Sourcing Result: {sourcing_result}")
    
    # 3. Negotiation: Analyze Hospitality Yield
    print("\n[2/4] Negotiation Intelligence: Calculating Package Yield...")
    negotiation_result = await negotiation.run(
        f"Analyze yield for the Heritage Tour. Estimated component costs: Hotel $5000, Flight $3000, Activities $2000. Apply 25% luxury markup."
    )
    print(f"Negotiation Result: {negotiation_result}")
    
    # 4. Sales: Generate Immersive Pitch
    print("\n[3/4] Sales Growth: Generating Immersive Pitch...")
    sales_result = await sales.run(
        f"Generate an immersive travel pitch for the Dubai Heritage Tour based on these components: {sourcing_result}. Price point: ${negotiation_result.split('$')[1].split()[0]}"
    )
    print(f"Sales Result:\n{sales_result}")
    
    # 5. Success Summary
    print("\n[4/4] MISSION LOG: Success")
    print("-------------------------------------------------")
    print("STATUS: Luxury Tour Package Ready")
    print("DESIGNATION: 'Dubai Heritage: The Sovereign Experience'")
    print("ESTIMATED MARGIN: 25%+")
    print("NEXT STEP: Deploy AI Video Outreach to GCC-based family offices.")
    print("-------------------------------------------------")

if __name__ == "__main__":
    asyncio.run(run_operation_heritage_tour())
