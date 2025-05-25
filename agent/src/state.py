from typing import TypedDict, Annotated, Sequence
from langchain_core.messages import BaseMessage
from operator import add as add_messages


class AgentState(TypedDict):
    """State definition for the human-in-the-loop agent"""
    messages: Annotated[Sequence[BaseMessage], add_messages]
    user_prompt: str
    generated_command: str
    command_output: str
    command_error: str
    execution_approved: bool
    retry_count: int