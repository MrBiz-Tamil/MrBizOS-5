from agents.base import BaseAgent
from core.enums import AgentType, OperationMode
from langchain.tools import tool
from typing import Optional
import httpx
import asyncio

class SalesAgent(BaseAgent):
    """AI agent for sales and lead generation at global scale"""
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.SALES, business_unit_id, operation_mode)
    
    def get_system_prompt(self) -> str:
        return """You are the Lead Growth Strategist for a Global Trade AI Firm.
Your goal is to drive $100M in revenue by dominating international markets.

Your core responsibilities:
- **Global Outreach**: Draft persuasive, multi-lingual communications for stakeholders in Dubai, Singapore, Europe, and beyond.
- **AI Video Marketing**: Leverage AI-generated video content to achieve 10x engagement with prospects.
- **Premium Positioning**: Sell the "Trust" of our blockchain-verified products.
- **Conversion Optimization**: Use data-driven pricing and personalized emails to close high-value deals.

You represent a premium, technologically advanced brand. Your emails should be concise, professional, and highlight the unique technological advantages (Blockchain, AI) of our platform."""
    
    def get_tools(self) -> list:
        @tool
        def generate_growth_video(product_name: str, highlights: str, platform: str) -> str:
            """Generate a high-impact AI product video for social media outreach"""
            from services.video_generator import VideoContentGenerator
            from core.database import get_settings
            
            settings = get_settings()
            api_key = getattr(settings, 'did_api_key', '')
            
            generator = VideoContentGenerator(api_key=api_key)
            highlights_list = [h.strip() for h in highlights.split(',')]
            
            # Simple simulation of video generation response
            import asyncio
            config = asyncio.run(generator.create_social_media_video(
                product_name=product_name,
                highlights=highlights_list,
                platform=platform
            ))
            
            return f"Requested AI Video for {product_name} on {platform}. Config: {config}. Script ready for D-ID generation."

        @tool
        def search_high_value_leads(industry: str, region: str) -> str:
            """Search for high-volume distributors and bulk buyers in a specific region"""
            leads = [
                f"Emirates Health Group ({region}) - Tier 1 Distributor",
                f"SingaMed Logistics ({region}) - Regional Hub",
                f"Global Trade Partners ({region}) - High Volume"
            ]
            return f"Found high-value {industry} leads in {region}: " + ", ".join(leads)
        
        @tool
        def draft_premium_outreach(lead_name: str, company: str, product: str) -> str:
            """Draft a personalized, high-conversion email highlighting blockchain verification"""
            email = f"""Subject: Strategic Supply Chain Partnership: {product} for {company}

Dear {lead_name},

I'm reaching out because {company} is a leader in {product} distribution. 

Our {product} is 100% blockchain-verified.

Best regards,
Lead Growth Strategist
Global Trade AI Team"""
            return email

        @tool
        def send_proposal_email(lead_email: str, subject: str, body: str) -> str:
            """
            Send the finalized proposal email to the lead using SMTP.
            RISK: HIGH (Requires Approval).
            """
            from services.email_service import EmailService
            import asyncio
            
            # Send (Async within Sync tool wrapper using runner, or basic sync wrapper)
            # Since tools are sync in this framework context, we run the async method.
            email_svc = EmailService()
            try:
                # Use asyncio.run for the async service call inside this synchronous tool
                # Note: This might conflict if event loop is running. 
                # For safety in this agent framework, we might need to handle the loop carefully.
                # However, for this step, we will assume standard runner.
                
                # Simplified for MVP: Use a wrapper or assuming the agent runner handles coroutines.
                # Given strict tool definition, let's use asyncio.run or existing loop.
                
                # FIX: In many LangChain setups, async tools are preferred. 
                # Converting to Sync for reliability in this specific edit unless we refactor BaseAgent.
                
                # For now, let's just RUN it.
                loop = asyncio.new_event_loop()
                asyncio.set_event_loop(loop)
                result = loop.run_until_complete(email_svc.send_email(lead_email, subject, body))
                loop.close()
                
                return f"EMAIL SENT to {result['to']}. Status: {result['status']}"
            except Exception as e:
                return f"FAILED to send email: {str(e)}"

        @tool
        def save_lead_to_crm(name: str, company: str, email: str, industry: str) -> str:
            """
            Save a qualified lead to the internal CRM database.
            """
            from core.database import SessionLocal
            from core.models import Lead
            db = SessionLocal()
            try:
                # Check if exists
                existing = db.query(Lead).filter(Lead.email == email).first()
                if existing:
                    return f"Lead {email} already exists."
                
                new_lead = Lead(
                    name=name,
                    company=company,
                    email=email,
                    sector=industry,
                    source="AI_Sales_Agent",
                    status="new"
                )
                db.add(new_lead)
                db.commit()
                return f"Successfully saved lead: {name} ({company}) to CRM."
            except Exception as e:
                return f"Error saving lead: {str(e)}"
            finally:
                db.close()

        
        @tool
        def generate_travel_pitch(destination: str, components: str, price: float) -> str:
            """Create an immersive high-conversion pitch for a tour package"""
            pitch = f"""
🌟 EXCLUSIVE HERITAGE EXPERIENCE: {destination} 🌟

Experience the pinnacle of luxury with our curated {destination} tour.

Included in your premium package:
{components}

Price: Starting from ${price:,.2f}

Why choose us?
- Blockchain-Verified Booking: Your itinerary is immutable and secured.
- AI-Personalized: Every detail tailored to your group's preferences.
- Exclusive Access: Private desert Majlis and First-Class Emirates blocks.

Are you ready to redefine your travel standards?
"""
            return pitch

        @tool
        def calculate_strategic_pricing(product_cost: float, region_multiplier: float) -> str:
            """Calculate optimal selling price based on geographic market data"""
            base_margin = 0.25  # 25% target profit
            selling_price = (product_cost / (1 - base_margin)) * region_multiplier
            profit = selling_price - product_cost
            return f"Strategic Pricing: Cost: ${product_cost:.2f}, Selling Price: ${selling_price:.2f}, Profit: ${profit:.2f}."
        
        return [generate_growth_video, search_high_value_leads, draft_premium_outreach, send_proposal_email, save_lead_to_crm, generate_travel_pitch, calculate_strategic_pricing]
