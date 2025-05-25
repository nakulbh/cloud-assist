# Agent Code Structure

The agent code has been refactored into multiple modules for better organization:

## Files Overview

- **`main.py`** - Entry point for running the agent
- **`src/state.py`** - Contains the `AgentState` type definition
- **`src/configuration.py`** - Environment setup and LLM configuration
- **`src/prompts.py`** - System prompts and data structures for user interactions
- **`src/utils.py`** - Utility functions for command execution and user interaction
- **`src/graph.py`** - Main graph structure and node definitions

## Running the Agent

```bash
python main.py
```

## Module Details

### state.py
- `AgentState`: TypedDict defining the state structure for the agent

### configuration.py
- `load_environment()`: Loads environment variables from .env file
- `get_groq_llm()`: Initializes and returns the Groq LLM instance

### prompts.py
- `get_command_generation_prompt()`: System prompt for generating commands
- `get_retry_command_prompt()`: System prompt for generating alternative commands
- `get_approval_request_data()`: Data structure for human approval requests
- `get_retry_request_data()`: Data structure for retry confirmation requests

### utils.py
- `clean_command_output()`: Removes markdown formatting from generated commands
- `execute_shell_command()`: Executes shell commands safely with timeout
- `print_result()`: Formats and prints execution results
- `handle_user_interrupt()`: Manages user interaction during interrupts

### graph.py
- Contains all the graph nodes and the main graph creation logic
- `create_agent_graph()`: Creates and returns the complete agent graph
- `main()`: Main function that runs the interactive agent
