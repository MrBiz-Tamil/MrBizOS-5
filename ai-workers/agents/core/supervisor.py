from agents.base import BaseAgent
from core.enums import AgentType, OperationMode, TaskStatus
from core.models import Task
from core.database import SessionLocal
from langchain.tools import tool
from typing import Optional
from datetime import datetime
import json

class SupervisorAgent(BaseAgent):
    """
    The Orchestrator. It doesn't do work itself; it delegates to other agents.
    It sits at the top of the hierarchy.
    """
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.SUPERVISOR, business_unit_id, operation_mode)
    
    def get_system_prompt(self) -> str:
        return """You are the **Supervisor Agent**.
Your role is to coordinate the specialized agents (Sales, Finance, Researcher) to achieve complex goals.

**Capabilities:**
- Delegate tasks to specific agents.
- Synthesize results from multiple agents.
- Report final outcomes to the user.

**Agents Available:**
- **Sales Agent**: For leads, outreach, CRM.
- **Finance Agent**: For invoices, expenses, reports.
- **Research Agent**: For market data, competitor analysis.

If a user request requires multiple steps (e.g., "Find leads and email them"), break it down and delegate."""

    def get_tools(self) -> list:
        @tool
        def delegate_task(target_agent: str, instruction: str) -> str:
            """Delegate a specific instruction to a sub-agent (sales, finance, research)."""
            db = SessionLocal()
            try:
                # Map string to AgentType Enum
                agent_type_map = {
                    "sales": AgentType.SALES,
                    "finance": AgentType.FINANCE,
                    "research": AgentType.MARKET_RESEARCH,
                    "crm": AgentType.CRM,
                    "logistics": AgentType.LOGISTICS,
                    "compliance": AgentType.COMPLIANCE,
                    "sourcing": AgentType.SOURCING
                }
                
                target = agent_type_map.get(target_agent.lower())
                if not target:
                    return f"Error: Unknown agent '{target_agent}'. Available: {list(agent_type_map.keys())}"

                # Create the task
                from core.models import Task, GovernanceRule 
                from core.enums import GovernanceLevel
                import json
                
                required_level = GovernanceLevel.LEVEL_2_EXECUTE_WITH_APPROVAL
                
                new_task = Task(
                    business_unit_id=self.business_unit_id,
                    agent_type=target,
                    action="delegated_instruction",
                    description=f"[DELEGATED BY SUPERVISOR] {instruction}",
                    payload=json.dumps({"instruction": instruction}),
                    status=TaskStatus.PENDING,
                    governance_level=required_level,
                    mode=self.operation_mode
                )
                
                db.add(new_task)
                db.commit()
                db.refresh(new_task)
                
                return f"Delegated to {target_agent.upper()}. Task ID: {new_task.id}. Status: {new_task.status}."
                
            except Exception as e:
                return f"Delegation failed: {str(e)}"
            finally:
                db.close()

        return [delegate_task]

    async def run_autonomy_check(self):
        """
        Scheduled Job: Periodically checks for stalled workflows and triggers corrective actions.
        """
        print(f"[{datetime.now()}] Supervisor: Running Autonomy Check...")
        db = SessionLocal()
        try:
            from core.models import Lead, Task
            from datetime import timedelta
            
            # 1. Check for Stalled Leads (New > 2 days)
            cutoff = datetime.utcnow() - timedelta(days=2)
            stalled_leads = db.query(Lead).filter(Lead.status == "new", Lead.created_at < cutoff).all()
            
            if stalled_leads:
                print(f"[{datetime.now()}] Supervisor: Found {len(stalled_leads)} stalled leads. Delegating cleanup...")
                for lead in stalled_leads:
                    # Create a Task for Sales Agent
                    task = Task(
                        business_unit_id=lead.business_unit_id,
                        agent_type=AgentType.SALES,
                        action="re_engage_lead",
                        description=f"Autonomy Trigger: Re-engage stalled lead {lead.name} (Idle > 48h)",
                        payload=json.dumps({"lead_id": lead.id}),
                        status=TaskStatus.PENDING,
                        mode=OperationMode.AUTOPILOT # Try to auto-run if rules allow
                    )
                    db.add(task)
                    # Mark lead as 'processing' so we don't loop forever
                    lead.status = "processing"
                
                db.commit()
                print(f"[{datetime.now()}] Supervisor: Created re-engagement tasks.")
            else:
                print(f"[{datetime.now()}] Supervisor: No stalled leads found.")

        except Exception as e:
            print(f"ERROR: Autonomy Check failed: {e}")
        finally:
            db.close()
