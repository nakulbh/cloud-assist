from dotenv import load_dotenv
import os
import subprocess
import json
from langgraph.graph import StateGraph, END
from langgraph.checkpoint.memory import MemorySaver
from langgraph.types import interrupt, Command
from typing import TypedDict, Annotated, Sequence, Literal
from langchain_core.messages import BaseMessage, SystemMessage, HumanMessage, ToolMessage
from operator import add as add_messages
from langchain_core.tools import tool
from langchain_groq import ChatGroq


load_dotenv()

# Initialize Groq LLM
groq_api_key = os.getenv("GROQ_API_KEY")
if not groq_api_key:
    raise ValueError("GROQ_API_KEY not found. Please set your Groq API key in the .env file.")

llm = ChatGroq(
    model='llama-3.1-8b-instant',
    api_key=groq_api_key
)

# State definition
class AgentState(TypedDict):
    messages: Annotated[Sequence[BaseMessage], add_messages]
    user_prompt: str
    generated_command: str
    command_output: str
    command_error: str
    execution_approved: bool
    retry_count: int

def generate_command_node(state: AgentState):
    """Generate a command based on the user prompt"""
    user_prompt = state["user_prompt"]
    
    system_message = SystemMessage(content="""
You are a command-line assistant. Your job is to generate shell commands based on user requests.
Generate ONLY the command, no explanations or additional text.
The command should be safe and appropriate for the user's request.

Examples:
- User: "show all running docker containers" -> Response: "docker ps -a"
- User: "list all files in current directory" -> Response: "ls -la"
- User: "show system disk usage" -> Response: "df -h"

Generate a single, safe command for the following request:
""")
    
    human_message = HumanMessage(content=user_prompt)
    
    response = llm.invoke([system_message, human_message])
    generated_command = response.content.strip()
    
    # Remove any markdown code blocks if present
    if generated_command.startswith("```"):
        lines = generated_command.split('\n')
        generated_command = lines[1] if len(lines) > 1 else generated_command
    if generated_command.endswith("```"):
        generated_command = generated_command[:-3].strip()
    
    return {
        "generated_command": generated_command,
        "messages": [system_message, human_message, response],
        "retry_count": state.get("retry_count", 0)
    }

def human_approval_node(state: AgentState) -> Command[Literal["execute_command", "generate_command", "__end__"]]:
    """Ask human for approval before executing the command"""
    generated_command = state["generated_command"]
    retry_count = state.get("retry_count", 0)
    
    approval_data = {
        "question": "Do you want to execute this command?",
        "command": generated_command,
        "user_prompt": state["user_prompt"],
        "retry_count": retry_count,
        "options": {
            "approve": "Execute the command",
            "reject": "Generate a different command", 
            "cancel": "Cancel the operation"
        }
    }
    
    # Interrupt and wait for human approval
    human_response = interrupt(approval_data)
    
    if human_response == "approve":
        return Command(goto="execute_command", update={"execution_approved": True})
    elif human_response == "reject":
        return Command(goto="generate_command", update={"execution_approved": False})
    else:  # cancel
        return Command(goto="__end__")

def execute_command_node(state: AgentState):
    """Execute the approved command"""
    command = state["generated_command"]
    
    try:
        # Execute the command
        result = subprocess.run(
            command, 
            shell=True, 
            capture_output=True, 
            text=True, 
            timeout=30  # 30 second timeout
        )
        
        if result.returncode == 0:
            # Command executed successfully
            return {
                "command_output": result.stdout,
                "command_error": "",
                "execution_approved": True
            }
        else:
            # Command failed
            return {
                "command_output": result.stdout,
                "command_error": result.stderr,
                "execution_approved": True
            }
            
    except subprocess.TimeoutExpired:
        return {
            "command_output": "",
            "command_error": "Command timed out after 30 seconds",
            "execution_approved": True
        }
    except Exception as e:
        return {
            "command_output": "",
            "command_error": f"Error executing command: {str(e)}",
            "execution_approved": True
        }

def check_result_node(state: AgentState) -> Command[Literal["__end__", "retry_command"]]:
    """Check if command executed successfully or needs retry"""
    command_error = state.get("command_error", "")
    command_output = state.get("command_output", "")
    retry_count = state.get("retry_count", 0)
    
    if command_error and retry_count < 3:  # Max 3 retries
        # Command failed, ask if user wants to retry with a different approach
        retry_data = {
            "question": "Command failed. Do you want me to try a different approach?",
            "original_command": state["generated_command"],
            "error": command_error,
            "output": command_output,
            "retry_count": retry_count,
            "options": {
                "retry": "Yes, try a different command",
                "stop": "No, stop here"
            }
        }
        
        human_response = interrupt(retry_data)
        
        if human_response == "retry":
            return Command(
                goto="retry_command", 
                update={"retry_count": retry_count + 1}
            )
        else:
            return Command(goto="__end__")
    else:
        # Success or max retries reached
        return Command(goto="__end__")

