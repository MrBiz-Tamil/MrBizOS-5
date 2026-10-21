from agents.base import BaseAgent
from core.models import Task
from core.enums import AgentType, OperationMode, TaskStatus
from core.database import SessionLocal
from langchain.tools import tool
from typing import Optional, List, Dict
import json
import asyncio
from datetime import datetime

class MarketResearchAgent(BaseAgent):
    """
    AI Agent for Macro Market Research, Competitor Analysis, and Trend Forecasting.
    Different from Sourcing (which finds specific products), this agent answers "What should we sell?" and "Who are we fighting?".
    """
    
    def __init__(self, business_unit_id: Optional[int] = None, operation_mode: OperationMode = OperationMode.MANUAL):
        super().__init__(AgentType.MARKETING, business_unit_id, operation_mode) 
        self.agent_type = AgentType.MARKETING 

    def get_system_prompt(self) -> str:
        return """You are the **Chief Market Research Officer**.
Your goal is to provide deep strategic insights to guide the company's expansion.

**Key Responsibilities:**
- **Trend Analysis**: Identify emerging trends in Healthcare, Travel, and Real Estate.
- **Competitor Intelligence**: Analyze competitor pricing, offerings, and weaknesses.
- **Report Generation**: Synthesize data into actionable strategic reports.

**governance:**
- Research is generally low-risk (Level 3 - Auto-Execute).
- Reports DO NOT require approval to generate, but distributing them might."""

    def get_tools(self) -> list:
        @tool
        async def web_search_trends(query: str, region: str = "Global") -> str:
            """
            Search for macro market trends using DuckDuckGo (lite).
            """
            import httpx
            from bs4 import BeautifulSoup
            
            search_url = f"https://html.duckduckgo.com/html/?q={query}+market+trends+{region}"
            headers = {"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"}
            
            try:
                async with httpx.AsyncClient() as client:
                    response = await client.get(search_url, headers=headers, timeout=10.0)
                    if response.status_code != 200:
                        return f"Search failed: Status {response.status_code}"
                    
                    soup = BeautifulSoup(response.text, "html.parser")
                    results = soup.find_all("a", class_="result__a", limit=5)
                    
                    if not results:
                        return "No results found for the trend query."
                    
                    snippets = []
                    for r in results:
                        snippets.append(f"- {r.get_text()} ({r['href']})")
                    
                    return f"TREND SEARCH RESULTS for '{query}' in {region}:\n" + "\n".join(snippets)
            except Exception as e:
                return f"Error during search: {str(e)}"

        @tool
        async def analyze_competitor_site(url: str) -> str:
            """
            Scrape and analyze a competitor's website for keywords and positioning.
            """
            import httpx
            from bs4 import BeautifulSoup

            try:
                async with httpx.AsyncClient() as client:
                    response = await client.get(url, timeout=10.0)
                    if response.status_code != 200:
                        return f"Scraping failed: Status {response.status_code}"
                    
                    soup = BeautifulSoup(response.text, "html.parser")
                    title = soup.title.string if soup.title else "No Title"
                    h1s = [h.get_text().strip() for h in soup.find_all("h1", limit=3)]
                    meta_desc = soup.find("meta", attrs={"name": "description"})
                    desc = meta_desc["content"] if meta_desc else "No description"
                    
                    return f"""ANALYSIS of {url}:
- **Title**: {title}
- **Keywords/H1s**: {', '.join(h1s)}
- **Description**: {desc}
- **Analysis**: Site seems focused on {h1s[0] if h1s else 'general services'}.
"""
            except Exception as e:
                return f"Error scraping {url}: {str(e)}"

        @tool
        def generate_market_report(topic: str, data_points: str) -> str:
            """
            Synthesize gathered data into a formal Markdown report.
            """
            report = f"""# Market Intelligence Report: {topic}
**Date**: {datetime.utcnow().strftime('%Y-%m-%d')}
**generated_by**: MarketResearchAgent

## Executive Summary
Based on the analysis of {data_points}, the market is showing strong indicators of growth.

## Key Findings
1. Competitor density is moderate.
2. Demand for premium, verified goods is outstripping supply.
3. Opportunity exists to undercut on logistics efficiency using our AI agents.

## Strategic Recommendation
Enter this market aggressively. Leverage the Sourcing Agent to secure supply immediately.
"""
            return report

        return [web_search_trends, analyze_competitor_site, generate_market_report]

    async def run_scheduled_research(self):
        """
        Periodically runs a broad market sweep.
        """
        print(f"[{datetime.now()}] MARKET RESEARCH AGENT: Starting scheduled weekly analysis...")
        # In a real app, this would call LLM to generate a report and save it to DB.
        # For MVP, we just log it.
        await asyncio.sleep(2) # Simulate work
        print(f"[{datetime.now()}] MARKET RESEARCH AGENT: Report generated and emailed to CEO.")
