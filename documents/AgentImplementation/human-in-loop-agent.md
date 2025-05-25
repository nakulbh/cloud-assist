# Human-in-the-Loop Agent Implementation with Agno

## Overview

This document outlines the implementation of a human-in-the-loop agent for Cloud-Assist using the Agno library. The agent will be invoked by the CLI and provide DevOps assistance while requiring human approval for potentially risky operations.

## Architecture

The architecture of the human-in-the-loop agent system consists of the following components:

1. **Agent**: The core component that executes tasks and requests human approval when necessary.
2. **Human Supervisor**: A person who reviews the agent's requests and provides approval or denial.
3. **Communication Interface**: The means by which the agent and human supervisor communicate. This can be a chat interface, email, or any other messaging platform.
4. **Logging and Monitoring**: A system to log all actions taken by the agent and monitor its performance.

## Workflow

The workflow of the human-in-the-loop agent system is as follows:

1. The agent receives a task to perform.
2. The agent evaluates the task and determines if it requires human approval.
3. If human approval is required, the agent sends a request to the human supervisor through the communication interface.
4. The human supervisor reviews the request and either approves or denies it.
5. The agent logs the supervisor's decision and proceeds with the task execution accordingly.
6. The logging and monitoring system records all actions and decisions for accountability and auditing purposes.

## Components

1. **CLI Frontend (Go)**
   - User interface using Bubble Tea
   - Manages user interaction
   - Sends requests to the agent server
   - Displays responses and command suggestions

2. **Agent Server (Python/Agno)**
   - Implements the agent logic using Agno
   - Processes user requests
   - Provides command suggestions
   - Handles tool execution with human approval

3. **Communication Layer**
   - REST API for simple requests
   - WebSockets for streaming responses and interactive sessions

## Implementation with Agno

### Agent Server

The agent server will be implemented in Python using the Agno library, which provides a powerful framework for building agents with built-in human approval flows.

#### Core Components

1. **Tool Definitions**
   - DevOps command execution
   - File operations
   - Environment inspection
   - Cloud service interactions

2. **Agent Configuration**
   - Personality and instructions
   - Available tools
   - Approval hooks for human-in-the-loop validation

3. **Communication Endpoints**
   - REST API for stateless requests
   - WebSocket for stateful sessions

### Communication Protocols

#### REST API vs WebSocket

Both protocols have advantages for different use cases:

| Feature | REST API | WebSocket |
|---------|----------|-----------|
| Stateless | Yes | No |
| Streaming responses | No | Yes |
| Connection overhead | Higher | Lower after connection |
| Real-time updates | Polling required | Native support |
| Implementation complexity | Lower | Higher |
| Resource usage | Lower | Higher |

#### Recommendation

Implement a hybrid approach:

1. **REST API** for:
   - Initial authentication
   - Simple, one-off requests
   - Status checking
   - Agent configuration

2. **WebSocket** for:
   - Interactive chat sessions
   - Command execution with streaming output
   - Long-running operations
   - Real-time feedback

## Human-in-the-Loop Implementation

The human approval flow is a critical component of our agent architecture, ensuring safety and control.

### Approval Process

1. Agent suggests a command or action
2. CLI displays the suggestion with explanation
3. User approves, denies, or requests more information
4. If approved, action is executed
5. Results are streamed back to user
6. Agent analyzes results and suggests next steps

### Example Approval Flow

```
This is the code block that represents the suggested code change:
```markdown
// Add this to your existing chat.go implementation

// filepath: /home/nakulbh/Desktop/Projects/PersonalProjects/cloud-assist/cli/ui/websocket_client.go
package ui

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/gorilla/websocket"
)

// MessageType defines the types of messages exchanged with the agent server
type MessageType string

const (
    TypeMessage        MessageType = "message"
    TypeCommandApproval MessageType = "command_approval"
    TypeCommandOutput   MessageType = "command_output"
)

// ClientMessage represents messages sent from the client to the server
type ClientMessage struct {
    Type    MessageType `json:"type"`
    Content string      `json:"content,omitempty"`
    Approved bool       `json:"approved,omitempty"`
}

// ServerMessage represents messages received from the server
type ServerMessage struct {
    Type      MessageType `json:"type"`
    Content   string      `json:"content,omitempty"`
    Command   []string    `json:"command,omitempty"`
    Explanation string    `json:"explanation,omitempty"`
}

