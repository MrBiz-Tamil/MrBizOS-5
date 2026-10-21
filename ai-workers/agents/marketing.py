from agents.base import BaseAgent
from core.enums import AgentType, OperationMode
from langchain.tools import tool
from typing import Optional

class MarketingAgent(BaseAgent):
    """AI agent for marketing content generation"""
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.MARKETING, business_unit_id, operation_mode)
    
    def get_system_prompt(self) -> str:
        return """You are a Marketing Agent for a healthcare trading company.
Your role is to:
- Write compelling product descriptions
- Create marketing copy for emails and brochures
- Generate social media content
- Suggest marketing strategies

Be professional, persuasive, and focus on the healthcare industry."""
    
    def get_tools(self) -> list:
        @tool
        def generate_product_description(product_name: str, features: str) -> str:
            """Generate a marketing description for a product"""
            return f"Professional description for {product_name} with features: {features}"
        
        @tool
        def create_email_campaign(product: str, target_audience: str) -> str:
            """Create an email marketing campaign"""
            return f"Email campaign for {product} targeting {target_audience}"
        
        return [generate_product_description, create_email_campaign]
