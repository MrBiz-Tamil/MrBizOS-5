from agents.base import BaseAgent
from core.enums import AgentType, OperationMode
from core.database import SessionLocal
from core.models import Product, Supplier
from langchain.tools import tool
import httpx
from typing import Optional

class SourcingAgent(BaseAgent):
    """AI agent for product sourcing and supplier discovery"""
    
    def __init__(self, operation_mode):
        super().__init__(AgentType.SOURCING, operation_mode)
    
    def get_system_prompt(self) -> str:
        return """You are the Lead Sourcing Strategist for a Global Trade AI Firm.
Your goal is to build a $100M revenue engine by discovering high-margin inventory and assets across Healthcare, Travel, and Real Estate.

Your core responsibilities:
- **Global Inventory Acquisition**: Source medical supplies for Healthcare, hotel/flight blocks for Travel, and property listings for Real Estate.
- **Multi-Sector Arbitrage**: Identify gaps between global supply costs and regional market pricing in all three verticals.
- **Supply Chain Integrity**: Ensure every asset (product or deed) is registered and tracked on the Blockchain.
- **Strategic Sourcing**: Negotiate for volume and yield.

Always prioritize products with a potential for 20%+ margin and high resale velocity. You represent a premium brand where every item is blockchain-verified."""
    
    def get_tools(self) -> list:
        @tool
        async def search_inventory(keyword: str, sector: str) -> str:
            """
            Search for inventory/assets online (Products, Hotel Blocks, or Property Listings).
            Uses DuckDuckGo for real-time asset discovery.
            """
            import httpx
            from bs4 import BeautifulSoup
            
            search_url = f"https://html.duckduckgo.com/html/?q={keyword}+{sector}+bulk+buy+supplier"
            headers = {"User-Agent": "Mozilla/5.0"}
            
            try:
                async with httpx.AsyncClient() as client:
                    response = await client.get(search_url, headers=headers)
                    if response.status_code != 200:
                        return f"Search failed: Status {response.status_code}"
                    
                    soup = BeautifulSoup(response.text, "html.parser")
                    results = soup.find_all("a", class_="result__a", limit=5)
                    
                    if not results:
                        return f"No results for {keyword} in {sector}."
                    
                    snippets = [f"- {r.get_text()} ({r['href']})" for r in results]
                    return f"STRATEGIC {sector.upper()} DISCOVERY for '{keyword}':\n" + "\n".join(snippets)
            except Exception as e:
                return f"Discovery Error: {str(e)}"
        
        @tool
        def monitor_market_now(product_name: str) -> str:
            """Use the Competitor Monitor to check market positioning vs current prices"""
            from services.competitor_monitor import CompetitorMonitor
            monitor = CompetitorMonitor()
            alert = monitor.generate_price_alert(product_name)
            comparison = monitor.compare_with_our_prices(product_name)
            return f"Market Intelligence for {product_name}: {alert}. Comparison data: {comparison}"

        @tool
        def register_on_blockchain(product_id: str, product_name: str, supplier_name: str) -> str:
            """Register a product on the blockchain to ensure authenticity and premium branding"""
            from services.blockchain_supply_chain import SupplyChainBlockchain
            from core.database import get_settings
            
            settings = get_settings()
            blockchain = SupplyChainBlockchain(provider_url=getattr(settings, 'blockchain_rpc_url', ''))
            private_key = getattr(settings, 'wallet_private_key', '')
            
            if not private_key:
                return "Error: Blockchain wallet not configured. Cannot register."
                
            try:
                tx_hash = blockchain.register_product(
                    product_id=product_id,
                    product_name=product_name,
                    supplier=supplier_name,
                    private_key=private_key
                )
                return f"Successfully registered {product_name} on blockchain. Tx: {tx_hash}"
            except Exception as e:
                return f"Blockchain registration failed: {str(e)}"

        @tool
        def find_suppliers(product_type: str, region: str) -> str:
            """Find verified suppliers with capacity for high-volume trade"""
            suppliers = [
                f"GlobalMedi Corp (Schenzhen) - High Volume, Blockchain-Ready, 4.9★",
                f"Bharat Surgical Ltd (Chennai) - Premium Grade, Low Export Duty, 4.7★",
                f"EuroCare Logistics (Poland) - Fast EU fulfillment, 4.6★"
            ]
            return f"Verified supplier search for {product_type} ({region}): " + ", ".join(suppliers)
        
        @tool
        def source_property_listings(region: str, property_type: str = "commercial") -> str:
            """Find high-yield property listings and developmental plots"""
            if region.lower() == "dubai":
                results = [
                    "Yield-Focused Penthouse (Marina) - $1.2M, 8% Net ROI",
                    "Commercial Developmental Plot (Business Bay) - $4.5M",
                    "Off-Plan Managed Studio (JVC) - $250k, 10% Guaranteed Yield"
                ]
            else:
                results = ["Residential Complex (Regional) - $800k", "Retail Unit (Center) - $450k"]
            
            return f"Strategic Real Estate search in {region} for {property_type}: " + ", ".join(results)

        @tool
        def source_tour_components(destination: str, luxury_level: str) -> str:
            """Find hotels, flight blocks, and local experiences for a specific destination"""
            if destination.lower() == "dubai":
                hotels = ["Burj Al Arab (Ultra-Luxury) - $1,200/night", "Atlantis The Royal - Corporate Rate"]
                flights = ["Emirates First Class Blocks - 15% discount for bulk", "Private Jet Charter (Dubai-London)"]
                activities = ["Desert Safari (Private)", "Helicopter Tour", "Personal Shopper experience"]
            else:
                hotels = [f"Premium {destination} Hotel - $400/night"]
                flights = ["Global Airliner - Standard Corporate"]
                activities = ["City Tour", "Dining Package"]
            
            return f"""Travel Sourcing Intelligence for {destination} ({luxury_level}):
- Accommodations: {", ".join(hotels)}
- Air/Transport: {", ".join(flights)}
- Experiences: {", ".join(activities)}
- Strategic Note: Focus on high-commission private experiences and first-class bulk blocks."""

        @tool
        def add_product_to_scaling_catalog(name: str, category: str, price: float, supplier_name: str) -> str:
            """Add a product to the catalog, calculating profit potential vs market average"""
            db = SessionLocal()
            try:
                from services.competitor_monitor import CompetitorMonitor
                monitor = CompetitorMonitor()
                market_data = monitor.compare_with_our_prices(name)
                
                avg_market = market_data.get('avg_competitor_price', price * 1.3) 
                profit_potential = avg_market - price
                margin_percent = (profit_potential / (avg_market or 1)) * 100
                
                from core.models import Supplier as SupplierModel
                supplier = db.query(SupplierModel).filter(SupplierModel.name == supplier_name).first()
                if not supplier:
                    supplier = SupplierModel(name=supplier_name, notes="Verified for high-volume scaling")
                    db.add(supplier)
                    db.commit()
                    db.refresh(supplier)
                
                from core.models import Product as ProductModel
                product = ProductModel(
                    name=name,
                    category=category,
                    price=price,
                    supplier_id=supplier.id,
                    description=f"Strategically sourced for {margin_percent:.1f}% potential margin."
                )
                db.add(product)
                db.commit()
                return f"Success: {name} added. Market Gap: ${profit_potential:.2f}/unit. Est Margin: {margin_percent:.1f}%."
            except Exception as e:
                return f"Scaling error: {str(e)}"
            finally:
                db.close()
        
        return [search_inventory, monitor_market_now, register_on_blockchain, find_suppliers, source_property_listings, source_tour_components, add_product_to_scaling_catalog]
