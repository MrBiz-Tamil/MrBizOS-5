"""
Simplified BaseAgent using Google Generative AI SDK directly.
No LangChain dependency - Python 3.12 compatible.
"""
import google.generativeai as genai
from typing import List, Dict, Any, Optional
import os
import json

class BaseAgent:
    """Base class for all AI agents using Google Generative AI"""
    
    def __init__(self, name: str, role: str, model: str = "gemini-pro"):
        self.name = name
        self.role = role
        self.model_name = model
        
        # Configure Google AI
        api_key = os.getenv("GOOGLE_API_KEY")
        if api_key:
            genai.configure(api_key=api_key)
        
        # Initialize model
        try:
            self.model = genai.GenerativeModel(model)
        except AttributeError:
            # Fallback for older versions or issues
            print("Warning: genai.GenerativeModel not found, using basic fallback")
            self.model = None
        except Exception as e:
            print(f"Warning: Failed to initialize model: {e}")
            self.model = None
        
        # System prompt
        self.system_prompt = f"""You are {name}, a {role}.
Your role is to help with export/import business operations.
Provide concise, actionable responses."""
    
    async def chat(self, message: str, context: Optional[Dict[str, Any]] = None) -> str:
        """
        Send a message to the agent and get a response.
        
        Args:
            message: User message
            context: Optional context dictionary
            
        Returns:
            Agent's response as string
        """
        try:
            # Build full prompt with context
            full_prompt = self.system_prompt + "\n\n"
            
            if context:
                full_prompt += f"Context: {json.dumps(context, indent=2)}\n\n"
            
            full_prompt += f"User: {message}\n\nAssistant:"
            
            # Generate response
            if self.model:
                response = self.model.generate_content(full_prompt)
                return response.text
            else:
                return "AI Model not initialized. Please check configuration."
            
        except Exception as e:
            return f"Error: {str(e)}"
    
    async def execute_task(self, task: str, params: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """
        Execute a specific task.
        
        Args:
            task: Task name
            params: Task parameters
            
        Returns:
            Task result dictionary
        """
        try:
            # Build task prompt
            prompt = f"{self.system_prompt}\n\n"
            prompt += f"Task: {task}\n"
            
            if params:
                prompt += f"Parameters: {json.dumps(params, indent=2)}\n"
            
            prompt += "\nProvide a structured response with actionable steps."
            
            # Generate response
            if self.model:
                response = self.model.generate_content(prompt)
                result_text = response.text
            else:
                result_text = "AI Model not initialized. Task execution mocked."
            
            return {
                "success": True,
                "task": task,
                "result": result_text,
                "agent": self.name
            }
            
        except Exception as e:
            return {
                "success": False,
                "task": task,
                "error": str(e),
                "agent": self.name
            }
