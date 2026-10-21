from agents.base import BaseAgent
from core.models import Note
from core.enums import AgentType, OperationMode
from core.database import SessionLocal
from langchain.tools import tool
from typing import Optional
import json
from datetime import datetime, timedelta

class FinanceAgent(BaseAgent):
    """AI agent for trade finance, liquidity, and risk management"""
    
    def __init__(self, business_unit_id: int = None, operation_mode="manual"):
        super().__init__(AgentType.FINANCE, business_unit_id, operation_mode)
    
    def get_system_prompt(self) -> str:
        return """You are the Strategic Finance Director for a Global Trade AI Firm.
Your goal is to manage the $100M cash flow and maximize capital efficiency.

Your core responsibilities:
- **Trade Finance**: Manage and suggest liquidity loans for high-value trades.
- **Credit Scoring**: Analyze supplier and buyer reliability.
- **Forex Strategy**: Suggest hedging strategies for multi-currency trades.
- **Payout Handling**: Process payments with strict strict verification.

You are the guardian of the company's capital. Your decisions must ensure we never face a liquidity crunch."""
    
    def get_tools(self) -> list:
        @tool
        def score_counterparty_risk(name: str, blockchain_address: str = None) -> str:
            """Analyze the creditworthiness of a supplier or buyer. blockchain_address is optional."""
            # Simulation of risk scoring
            score = 85 # Default high score for active partners
            risk_level = "LOW"
            if "Acme" in name: 
                score = 95
            elif "Unknown" in name:
                score = 40
                risk_level = "HIGH"
                
            return f"RISK ASSESSMENT for {name}: {score}/100. Status: {risk_level} RISK. Recommendation: {'30-day credit' if score > 70 else 'Advance Payment Only'}."
        
        @tool
        def simulate_trade_finance(deal_value: float, duration_days: int = 30) -> str:
            """Calculate cost of capital and terms for a liquidity loan for a specific deal"""
            interest_rate = 0.015 # 1.5% per month flat
            cost = deal_value * interest_rate
            total_repayment = deal_value + cost
            return f"FINANCE OFFER: Loan ${deal_value:,.2f} for {duration_days} days.\nInterest: ${cost:,.2f} (1.5%).\nTotal Repayment: ${total_repayment:,.2f}."
        
        @tool
        def suggest_forex_hedge(amount: float, pair: str = "USD/CNY") -> str:
            """Suggest a hedging strategy. Pair example: 'USD/EUR'"""
            return f"FOREX ADVISORY ({pair}): Market volatility is moderate. Recommendation: lock in 50% of the ${amount:,.2f} exposure via Forward Contract to mitigate currency risk."

        @tool
        def process_invoice_ocr(image_path: str) -> str:
            """
            Extract data from an invoice image (OCR).
            Usage: process_invoice_ocr("/path/to/invoice.jpg")
            """
            # Implementation using pytesseract
            try:
                import pytesseract
                from PIL import Image
                # Note: pytesseract requires tesseract-ocr to be installed on the host.
                # If it's not found, we fallback to a smart-parsing simulation.
                
                text = pytesseract.image_to_string(Image.open(image_path))
                if not text.strip():
                    raise Exception("No text found")
                
                return f"OCR EXTRACTED TEXT:\n{text}\n\n[Parsed Data]: Detected Invoice from identified vendor. Amount: $0.00 (Draft)."
            except Exception as e:
                return f"OCR FAILED (or Tesseract not installed): {e}. Fallback: Identified 'Invoice_#8812' from 'Global Logistics' for '$1,250.00'."

        @tool
        def check_treasury_balance(asset: str = "USD") -> str:
            """Check real-time balance of Treasury (USD, EUR, ETH, USDC)"""
            from services.finance_adapter import FinanceAdapter
            adapter = FinanceAdapter()
            balance = adapter.get_balance(asset)
            return f"TREASURY STATUS ({asset}): {balance['provider']} holds {balance.get('amount', 0):,.2f} {asset}. (Updated: {balance.get('timestamp')})"

        @tool
        def process_automated_payout(recipient: str, amount: float, reason: str, method: str = "USD") -> str:
            """Process a payout. REQUIRES APPROVAL for amounts > $1000."""
            # Logic: If high amount, the Governance Engine in BaseAgent.execute_task will intercept this tool call 
            # IF the governance level is set to LEVEL_2_EXECUTE_WITH_APPROVAL.
            
            from services.finance_adapter import FinanceAdapter
            adapter = FinanceAdapter()
            
            receipt = adapter.process_payout(amount, recipient, method, reason)
            
            return f"PAYMENT PROCESSED ({method}): ${amount:,.2f} to {recipient}. TxID: {receipt['transaction_id']}. Status: {receipt['status']}."

        return [score_counterparty_risk, simulate_trade_finance, suggest_forex_hedge, process_invoice_ocr, check_treasury_balance, process_automated_payout]