def retry_command_node(state: AgentState):
    """Generate a new command based on the previous error"""
    user_prompt = state["user_prompt"]
    previous_command = state["generated_command"]
    error = state["command_error"]
    retry_count = state["retry_count"]
    
    system_message = SystemMessage(content=f"""
You are a command-line assistant. The previous command failed with an error.
Generate a different, alternative command to accomplish the same goal.

Original request: {user_prompt}
Previous command that failed: {previous_command}
Error encountered: {error}
Retry attempt: {retry_count}/3

Generate ONLY a single alternative command, no explanations or additional text.
Make sure the new command is different from the previous one and addresses the error.
""")
    
    human_message = HumanMessage(content=f"Generate alternative command for: {user_prompt}")
    
    response = llm.invoke([system_message, human_message])
    generated_command = response.content.strip()
    
    # Remove any markdown code blocks if present
    if generated_command.startswith("```"):
        lines = generated_command.split('\n')
        generated_command = lines[1] if len(lines) > 1 else generated_command
    if generated_command.endswith("```"):
        generated_command = generated_command[:-3].strip()
    
    return {
        "generated_command": generated_command
    }

def create_agent_graph():
    """Create the agent graph with human-in-the-loop workflow"""
    
    # Create the graph
    workflow = StateGraph(AgentState)
    
    # Add nodes
    workflow.add_node("generate_command", generate_command_node)
    workflow.add_node("human_approval", human_approval_node)
    workflow.add_node("execute_command", execute_command_node)
    workflow.add_node("check_result", check_result_node)
    workflow.add_node("retry_command", retry_command_node)
    
    # Set entry point
    workflow.set_entry_point("generate_command")
    
    # Add edges
    workflow.add_edge("generate_command", "human_approval")
    workflow.add_edge("execute_command", "check_result")
    workflow.add_edge("retry_command", "human_approval")
    
    # Add conditional edges
    # human_approval can go to execute_command, generate_command, or end
    # check_result can go to end or retry_command
    
    # Compile with memory saver for checkpointing
    memory = MemorySaver()
    app = workflow.compile(checkpointer=memory)
    
    return app

def print_result(result, state):
    """Print the execution result in a formatted way"""
    print("\n" + "="*50)
    print("EXECUTION RESULT")
    print("="*50)
    
    if state.get("command_output"):
        print(f"Command: {state['generated_command']}")
        print(f"Output:\n{state['command_output']}")
        
    if state.get("command_error"):
        print(f"Error: {state['command_error']}")
        
    print("="*50)

def main():
    """Main function to run the human-in-the-loop agent"""
    print("Human-in-the-Loop Command Agent")
    print("================================")
    print("Enter your command request (e.g., 'show all running docker containers')")
    print("Type 'quit' to exit\n")
    
    app = create_agent_graph()
    
    while True:
        user_input = input("Your request: ").strip()
        
        if user_input.lower() in ['quit', 'exit', 'q']:
            print("Goodbye!")
            break
            
        if not user_input:
            continue
            
        # Create initial state
        initial_state = {
            "user_prompt": user_input,
            "messages": [],
            "generated_command": "",
            "command_output": "",
            "command_error": "",
            "execution_approved": False,
            "retry_count": 0
        }
        
        # Configuration for the graph execution
        config = {"configurable": {"thread_id": f"session_{hash(user_input)}"}}
        
        try:
            # Run the graph until it completes or hits an interrupt
            result = app.invoke(initial_state, config=config)
            
            # Check if we hit an interrupt
            while result.get("__interrupt__"):
                print("\n" + "-"*50)
                interrupts = result["__interrupt__"]
                
                for interrupt_obj in interrupts:
                    interrupt_value = interrupt_obj.value
                    
                    # Handle different types of interrupts
                    if "question" in interrupt_value:
                        print(f"\n{interrupt_value['question']}")
                        
                        if "command" in interrupt_value:
                            print(f"Generated command: {interrupt_value['command']}")
                            
                        if "error" in interrupt_value:
                            print(f"Error: {interrupt_value['error']}")
                            
                        if "options" in interrupt_value:
                            print("\nOptions:")
                            for key, desc in interrupt_value["options"].items():
                                print(f"  {key}: {desc}")
                            
                        # Get user response
                        while True:
                            if "options" in interrupt_value:
                                valid_options = list(interrupt_value["options"].keys())
                                user_response = input(f"\nYour choice ({'/'.join(valid_options)}): ").strip().lower()
                                if user_response in valid_options:
                                    break
                                else:
                                    print(f"Please enter one of: {', '.join(valid_options)}")
                            else:
                                user_response = input("\nYour response: ").strip()
                                break
                        
                        # Resume the graph with the user's response
                        result = app.invoke(Command(resume=user_response), config=config)
                        break
            
            # Print final result
            final_state = app.get_state(config).values
            print_result(result, final_state)
            
        except KeyboardInterrupt:
            print("\n\nOperation cancelled by user.")
        except Exception as e:
            print(f"\nError: {str(e)}")
        
        print()  # Add spacing between requests


if __name__ == "__main__":
    main()
