# LangGraph Studio Setup

This project is configured to work with LangGraph Studio for visual debugging and interaction.

## Prerequisites

1. **LangGraph Studio**: Download from [LangGraph Studio](https://github.com/langchain-ai/langgraph-studio)
2. **Docker**: Required for LangGraph Studio (Docker Compose v2.22.0+)
3. **Python 3.13+**: As specified in pyproject.toml

## Project Structure

```
agent/
├── src/                          # Source code
│   ├── __init__.py
│   ├── state.py                  # AgentState definition
│   ├── configuration.py          # Environment & LLM setup
│   ├── prompts.py               # System prompts
│   ├── utils.py                 # Utility functions
│   └── graph.py                 # Main graph definition (exports 'graph')
├── .env                         # Environment variables
├── langgraph.json              # LangGraph Studio configuration
├── pyproject.toml              # Python dependencies
└── main.py                     # CLI entry point
```

## Setup Instructions

### 1. Environment Configuration

Create a `.env` file with your API keys:

```bash
# Required for the agent
GROQ_API_KEY=your_groq_api_key_here

# Optional: Add other provider keys if you want to extend the agent
OPENAI_API_KEY=your_openai_key_here
ANTHROPIC_API_KEY=your_anthropic_key_here
```

### 2. Install Dependencies

Using uv (recommended):
```bash
uv sync
```

Or using pip:
```bash
pip install -e .
```

### 3. Test the Graph

Verify the graph loads correctly:
```bash
python -c "from src.graph import graph; print('Graph loaded successfully')"
```

### 4. Run with LangGraph Studio

1. Open LangGraph Studio
2. Open this project directory
3. The `langgraph.json` configuration will be automatically detected
4. The graph will be visualized and ready for interaction

## LangGraph Studio Configuration

The `langgraph.json` file configures:

- **Dependencies**: Points to `./src` directory
- **Graphs**: Exports `command_agent` from `./src/graph.py:graph`
- **Environment**: Uses `.env` file for environment variables

## Graph Features

This human-in-the-loop command agent includes:

- **Visual Flow**: See the decision flow in LangGraph Studio
- **Interactive Debugging**: Step through each node
- **State Inspection**: View agent state at each step
- **Human Approval**: Visual prompts for command execution approval
- **Retry Logic**: Visual representation of error handling and retries

## Studio Usage

1. **Graph Mode**: Full feature set with detailed execution views
2. **Chat Mode**: Simplified interface for testing (if MessagesState is used)

## Running Locally

For command-line usage without Studio:
```bash
python main.py
```

## Troubleshooting

### Graph Import Issues
- Ensure all dependencies are installed
- Check that `.env` file has `GROQ_API_KEY`
- Verify Python path includes project directory

### Studio Connection Issues
- Confirm Docker is running
- Check that port 8123 is available
- Verify `langgraph.json` syntax is valid

### Environment Issues
- Ensure `.env` file exists and has proper format
- Check that API keys are valid
- Verify Python version compatibility (3.13+)
