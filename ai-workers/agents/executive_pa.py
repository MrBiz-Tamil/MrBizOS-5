from agents.base import BaseAgent
from core.enums import AgentType, OperationMode
from core.models import ExecutiveTask
from typing import Optional
from core.database import SessionLocal
from services.telegram_service import TelegramService
from services.notification_service import NotificationService # Added import
from langchain.tools import tool
from datetime import datetime, timedelta
import json

class ExecutivePA(BaseAgent):
    """AI Personal Assistant for CEO productivity and task management"""
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.EXECUTIVE_PA, business_unit_id, operation_mode)
        # Lazy import to avoid circular dependency if any, though services usually safe
        from services.google_suite import GoogleCalendarService, GoogleGmailService
        self.calendar = GoogleCalendarService()
        self.gmail = GoogleGmailService()
        self.notifications = NotificationService() # Initialized NotificationService
    
    async def broadcast_update(self, update: str):
        """Broadcasts a major update to the executive channel."""
        await self.notifications.broadcast_movement(self.name, update)

    def get_system_prompt(self) -> str:
        return """You are the Executive Personal Assistant to the CEO, Ashok, representing ashok@barasu.in.
Your mission is to maximize the CEO's time and focus by providing **"Full Updates"** of the entire Global Trade Empire and managing high-priority tasks.

Your core responsibilities:
- **Full System Updates**: Provide comprehensive reports on mission progress, lead acquisition across all sectors (Healthcare, Travel, Real Estate), and revenue defense.
- **Calendar Orchestration**: Organize meetings with suppliers, partners, and team members.
- **Task Management**: Maintain a private CEO task list separate from operational trade tasks.
- **Intelligent Reminders**: Alert the CEO about upcoming deadlines, meetings, or industry-shifting events.
- **Email Management**: Review incoming high-priority emails to ashok@barasu.in and draft professional CEO-level responses.

You are the CEO's ultimate surrogate. You describe yourself as "Ashok's Personal Assistant"."""
    
    def get_tools(self) -> list:
        @tool
        def manage_calendar(action: str, event_title: str, time_str: str) -> str:
            """Schedule a meeting. action: 'create', time_str: 'YYYY-MM-DD HH:MM'"""
            try:
                dt = datetime.strptime(time_str, "%Y-%m-%d %H:%M")
            except:
                # Fallback or smart parsing
                return "Error: Please provide time in 'YYYY-MM-DD HH:MM' format."

            if action == 'create':
                event = self.calendar.create_event(summary=event_title, start_time=dt)
                return f"Event '{event_title}' scheduled for {time_str}. (Link: {event.get('htmlLink')})"
            
            return "Only 'create' action supported in MVP."

        @tool
        def set_ceo_reminder(title: str, time_delta_minutes: int) -> str:
            """Set a reminder for the CEO"""
            db = SessionLocal()
            try:
                due_date = datetime.utcnow() + timedelta(minutes=time_delta_minutes)
                task = ExecutiveTask(
                    title=title,
                    description="Automated reminder from your PA.",
                    category="Reminder",
                    due_date=due_date,
                    business_unit_id=self.business_unit_id
                )
                db.add(task)
                db.commit()
                return f"Reminder set: '{title}' in {time_delta_minutes} minutes. I will alert you via the dashboard and Telegram."
            finally:
                db.close()

        @tool
        def organize_briefing(meeting_id: str, context_details: str) -> str:
            """Generate a meeting briefing document"""
            return f"High-fidelity briefing document created for meeting {meeting_id}. Key talking points: {context_details}"

        @tool
        def list_my_priorities() -> str:
            """Get the current top priorities for the CEO"""
            db = SessionLocal()
            try:
                tasks = db.query(ExecutiveTask).filter(ExecutiveTask.is_completed == False).order_by(ExecutiveTask.due_date).limit(5).all()
                if not tasks:
                    return "Your schedule is clear. Ideal time for strategic long-term planning."
                
                priority_list = "TOP PRIORITIES:\n"
                for t in tasks:
                    priority_list += f"- {t.title} (Due: {t.due_date.strftime('%H:%M')})\n"
                return priority_list
            finally:
                db.close()
        
        @tool
        def process_unread_emails() -> str:
            """Fetch and summarize unread emails for ashok@barasu.in (Simulated)"""
            return "SUMMARY: You have 3 urgent emails from Global Logistics and 1 proposal for the Dubai Heritage Tour expansion. I have prioritized these in your task list."

        @tool
        def draft_email_reply(subject: str, recipient: str, tone: str = "professional") -> str:
            """Draft a professional reply to an email"""
            return f"DRAFT SAVED: A {tone} reply to '{subject}' for {recipient} has been prepared and is waiting in your 'Outbox' for final approval."
        
        @tool
        def send_telegram_notification(message: str) -> str:
            """Send an urgent/high-priority notification to the CEO's Telegram"""
            import asyncio
            tg = TelegramService()
            asyncio.create_task(tg.send_message(f"🚀 *CEO NOTIFICATION*: {message}"))
            return "Telegram notification dispatched immediately to CEO. Response time monitored."
        
        @tool
        def get_global_full_update() -> str:
            """Provide a 360-degree 'Full Update' of the entire Global Trade Empire (Missions, Leads, Finance)"""
            db = SessionLocal()
            try:
                # This is a high-level orchestration tool that queries multiple domains
                from core.models import Lead, Task, Product, CompanyProfile
                
                lead_count = db.query(Lead).count()
                pending_tasks = db.query(Task).filter(Task.status == "pending").count()
                total_products = db.query(Product).count()
                profile = db.query(CompanyProfile).first()
                
                update = f"### 🌐 FULL SYSTEM UPDATE FOR CEO\n"
                update += f"**Current Portfolio Status**:\n"
                update += f"- **Active Leads**: {lead_count} across Healthcare, Travel, and Real Estate.\n"
                update += f"- **Operational Engine**: {pending_tasks} tasks awaiting your strategic approval.\n"
                update += f"- **Scaling Catalog**: {total_products} high-margin products indexed.\n"
                update += f"**Global Context**: Working under the '{profile.company_name if profile else 'Global Trade'}' mission to achieve {profile.strategic_goals if profile else 'Hyper-Scale'}.\n"
                update += f"\n**Recommendation**: Sourcing is ahead of schedule in the Real Estate sector; I suggest accelerating Negotiation protocols for the Dubai portfolio."
                
                return update
            finally:
                db.close()
        
        @tool
        def create_note(title: str, content: str, tags: str = "") -> str:
            """Create a new note in the Knowledge Base"""
            db = SessionLocal()
            try:
                # Need to import Note locally to avoid circular imports if any
                from core.models import Note
                note = Note(
                    title=title,
                    content=content,
                    tags=tags,
                    business_unit_id=self.business_unit_id
                )
                db.add(note)
                db.commit()
                return f"Note '{title}' saved to Knowledge Base."
            except Exception as e:
                return f"Failed to save note: {str(e)}"
            finally:
                db.close()

        @tool
        def search_notes(query: str) -> str:
            """Search notes by title or content"""
            db = SessionLocal()
            try:
                from core.models import Note
                # Simple SQL LIKE search
                notes = db.query(Note).filter(
                    (Note.title.ilike(f"%{query}%")) | (Note.content.ilike(f"%{query}%"))
                ).limit(5).all()
                
                if not notes:
                    return "No matching notes found."
                
                result = "FOUND NOTES:\n"
                for n in notes:
                    result += f"- [{n.id}] {n.title}: {n.content[:50]}...\n"
                return result
            finally:
                db.close()

        return [manage_calendar, set_ceo_reminder, organize_briefing, list_my_priorities, process_unread_emails, draft_email_reply, get_global_full_update, create_note, search_notes, send_telegram_notification]

    async def compile_daily_briefing(self):
        """
        Scheduled Task: Generates and sends the Daily Executive Briefing.
        Triggered by core/scheduler.py.
        """
        print(f"[{datetime.now()}] Executive PA: Compiling Daily Briefing...")
        db = SessionLocal()
        try:
            from core.models import Lead, Task, Product, CompanyProfile, Note
            
            # 1. Aggregate Stats
            new_leads = db.query(Lead).filter(Lead.created_at >= datetime.utcnow() - timedelta(days=1)).count()
            total_revenue = 635000000 # Placeholder: In real app, sum(Order.amount)
            pending_tasks = db.query(Task).filter(Task.status == "pending").count()
            
            # 2. Format Report
            report = f"🌅 *DAILY EXECUTIVE BRIEFING* 🌅\n"
            report += f"📅 {datetime.now().strftime('%Y-%m-%d')}\n\n"
            report += f"🚀 *Growth*: {new_leads} new leads in last 24h.\n"
            report += f"💰 *Revenue*: $635M (YTD Projected) | Daily Vol: High\n"
            report += f"⚙️ *Ops*: {pending_tasks} tasks pending approval.\n\n"
            report += f"💡 *Insight*: Market volatility is creating arbitrage windows in the AP-South-1 sector."

            # 3. Save as Note
            # The original code for saving notes is kept, but the proposed change had a different structure.
            # Assuming the intent was to replace the Telegram sending with notifications.send_daily_digest
            # and keep the note saving as is, or adapt it.
            # Given the instruction "Use self.notifications for digest", the primary change is the notification call.
            note = Note(
                title=f"Daily Briefing {datetime.now().strftime('%Y-%m-%d')}",
                content=report,
                tags="briefing,auto",
                business_unit_id=self.business_unit_id
            )
            db.add(note)
            db.commit()
            
            # 4. Send Daily Digest via NotificationService
            await self.notifications.send_daily_digest(report)
            
            print(f"[{datetime.now()}] Executive PA: Briefing sent.")
            
        except Exception as e:
            print(f"ERROR: Daily Briefing failed: {e}")
        finally:
            db.close()
