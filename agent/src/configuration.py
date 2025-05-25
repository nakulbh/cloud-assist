from dotenv import load_dotenv
import os
from langchain_groq import ChatGroq


def load_environment():
    """Load environment variables from .env file"""
    load_dotenv()


def get_groq_llm():
    """Initialize and return Groq LLM instance"""
    groq_api_key = os.getenv("GROQ_API_KEY")
    if not groq_api_key:
        raise ValueError("GROQ_API_KEY not found. Please set your Groq API key in the .env file.")
    
    return ChatGroq(
        model='llama-3.1-8b-instant',
        api_key=groq_api_key
    )