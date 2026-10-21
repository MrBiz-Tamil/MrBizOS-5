from agents.base import BaseAgent
from core.models import AgentType, Task, TaskStatus, OperationMode, GovernanceLevel
from core.database import SessionLocal
from typing import List, Dict, Optional
import json
from datetime import datetime

class SupervisorAgent(BaseAgent):
    """
    The 'Manager' agent that delegates tasks to other agents.
    """
    
    def __init__(self, business_unit_id: Optional[int] = None):
        super().__init__(AgentType.CORE, business_unit_id, OperationMode.AUTOPILOT)
    
    def get_system_prompt(self) -> str:
        return """You are the Chief Operations Supervisor AI.
        Your goal is to ensure the smooth operation of the Global Trading Business.
        
        Responsibilities:
        1. Monitor the Task Queue.
        2. Delegate tasks to the appropriate Specialist Agent (Negotiation, Sourcing, Sales).
        3. Review low-risk actions and Auto-Approve them if they meet Governance Rules.
        4. Escalate high-risk or ambiguous tasks to the Human User.
        """
        
    def run_autonomy_check(self):
        """
        Main loop to check for pending tasks and process them.
        """
        print(f"[{datetime.utcnow()}] Supervisor checking for tasks...")
        db = SessionLocal()
        try:
            # 1. Fetch Pending Tasks
            tasks = db.query(Task).filter(Task.status == TaskStatus.PENDING).all()
            
            if not tasks:
                print("No pending tasks.")
                return
            
            print(f"Found {len(tasks)} pending tasks.")
            
            for task in tasks:
                self.process_task(db, task)
                
            db.commit()
            
        except Exception as e:
            print(f"Error in Supervisor loop: {e}")
            db.rollback()
        finally:
            db.close()

    def process_task(self, db, task: Task):
        """
        Decide what to do with a task.
        """
        print(f"Processing Task #{task.id}: {task.action} ({task.agent_type})")
        
        # 1. Check Governance Level
        if task.governance_level == GovernanceLevel.LEVEL_1_AUTONOMOUS:
            # Auto-Approve
            print(f"  -> Auto-Approving Level 1 Task")
            task.status = TaskStatus.COMPLETED
            task.result = "Auto-approved by Supervisor (Level 1)"
            task.completed_at = datetime.utcnow()
            
        elif task.governance_level == GovernanceLevel.LEVEL_2_EXECUTE_WITH_APPROVAL:
            # If it's waiting for approval, we leave it for human.
            # But maybe we can add a recommendation?
            if not task.reviewer_notes:
                 task.reviewer_notes = "Supervisor Recommendation: Approve (Standard Procedure)"
                 print("  -> Added recommendation for Human Reviewer.")
                 
        elif task.agent_type == AgentType.SOURCING and task.status == TaskStatus.PENDING:
             # Example: Delegate 'Find Suppliers' -> Sourcing Agent
             # Real implementation would call SourcingAgent.run() here
             print("  -> Delegating to Sourcing Agent...")
             # mock execution
             task.status = TaskStatus.IN_PROGRESS
             # In a real system, this would trigger an async job
             
        else:
            print(f"  -> No action taken. Waiting for human.")

# Usage:
# supervisor = SupervisorAgent()
# supervisor.run_autonomy_check()