// WebSocketClient handles communication with the agent server
type WebSocketClient struct {
    conn           *websocket.Conn
    serverURL      string
    onMessage      func(string)
    onCommandSuggestion func([]string, string) bool
    onCommandOutput func(string)
    done           chan struct{}
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(serverURL string) *WebSocketClient {
    return &WebSocketClient{
        serverURL: serverURL,
        done:      make(chan struct{}),
    }
}

// Connect establishes a connection to the WebSocket server
func (c *WebSocketClient) Connect() error {
    var err error
    c.conn, _, err = websocket.DefaultDialer.Dial(c.serverURL, nil)
    if err != nil {
        return fmt.Errorf("websocket connection failed: %w", err)
    }

    // Start listening for messages
    go c.readPump()
    
    return nil
}

// SendMessage sends a message to the agent
func (c *WebSocketClient) SendMessage(content string) error {
    message := ClientMessage{
        Type:    TypeMessage,
        Content: content,
    }
    
    return c.conn.WriteJSON(message)
}

// SendApproval sends an approval response for a command
func (c *WebSocketClient) SendApproval(approved bool) error {
    message := ClientMessage{
        Type:     TypeMessage,
        Approved: approved,
    }
    
    return c.conn.WriteJSON(message)
}

// SetMessageHandler sets the callback for received messages
func (c *WebSocketClient) SetMessageHandler(handler func(string)) {
    c.onMessage = handler
}

// SetCommandSuggestionHandler sets the callback for command suggestions
func (c *WebSocketClient) SetCommandSuggestionHandler(handler func([]string, string) bool) {
    c.onCommandSuggestion = handler
}

// SetCommandOutputHandler sets the callback for command output
func (c *WebSocketClient) SetCommandOutputHandler(handler func(string)) {
    c.onCommandOutput = handler
}

// Close closes the WebSocket connection
func (c *WebSocketClient) Close() {
    close(c.done)
    if c.conn != nil {
        // Send close message
        err := c.conn.WriteMessage(
            websocket.CloseMessage, 
            websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
        )
        if err != nil {
            log.Println("Error during connection close:", err)
        }
        
        // Close connection
        c.conn.Close()
    }
}

// readPump handles incoming messages
func (c *WebSocketClient) readPump() {
    defer func() {
        c.conn.Close()
    }()
    
    for {
        select {
        case <-c.done:
            return
        default:
            // Read message
            _, message, err := c.conn.ReadMessage()
            if err != nil {
                if websocket.IsUnexpectedCloseError(
                    err, 
                    websocket.CloseGoingAway, 
                    websocket.CloseAbnormalClosure,
                ) {
                    log.Printf("websocket error: %v", err)
                }
                return
            }
            
            // Parse message
            var serverMsg ServerMessage
            if err := json.Unmarshal(message, &serverMsg); err != nil {
                log.Printf("Error parsing message: %v", err)
                continue
            }
            
            // Handle message based on type
            switch serverMsg.Type {
            case TypeMessage:
                if c.onMessage != nil {
                    c.onMessage(serverMsg.Content)
                }
            case TypeCommandApproval:
                if c.onCommandSuggestion != nil {
                    approved := c.onCommandSuggestion(serverMsg.Command, serverMsg.Explanation)
                    c.SendApproval(approved)
                }
            case TypeCommandOutput:
                if c.onCommandOutput != nil {
                    c.onCommandOutput(serverMsg.Content)
                }
            }
        }
    }
}

// Integration example - how to use the WebSocket client in your chat model
func (m *ChatModel) initializeWebSocketClient() {
    // Create and connect the WebSocket client
    client := NewWebSocketClient("ws://localhost:8000/ws/chat")
    
    // Set message handler
    client.SetMessageHandler(func(content string) {
        m.messages = append(m.messages, message{
            content: content,
            msgType: agentMessage,
        })
        m.updateViewport()
    })
    
    // Set command suggestion handler
    client.SetCommandSuggestionHandler(func(command []string, explanation string) bool {
        // Display command suggestion to user
        cmdStr := fmt.Sprintf("%s", command)
        m.messages = append(m.messages, message{
            content: cmdStr,
            msgType: commandSuggestion,
        })
        m.messages = append(m.messages, message{
            content: explanation,
            msgType: agentMessage,
        })
        m.currentCommand = cmdStr
        m.suggestionMode = true
        m.showInput = false
        m.updateViewport()
        
        // Wait for user response (this would need to be handled in your actual implementation)
        // This is just a placeholder - in real implementation you would need to manage state machine
        return true
    })
    
    // Set command output handler
    client.SetCommandOutputHandler(func(output string) {
        m.messages = append(m.messages, message{
            content: output,
            msgType: commandOutput,
        })
        m.updateViewport()
    })
    
    // Connect to server
    if err := client.Connect(); err != nil {
        m.messages = append(m.messages, message{
            content: fmt.Sprintf("Error connecting to agent server: %v", err),
            msgType: errorMessage,
        })
        m.updateViewport()
        return
    }
    
    // Store client for later use
    m.webSocketClient = client
}
```
````
This is the code block that represents the suggested code change:
````markdown
## Implementation Examples

### 1. Agno-based Agent Server

```python
from agno.agent import Agent
from agno.models.openai import OpenAI
from agno.tools import FunctionCall, tool
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
import os
import json
import asyncio
from typing import List, Dict, Any, Iterator

# Initialize FastAPI app
app = FastAPI(title="Cloud-Assist Agent Server")

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Pre-execution hook for human approval
def pre_hook(fc: FunctionCall, ws=None):
    """Request human approval before executing command"""
    if not ws:
        # Default behavior for non-websocket contexts
        print(f"About to run {fc.function.name}")
        response = input("Do you want to continue? (y/n): ").strip().lower()
        if response != "y":
            raise Exception("Command execution cancelled by user")
        return
    
    # WebSocket implementation would send approval request to client
    # and wait for response - to be implemented in the WebSocket handler

# Define DevOps tools with human approval
@tool(pre_hook=pre_hook)
def execute_command(command: List[str], cwd: str = None) -> Iterator[str]:
    """Execute a shell command with user approval.
    
    Args:
        command (List[str]): Command to execute as a list of arguments
        cwd (str, optional): Working directory
        
    Returns:
        Iterator[str]: Command output lines
    """
    import subprocess
    
    # Execute command in a safe environment
    process = subprocess.Popen(
        command,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        cwd=cwd
    )
    
    # Stream output
    for line in process.stdout:
        yield line.rstrip()
    
    # Wait for process to complete and get return code
    return_code = process.wait()
    if return_code != 0:
        yield f"Command failed with exit code {return_code}"

# Create agent with appropriate tools and instructions
agent = Agent(
    model=OpenAI(),
    description="Cloud-Assist DevOps Agent",
    instructions="""
    You are an AI-powered DevOps assistant called Cloud-Assist.
    
    Your responsibilities:
    - Help users manage their infrastructure
    - Suggest commands to solve problems
    - Provide clear explanations of what commands do
    - Always get confirmation before executing commands
    
    Safety guidelines:
    - Never run destructive commands without explicit warnings
    - Explain potential risks of suggested commands
    - Start with read-only operations when diagnosing issues
    - Verify success after each command execution
    """,
    tools=[execute_command],
    show_tool_calls=True,
    markdown=True,
)

# REST API endpoint for one-off requests
@app.post("/api/query")
async def query(request: Dict[str, Any]):
    """Process a single query and return response"""
    try:
        response = agent.run(request["message"])
        return {"response": response}
    except Exception as e:
        return {"error": str(e)}

# WebSocket connection manager
class ConnectionManager:
    def __init__(self):
        self.active_connections: List[WebSocket] = []

    async def connect(self, websocket: WebSocket):
        await websocket.accept()
        self.active_connections.append(websocket)

    def disconnect(self, websocket: WebSocket):
        self.active_connections.remove(websocket)

    async def send_response(self, websocket: WebSocket, message: str):
        await websocket.send_text(json.dumps({"type": "message", "content": message}))
    
    async def send_command_approval(self, websocket: WebSocket, command: List[str], explanation: str):
        await websocket.send_text(json.dumps({
            "type": "command_approval", 
            "command": command,
            "explanation": explanation
        }))
        
    async def send_command_output(self, websocket: WebSocket, output: str):
        await websocket.send_text(json.dumps({
            "type": "command_output", 
            "content": output
        }))

manager = ConnectionManager()

# WebSocket endpoint for interactive sessions
@app.websocket("/ws/chat")
async def websocket_endpoint(websocket: WebSocket):
    await manager.connect(websocket)
    try:
        while True:
            # Receive message from client
            data = await websocket.receive_text()
            message_data = json.loads(data)
            
            # Handle different message types
            if message_data["type"] == "message":
                # Set up custom hooks for this websocket session
                async def ws_pre_hook(fc: FunctionCall):
                    command = fc.args.get("command", [])
                    await manager.send_command_approval(
                        websocket, 
                        command,
                        f"About to execute: {' '.join(command)}"
                    )
                    # Wait for approval
                    approval = await websocket.receive_text()
                    approval_data = json.loads(approval)
                    if not approval_data.get("approved", False):
                        raise Exception("Command execution cancelled by user")
                
                # Create a session-specific agent with the websocket-enabled pre_hook
                session_agent = Agent(
                    model=OpenAI(),
                    description="Cloud-Assist DevOps Agent",
                    instructions=agent.instructions,
                    tools=[execute_command],
                    show_tool_calls=True,
                    markdown=True,
                )
                
                # Process message with streaming response
                async for chunk in session_agent.astream(message_data["content"]):
                    await manager.send_response(websocket, chunk)
            
            elif message_data["type"] == "approval":
                # This is handled within the ws_pre_hook function
                # Just a placeholder for the protocol
                pass
    
    except WebSocketDisconnect:
        manager.disconnect(websocket)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
````

