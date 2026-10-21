from agents.base import BaseAgent
from core.enums import AgentType, OperationMode
from langchain.tools import tool
from datetime import datetime
from typing import List, Optional

class CRMAgent(BaseAgent):
    """AI agent for customer relationship management"""
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.CRM, business_unit_id, operation_mode)
    
    def get_system_prompt(self) -> str:
        return """You are a CRM Agent for a healthcare trading company.
Your role is to:
- Manage customer relationships and data
- Track interactions and follow-ups
- Segment customers by value and engagement
- Suggest next best actions for each lead
- Maintain data quality and completeness

Be organized, detail-oriented, and focus on building long-term customer relationships."""
    
    def get_tools(self) -> list:
        @tool
        def get_lead_details(lead_id: int) -> str:
            """Get detailed information about a specific lead"""
            from core.database import SessionLocal
            from core.models import Lead, Interaction
            db = SessionLocal()
            try:
                lead = db.query(Lead).filter(Lead.id == lead_id).first()
                if not lead:
                    return f"Lead {lead_id} not found"
                
                interactions = db.query(Interaction).filter(Interaction.lead_id == lead_id).all()
                
                result = f"Lead: {lead.name}\n"
                result += f"Company: {lead.company or 'N/A'}\n"
                result += f"Email: {lead.email or 'N/A'}\n"
                result += f"Status: {lead.status}\n"
                result += f"Source: {lead.source or 'N/A'}\n"
                result += f"Interactions: {len(interactions)}\n"
                
                if interactions:
                    result += "\nRecent interactions:\n"
                    for interaction in interactions[-3:]:
                        result += f"- {interaction.type}: {interaction.subject}\n"
                
                return result
            finally:
                db.close()
        
        @tool
        def log_interaction(lead_id: int, interaction_type: str, subject: str, content: str) -> str:
            """Log an interaction with a lead (email, call, meeting)"""
            from core.database import SessionLocal
            from core.models import Interaction
            db = SessionLocal()
            try:
                interaction = Interaction(
                    lead_id=lead_id,
                    type=interaction_type,
                    subject=subject,
                    content=content
                )
                db.add(interaction)
                db.commit()
                return f"Logged {interaction_type} interaction with lead {lead_id}"
            except Exception as e:
                return f"Error logging interaction: {str(e)}"
            finally:
                db.close()
        
        @tool
        def update_lead_status(lead_id: int, new_status: str) -> str:
            """Update the status of a lead (new, contacted, qualified, converted, lost)"""
            from core.database import SessionLocal
            from core.models import Lead
            db = SessionLocal()
            try:
                lead = db.query(Lead).filter(Lead.id == lead_id).first()
                if not lead:
                    return f"Lead {lead_id} not found"
                
                old_status = lead.status
                lead.status = new_status
                db.commit()
                return f"Updated lead {lead_id} status from '{old_status}' to '{new_status}'"
            except Exception as e:
                return f"Error updating status: {str(e)}"
            finally:
                db.close()
        
        @tool
        def segment_leads(criteria: str) -> str:
            """Segment leads based on criteria (status, source, engagement)"""
            from core.database import SessionLocal
            from core.models import Lead
            db = SessionLocal()
            try:
                if criteria == "status":
                    leads = db.query(Lead.status, db.func.count(Lead.id)).group_by(Lead.status).all()
                    result = "Leads by status:\n"
                    for status, count in leads:
                        result += f"- {status}: {count}\n"
                elif criteria == "source":
                    leads = db.query(Lead.source, db.func.count(Lead.id)).group_by(Lead.source).all()
                    result = "Leads by source:\n"
                    for source, count in leads:
                        result += f"- {source or 'Unknown'}: {count}\n"
                else:
                    result = "Supported criteria: status, source"
                
                return result
            finally:
                db.close()
        
        @tool
        def suggest_next_action(lead_id: int) -> str:
            """Suggest the next best action for a lead based on their history"""
            from core.database import SessionLocal
            from core.models import Lead, Interaction
            db = SessionLocal()
            try:
                lead = db.query(Lead).filter(Lead.id == lead_id).first()
                if not lead:
                    return f"Lead {lead_id} not found"
                
                interactions = db.query(Interaction).filter(Interaction.lead_id == lead_id).all()
                
                if lead.status == "new":
                    return "Suggested action: Send initial outreach email"
                elif lead.status == "contacted" and len(interactions) < 2:
                    return "Suggested action: Follow up with a call"
                elif lead.status == "qualified":
                    return "Suggested action: Schedule product demo or send proposal"
                elif len(interactions) == 0:
                    return "Suggested action: Make first contact"
                else:
                    return "Suggested action: Review interaction history and send personalized follow-up"
            finally:
                db.close()
        
        @tool
        def get_all_leads_summary() -> str:
            """Get a summary of all leads in the system"""
            from core.database import SessionLocal
            from core.models import Lead
            db = SessionLocal()
            try:
                total = db.query(Lead).count()
                new = db.query(Lead).filter(Lead.status == "new").count()
                contacted = db.query(Lead).filter(Lead.status == "contacted").count()
                qualified = db.query(Lead).filter(Lead.status == "qualified").count()
                
                return f"Total Leads: {total}\nNew: {new}\nContacted: {contacted}\nQualified: {qualified}"
            finally:
                db.close()
        
        return [get_lead_details, log_interaction, update_lead_status, segment_leads, 
                suggest_next_action, get_all_leads_summary]
