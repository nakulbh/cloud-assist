from langgraph.graph import StateGraph
from langgraph.checkpoint.memory import MemorySaver
from langgraph.types import interrupt, Command
from typing import Literal
from langchain_core.messages import HumanMessage

# Import from our modules
from .state import AgentState
from .configuration import load_environment, get_groq_llm
from .prompts import (
    get_command_generation_prompt, 
    get_retry_command_prompt,
    get_approval_request_data,
    get_retry_request_data
)
from .utils import (
    clean_command_output,
    execute_shell_command,
    print_result,
    handle_user_interrupt
)

# Initialize environment and LLM
load_environment()
llm = get_groq_llm()

def generate_command_node(state: AgentState):
    """Generate a command based on the user prompt"""
    user_prompt = state["user_prompt"]
    
    system_message = get_command_generation_prompt()
    human_message = HumanMessage(content=user_prompt)
    
    response = llm.invoke([system_message, human_message])
    generated_command = clean_command_output(response.content)
    
    return {
        "generated_command": generated_command,
        "messages": [system_message, human_message, response],
        "retry_count": state.get("retry_count", 0)
    }

def human_approval_node(state: AgentState) -> Command[Literal["execute_command", "generate_command", "__end__"]]:
    """Ask human for approval before executing the command"""
    generated_command = state["generated_command"]
    retry_count = state.get("retry_count", 0)
    
    approval_data = get_approval_request_data(
        generated_command, 
        state["user_prompt"], 
        retry_count
    )
    
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
    return execute_shell_command(command)

def check_result_node(state: AgentState) -> Command[Literal["__end__", "retry_command"]]:
    """Check if command executed successfully or needs retry"""
    command_error = state.get("command_error", "")
    command_output = state.get("command_output", "")
    retry_count = state.get("retry_count", 0)
    
    if command_error and retry_count < 3:  # Max 3 retries
        # Command failed, ask if user wants to retry with a different approach
        retry_data = get_retry_request_data(
            state["generated_command"],
            command_error,
            command_output,
            retry_count
        )
        
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
    
    system_message = get_retry_command_prompt(
        user_prompt, 
        previous_command, 
        error, 
        retry_count
    )
    human_message = HumanMessage(content=f"Generate alternative command for: {user_prompt}")
    
    response = llm.invoke([system_message, human_message])
    generated_command = clean_command_output(response.content)
    
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
    graph = workflow.compile(checkpointer=memory)
    
    return graph

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
            
            # Handle interrupts
            result = handle_user_interrupt(result, config, app)
            
            # Print final result
            final_state = app.get_state(config).values
            print_result(result, final_state)
            
        except KeyboardInterrupt:
            print("\n\nOperation cancelled by user.")
        except Exception as e:
            print(f"\nError: {str(e)}")
        
        print()  # Add spacing between requests


# Create the graph instance for LangGraph Studio
graph = create_agent_graph()

if __name__ == "__main__":
    main()
