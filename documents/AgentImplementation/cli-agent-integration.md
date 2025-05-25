# CLI to Agent Integration

## Overview

This document outlines the implementation of the CLI client that connects to the Agno-based agent server. The CLI will act as the client interface where users can submit requests, approve or deny suggested commands, and view the results in real-time.

## Architecture

## Implementation Requirements

### CLI Side (Go)

1. WebSocket client to connect to the agent server
2. UI components to display agent responses and command suggestions
3. Approval flow for suggested commands
4. Context management to maintain conversation state

### Agent Server Side (Python)

1. WebSocket server to handle client connections
2. Agno-based agent implementation with tool definitions
3. Human-in-the-loop approval hooks
4. Output streaming to the CLI client

## Implementation Plan

### 1. Files to Be Created/Modified on CLI Side

#### A. Create WebSocket Client

Create a new file for WebSocket communication to connect the CLI with the agent server.

#### B. Modify Chat UI

Update the existing chat UI to integrate with the agent client and handle command approvals.

#### C. Create Agent Client

Create a client to handle communication with the agent server.

### 2. Files to Be Created on Agent Server Side

#### A. Create Agent Server

Implement the Agno-based agent server with WebSocket support.

#### B. Create Tool Definitions

Define the DevOps tools that the agent can use with human approval.

## Code Implementation

### CLI Side Implementation

#### 1. WebSocket Client Implementation

```go
// filepath: /home/nakulbh/Desktop/Projects/PersonalProjects/cloud-assist/cli/client/websocket_client.go
package client

import (
    "encoding/json"
    "fmt"
    "log"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

// MessageType defines the types of messages exchanged with the agent server
type MessageType string

const (
    TypeMessage         MessageType = "message"
    TypeCommandApproval MessageType = "command_approval"
    TypeCommandOutput   MessageType = "command_output"
    TypeError           MessageType = "error"
)

// ClientMessage represents messages sent from the client to the server
type ClientMessage struct {
    Type     MessageType `json:"type"`
    Content  string      `json:"content,omitempty"`
    Approved bool        `json:"approved,omitempty"`
}

// ServerMessage represents messages received from the server
type ServerMessage struct {
    Type        MessageType `json:"type"`
    Content     string      `json:"content,omitempty"`
    Command     []string    `json:"command,omitempty"`
    Explanation string      `json:"explanation,omitempty"`
    Error       string      `json:"error,omitempty"`
}

// MessageHandler defines the callback functions for different message types
type MessageHandler struct {
    OnTextMessage      func(content string)
    OnCommandApproval  func(command []string, explanation string) bool
    OnCommandOutput    func(content string)
    OnError            func(errorMsg string)
    OnConnectionChange func(connected bool)
}

// WebSocketClient handles communication with the agent server
type WebSocketClient struct {
    conn            *websocket.Conn
    serverURL       string
    handler         MessageHandler
    done            chan struct{}
    connected       bool
    connectionMutex sync.Mutex
    sendMutex       sync.Mutex
    reconnectDelay  time.Duration
    maxReconnects   int
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(serverURL string, handler MessageHandler) *WebSocketClient {
    return &WebSocketClient{
        serverURL:      serverURL,
        handler:        handler,
        done:           make(chan struct{}),
        reconnectDelay: 2 * time.Second,
        maxReconnects:  5,
    }
}

// Connect establishes a connection to the WebSocket server
func (c *WebSocketClient) Connect() error {
    c.connectionMutex.Lock()
    defer c.connectionMutex.Unlock()

    if c.connected {
        return nil
    }

    var err error
    c.conn, _, err = websocket.DefaultDialer.Dial(c.serverURL, nil)
    if err != nil {
        return fmt.Errorf("websocket connection failed: %w", err)
    }

    c.connected = true
    if c.handler.OnConnectionChange != nil {
        c.handler.OnConnectionChange(true)
    }

    // Start listening for messages
    go c.readPump()
    return nil
}

// SendMessage sends a message to the agent
func (c *WebSocketClient) SendMessage(content string) error {
    c.connectionMutex.Lock()
    if !c.connected {
        c.connectionMutex.Unlock()
        return fmt.Errorf("client not connected")
    }
    c.connectionMutex.Unlock()

    c.sendMutex.Lock()
    defer c.sendMutex.Unlock()

    message := ClientMessage{
        Type:    TypeMessage,
        Content: content,
    }

    return c.conn.WriteJSON(message)
}

// SendApproval sends an approval response for a command
func (c *WebSocketClient) SendApproval(approved bool) error {
    c.connectionMutex.Lock()
    if !c.connected {
        c.connectionMutex.Unlock()
        return fmt.Errorf("client not connected")
    }
    c.connectionMutex.Unlock()

    c.sendMutex.Lock()
    defer c.sendMutex.Unlock()

    message := ClientMessage{
        Type:     TypeCommandApproval,
        Approved: approved,
    }

    return c.conn.WriteJSON(message)
}

// Close closes the WebSocket connection
func (c *WebSocketClient) Close() {
    close(c.done)
    
    c.connectionMutex.Lock()
    defer c.connectionMutex.Unlock()
    
    if !c.connected || c.conn == nil {
        return
    }

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
    c.connected = false
    
    if c.handler.OnConnectionChange != nil {
        c.handler.OnConnectionChange(false)
    }
}

// IsConnected returns the connection status
func (c *WebSocketClient) IsConnected() bool {
    c.connectionMutex.Lock()
    defer c.connectionMutex.Unlock()
    return c.connected
}

// readPump handles incoming messages
func (c *WebSocketClient) readPump() {
    defer func() {
        c.connectionMutex.Lock()
        if c.conn != nil {
            c.conn.Close()
        }
        c.connected = false
        c.connectionMutex.Unlock()
        
        if c.handler.OnConnectionChange != nil {
            c.handler.OnConnectionChange(false)
        }
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
                // Consider auto-reconnect here
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
                if c.handler.OnTextMessage != nil {
                    c.handler.OnTextMessage(serverMsg.Content)
                }
            case TypeCommandApproval:
                if c.handler.OnCommandApproval != nil {
                    approved := c.handler.OnCommandApproval(serverMsg.Command, serverMsg.Explanation)
                    c.SendApproval(approved)
                }
            case TypeCommandOutput:
                if c.handler.OnCommandOutput != nil {
                    c.handler.OnCommandOutput(serverMsg.Content)
                }
            case TypeError:
                if c.handler.OnError != nil {
                    c.handler.OnError(serverMsg.Error)
                }
            }
        }
    }
}
```

#### 2. Agent Client Implementation

```go
// filepath: /home/nakulbh/Desktop/Projects/PersonalProjects/cloud-assist/cli/client/agent_client.go
package client

import (
    "fmt"
    "log"
    "sync"
)

// ResponseCallback defines the function signature for handling agent responses
type ResponseCallback func(messageType MessageType, content string, commandData []string, explanation string)

// AgentClient manages the connection to the agent server and handles responses
type AgentClient struct {
    wsClient         *WebSocketClient
    serverURL        string
    responseCallback ResponseCallback
    pendingApproval  bool
    approvalMutex    sync.Mutex
    commandQueue     [][]string
    queueMutex       sync.Mutex
}

// NewAgentClient creates a new agent client
func NewAgentClient(serverURL string, callback ResponseCallback) *AgentClient {
    client := &AgentClient{
        serverURL:        serverURL,
        responseCallback: callback,
        commandQueue:     make([][]string, 0),
    }

    // Create WebSocket client with appropriate handlers
    client.wsClient = NewWebSocketClient(serverURL, MessageHandler{
        OnTextMessage: func(content string) {
            if callback != nil {
                callback(TypeMessage, content, nil, "")
            }
        },
        OnCommandApproval: func(command []string, explanation string) bool {
            client.approvalMutex.Lock()
            client.pendingApproval = true
            client.approvalMutex.Unlock()

            // Queue the command
            client.queueMutex.Lock()
            client.commandQueue = append(client.commandQueue, command)
            client.queueMutex.Unlock()

            // Notify via callback but don't auto-approve
            if callback != nil {
                callback(TypeCommandApproval, "", command, explanation)
            }

            // Return false - we'll handle actual approval separately
            return false
        },
        OnCommandOutput: func(content string) {
            if callback != nil {
                callback(TypeCommandOutput, content, nil, "")
            }
        },
        OnError: func(errorMsg string) {
            if callback != nil {
                callback(TypeError, errorMsg, nil, "")
            }
        },
        OnConnectionChange: func(connected bool) {
            status := "connected"
            if !connected {
                status = "disconnected"
            }
            log.Printf("Agent connection status: %s", status)
        },
    })

    return client
}

// Connect establishes a connection to the agent server
func (c *AgentClient) Connect() error {
    return c.wsClient.Connect()
}

// SendMessage sends a message to the agent
func (c *AgentClient) SendMessage(message string) error {
    return c.wsClient.SendMessage(message)
}

// ApproveCommand approves the pending command
func (c *AgentClient) ApproveCommand() error {
    c.approvalMutex.Lock()
    defer c.approvalMutex.Unlock()

    if !c.pendingApproval {
        return fmt.Errorf("no pending command to approve")
    }

    c.pendingApproval = false
    return c.wsClient.SendApproval(true)
}

// DenyCommand denies the pending command
func (c *AgentClient) DenyCommand() error {
    c.approvalMutex.Lock()
    defer c.approvalMutex.Unlock()

    if !c.pendingApproval {
        return fmt.Errorf("no pending command to deny")
    }

    c.pendingApproval = false
    return c.wsClient.SendApproval(false)
}

// HasPendingApproval checks if there's a command waiting for approval
func (c *AgentClient) HasPendingApproval() bool {
    c.approvalMutex.Lock()
    defer c.approvalMutex.Unlock()
    return c.pendingApproval
}

// GetNextCommand gets the next command from the queue
func (c *AgentClient) GetNextCommand() []string {
    c.queueMutex.Lock()
    defer c.queueMutex.Unlock()

    if len(c.commandQueue) == 0 {
        return nil
    }

    command := c.commandQueue[0]
    c.commandQueue = c.commandQueue[1:]
    return command
}

// Close closes the connection to the agent server
func (c *AgentClient) Close() {
    c.wsClient.Close()
}
```

#### 3. Chat UI Modification

```go
// filepath: /home/nakulbh/Desktop/Projects/PersonalProjects/cloud-assist/cli/ui/chat.go
package ui

import (
    "cloud-assist/cli/client"
    "fmt"
    "strings"
    // ... existing imports ...
)

// Additional properties to add to ChatModel
type ChatModel struct {
    // ... existing fields ...
    agentClient       *client.AgentClient
    useRemoteAgent    bool
    agentURL          string
    pendingApproval   bool
    currentCommand    string
    currentExplanation string
}

// NewChatModel creates a new chat model
func NewChatModel(width, height int) ChatModel {
    // ... existing initialization code ...
    
    // Configure for agent client
    model.useRemoteAgent = true  // Enable by default, could be configurable
    model.agentURL = "ws://localhost:8000/ws/chat"  // Default URL, could be configurable
    
    // Initialize the agent if enabled
    if model.useRemoteAgent {
        model.initializeAgentClient()
    } else {
        // Initialize the mock agent service (existing code)
        model.agentService = mock.NewAgentService()
        // ... existing initialization ...
    }

    // ... rest of the existing initialization ...
    
    return model
}

// New method to initialize the agent client
func (m *ChatModel) initializeAgentClient() {
    m.agentClient = client.NewAgentClient(m.agentURL, func(msgType client.MessageType, content string, command []string, explanation string) {
        switch msgType {
        case client.TypeMessage:
            m.messages = append(m.messages, message{
                content: content,
                msgType: agentMessage,
            })
        case client.TypeCommandApproval:
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
            m.currentExplanation = explanation
            m.suggestionMode = true
            m.pendingApproval = true
            m.showInput = false
        case client.TypeCommandOutput:
            m.messages = append(m.messages, message{
                content: content,
                msgType: commandOutput,
            })
        case client.TypeError:
            m.messages = append(m.messages, message{
                content: content,
                msgType: errorMessage,
            })
        }
        m.updateViewport()
    })

    // Connect to the agent server
    err := m.agentClient.Connect()
    if err != nil {
        m.messages = append(m.messages, message{
            content: fmt.Sprintf("Error connecting to agent server: %v", err),
            msgType: errorMessage,
        })
        m.updateViewport()
        
        // Fall back to mock agent if connection fails
        m.useRemoteAgent = false
        m.agentService = mock.NewAgentService()
    } else {
        // Add welcome message
        m.messages = append(m.messages, message{
            content: "Connected to Cloud-Assist Agent. What would you like help with?",
            msgType: agentMessage,
        })
        m.updateViewport()
    }
}

// Modify handleUserInput to use the agent client when enabled
func (m ChatModel) handleUserInput() (tea.Model, tea.Cmd) {
    userInput := m.input.Value()
    if strings.TrimSpace(userInput) == "" {
        return m, nil
    }

    // Add user message to chat
    m.messages = append(m.messages, message{
        content: userInput,
        msgType: userMessage,
    })

    // Reset input
    m.input = NewMultiline("", "What would you like to do?", m.width-4, 5)

    if m.useRemoteAgent && m.agentClient != nil {
        // Send message to remote agent
        err := m.agentClient.SendMessage(userInput)
        if err != nil {
            m.messages = append(m.messages, message{
                content: fmt.Sprintf("Error sending message to agent: %v", err),
                msgType: errorMessage,
            })
        }
    } else {
        // Use mock agent (existing code)
        // ... existing mock agent code ...
    }

    m.updateViewport()
    return m, nil
}

// Modify handleCommandApproval to use the agent client when enabled
func (m ChatModel) handleCommandApproval(response string) (tea.Model, tea.Cmd) {
    // Add user response to chat
    var responseText string
    switch response {
    case "y":
        responseText = "y"
    case "n":
        responseText = "n"
    case "e":
        responseText = "e"
    }

    m.messages = append(m.messages, message{
        content: responseText,
        msgType: userMessage,
    })

    if m.useRemoteAgent && m.agentClient != nil {
        if response == "y" {
            // Approve the command
            err := m.agentClient.ApproveCommand()
            if err != nil {
                m.messages = append(m.messages, message{
                    content: fmt.Sprintf("Error approving command: %v", err),
                    msgType: errorMessage,
                })
            }
            // The command output will come through the websocket
        } else if response == "e" {
            // Just display the explanation again
            m.messages = append(m.messages, message{
                content: m.currentExplanation,
                msgType: agentMessage,
            })
            // Keep suggestion mode active after explanation
            m.suggestionMode = true
            m.showInput = false
        } else {
            // Deny the command
            err := m.agentClient.DenyCommand()
            if err != nil {
                m.messages = append(m.messages, message{
                    content: fmt.Sprintf("Error denying command: %v", err),
                    msgType: errorMessage,
                })
            }
            // Exit suggestion mode
            m.suggestionMode = false
            m.pendingApproval = false
            m.showInput = true
            m.messages = append(m.messages, message{
                content: "Command skipped. What would you like to do instead?",
                msgType: agentMessage,
            })
        }
    } else {
        // Use mock agent (existing code)
        // ... existing mock agent code ...
    }

    m.updateViewport()
    return m, nil
}

// Add cleanup method for agent client
func (m *ChatModel) Cleanup() {
    if m.useRemoteAgent && m.agentClient != nil {
        m.agentClient.Close()
    }
}
```

### Agent Server Implementation

```python
# filepath: /home/nakulbh/Desktop/Projects/PersonalProjects/cloud-assist/agent/agent_server.py
from agno.agent import Agent
from agno.models.openai import OpenAI
from agno.tools import FunctionCall, tool
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
import os
import json
import asyncio
import logging
from typing import List, Dict, Any, Iterator, Optional, Callable, Union

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("cloud-assist")

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

# Global settings
ENABLE_COMMAND_EXECUTION = True
DEFAULT_MODEL = os.getenv("AGENT_MODEL", "openai")
DEFAULT_API_KEY = os.getenv("OPENAI_API_KEY", "")


class WebSocketState:
    """Manages state for a WebSocket connection."""
    
    def __init__(self, websocket: WebSocket):
        self.websocket = websocket
        self.pending_approval: Optional[asyncio.Future] = None
        self.user_id: str = "anonymous"
        self.context: Dict[str, Any] = {}
        self.session_id: str = ""


# Connection manager
class ConnectionManager:
    def __init__(self):
        self.active_connections: Dict[str, WebSocketState] = {}
        self.connection_count = 0

    async def connect(self, websocket: WebSocket) -> str:
        await websocket.accept()
        self.connection_count += 1
        session_id = f"session_{self.connection_count}"
        state = WebSocketState(websocket)
        state.session_id = session_id
        self.active_connections[session_id] = state
        logger.info(f"New connection: {session_id}")
        return session_id

    def disconnect(self, session_id: str):
        if session_id in self.active_connections:
            state = self.active_connections[session_id]
            # Cancel any pending approval requests
            if state.pending_approval and not state.pending_approval.done():
                state.pending_approval.cancel()
            del self.active_connections[session_id]
            logger.info(f"Connection closed: {session_id}")

    async def send_message(self, session_id: str, content: str):
        if session_id in self.active_connections:
            await self.active_connections[session_id].websocket.send_text(
                json.dumps({"type": "message", "content": content})
            )

    async def send_error(self, session_id: str, error: str):
        if session_id in self.active_connections:
            await self.active_connections[session_id].websocket.send_text(
                json.dumps({"type": "error", "error": error})
            )

    async def send_command_approval(
        self, session_id: str, command: List[str], explanation: str
    ) -> bool:
        """
        Send command approval request and wait for response.
        Returns True if approved, False if denied or timeout.
        """
        if session_id not in self.active_connections:
            return False
            
        state = self.active_connections[session_id]
        
        # Create a future to store the approval result
        state.pending_approval = asyncio.Future()
        
        # Send the approval request
        await state.websocket.send_text(
            json.dumps({
                "type": "command_approval", 
                "command": command,
                "explanation": explanation
            })
        )
        
        try:
            # Wait for the response with a timeout
            return await asyncio.wait_for(state.pending_approval, timeout=300)
        except asyncio.TimeoutError:
            logger.warning(f"Approval request timed out for session {session_id}")
            return False
        except asyncio.CancelledError:
            logger.warning(f"Approval request cancelled for session {session_id}")
            return False

    async def send_command_output(self, session_id: str, output: str):
        if session_id in self.active_connections:
            await self.active_connections[session_id].websocket.send_text(
                json.dumps({"type": "command_output", "content": output})
            )

    def set_approval_result(self, session_id: str, approved: bool):
        """Set the result for a pending approval request."""
        if session_id in self.active_connections:
            state = self.active_connections[session_id]
            if state.pending_approval and not state.pending_approval.done():
                state.pending_approval.set_result(approved)


# Create connection manager
manager = ConnectionManager()


# Define the pre-execution hook for human approval
async def ws_approval_hook(
    fc: FunctionCall, 
    session_id: str,
    explanation_override: Optional[str] = None
) -> bool:
    """Request human approval before executing a command."""
    if not ENABLE_COMMAND_EXECUTION:
        logger.warning(f"Command execution is disabled: {fc.function.name}")
        return False
        
    if fc.function.name == "execute_command":
        command = fc.args.get("command", [])
        explanation = explanation_override or f"About to execute: {' '.join(command)}"
        
        # Get approval from user via WebSocket
        approved = await manager.send_command_approval(session_id, command, explanation)
        
        if not approved:
            logger.info(f"Command execution declined by user: {' '.join(command)}")
            
        return approved
    
    # For other function types, default to approve
    return True


# Define DevOps tools with human approval
@tool()
async def execute_command(
    command: List[str], 
    cwd: Optional[str] = None
) -> Iterator[str]:
    """Execute a shell command with user approval.
    
    Args:
        command (List[str]): Command to execute as a list of arguments
        cwd (str, optional): Working directory
        
    Returns:
        Iterator[str]: Command output lines
    """
    import subprocess
    import shlex
    
    # Security check - we could implement more thorough checks here
    if not command:
        yield "Error: Empty command"
        return
        
    # Log the command being executed
    cmd_str = " ".join(shlex.quote(arg) for arg in command)
    logger.info(f"Executing command: {cmd_str}")
    
    try:
        # Execute command in a safe environment
        process = subprocess.Popen(
            command,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            cwd=cwd,
            env=os.environ.copy()  # Use a copy of the current environment
        )
        
        # Stream output
        for line in process.stdout:
            line = line.rstrip()
            yield line
        
        # Wait for process to complete and get return code
        return_code = process.wait()
        if return_code != 0:
            yield f"Command failed with exit code {return_code}"
            
    except Exception as e:
        yield f"Error executing command: {str(e)}"


# Create the default agent
def create_agent(model_name: str = DEFAULT_MODEL) -> Agent:
    """Create an agent with the specified model."""
    
    if model_name.startswith("openai"):
        from agno.models.openai import OpenAI
        model = OpenAI()
    elif model_name.startswith("groq"):
        from agno.models.groq import Groq
        model = Groq()
    else:
        # Default to OpenAI
        from agno.models.openai import OpenAI
        model = OpenAI
    
    return Agent(
        model=model,
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
        
        When suggesting commands:
        1. First use informational commands to understand the environment
        2. Explain what you're going to do before suggesting commands
        3. Check command results to verify success
        4. Provide context and explanation for each command
        """,
        tools=[execute_command],
        show_tool_calls=True,
        markdown=True,
    )


# Create the default agent instance
agent = create_agent()


# REST API endpoint for checking server status
@app.get("/api/status")
async def status():
    """Return server status information"""
    return {
        "status": "online",
        "connections": len(manager.active_connections),
        "command_execution_enabled": ENABLE_COMMAND_EXECUTION
    }


# WebSocket endpoint for interactive sessions
@app.websocket("/ws/chat")
async def websocket_endpoint(websocket: WebSocket):
    session_id = await manager.connect(websocket)
    try:
        # Create a session-specific agent
        session_agent = create_agent()
        
        while True:
            # Receive message from client
            data = await websocket.receive_text()
            message_data = json.loads(data)
            
            # Handle different message types
            if message_data["type"] == "message":
                user_message = message_data.get("content", "")
                logger.info(f"Received message from {session_id}: {user_message[:50]}...")
                
                try:
                    # Process message with streaming response
                    async for chunk in session_agent.astream(
                        user_message,
                        tool_configs={
                            # Configure the approval hook with session_id
                            "execute_command": {
                                "pre_call": lambda fc: ws_approval_hook(fc, session_id)
                            }
                        }
                    ):
                        await manager.send_message(session_id, chunk)
                except Exception as e:
                    logger.error(f"Error processing message: {str(e)}")
                    await manager.send_error(session_id, f"Error: {str(e)}")
            
            elif message_data["type"] == "command_approval":
                # Handle approval response
                approved = message_data.get("approved", False)
                manager.set_approval_result(session_id, approved)
                
                if approved:
                    logger.info(f"Command approved by user {session_id}")
                else:
                    logger.info(f"Command denied by user {session_id}")
    
    except WebSocketDisconnect:
        logger.info(f"WebSocket disconnected: {session_id}")
        manager.disconnect(session_id)
    except Exception as e:
        logger.error(f"Error in WebSocket connection: {str(e)}")
        manager.disconnect(session_id)


if __name__ == "__main__":
    import uvicorn
    
    port = int(os.getenv("PORT", "8000"))
    host = os.getenv("HOST", "0.0.0.0")
    
    logger.info(f"Starting Cloud-Assist Agent Server on {host}:{port}")
    uvicorn.run(app, host=host, port=port)
```

### Agent Tools Implementation

```python
# filepath: /home/nakulbh/Desktop/Projects/PersonalProjects/cloud-assist/agent/tools.py
from agno.tools import tool
from typing import List, Dict, Any, Optional, Iterator, Union
import logging
import os
import json
import subprocess
import shlex
import shutil
from pathlib import Path

logger = logging.getLogger("cloud-assist.tools")

# Tool for executing shell commands
@tool()
async def execute_command(
    command: List[str], 
    cwd: Optional[str] = None,
    timeout: Optional[int] = 60,
    env: Optional[Dict[str, str]] = None
) -> Iterator[str]:
    """Execute a shell command with user approval.
    
    Args:
        command (List[str]): Command to execute as a list of arguments
        cwd (str, optional): Working directory
        timeout (int, optional): Timeout in seconds
        env (Dict[str, str], optional): Additional environment variables
        
    Returns:
        Iterator[str]: Command output lines
    """
    import subprocess
    
    # Security check
    if not command:
        yield "Error: Empty command"
        return
        
    # Log the command being executed
    cmd_str = " ".join(shlex.quote(arg) for arg in command)
    logger.info(f"Executing command: {cmd_str}")
    
    # Prepare environment
    process_env = os.environ.copy()
    if env:
        process_env.update(env)
    
    try:
        # Execute command in a safe environment
        process = subprocess.Popen(
            command,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            cwd=cwd,
            env=process_env
        )
        
        # Stream output
        for line in process.stdout:
            line = line.rstrip()
            yield line
        
        # Wait for process to complete and get return code
        return_code = process.wait(timeout=timeout if timeout else None)
        if return_code != 0:
            yield f"Command failed with exit code {return_code}"
            
    except subprocess.TimeoutExpired:
        yield f"Command timed out after {timeout} seconds"
        try:
            process.terminate()
        except:
            pass
    except Exception as e:
        yield f"Error executing command: {str(e)}"


# Tool for file operations
@tool()
async def read_file(
    path: str,
    max_size: Optional[int] = 1024 * 1024  # 1MB default limit
) -> str:
    """Read the contents of a file.
    
    Args:
        path (str): Path to the file
        max_size (int, optional): Maximum file size to read in bytes
        
    Returns:
        str: File contents
    """
    file_path = Path(path)
    
    # Security checks
    if not file_path.exists():
        return f"Error: File not found: {path}"
    
    if not file_path.is_file():
        return f"Error: Not a file: {path}"
    
    try:
        # Check file size
        size = file_path.stat().st_size
        if max_size and size > max_size:
            return f"Error: File too large ({size} bytes). Maximum size is {max_size} bytes."
        
        # Read the file
        with open(file_path, 'r') as f:
            return f.read()
    
    except Exception as e:
        return f"Error reading file: {str(e)}"


@tool()
async def write_file(
    path: str,
    content: str,
    mode: str = "w"  # 'w' for write, 'a' for append
) -> str:
    """Write content to a file.
    
    Args:
        path (str): Path to the file
        content (str): Content to write
        mode (str): Write mode ('w' for write, 'a' for append)
        
    Returns:
        str: Result message
    """
    file_path = Path(path)
    
    # Security checks
    if file_path.exists() and not file_path.is_file():
        return f"Error: Path exists but is not a file: {path}"
    
    # Ensure the directory exists
    file_path.parent.mkdir(parents=True, exist_ok=True)
    
    try:
        # Write the file
        with open(file_path, mode) as f:
            f.write(content)
        
        return f"Successfully wrote {len(content)} characters to {path}"
    
    except Exception as e:
        return f"Error writing file: {str(e)}"


@tool()
async def list_directory(
    path: str,
    recursive: bool = False,
    include_hidden: bool = False
) -> List[Dict[str, Any]]:
    """List contents of a directory.
    
    Args:
        path (str): Path to the directory
        recursive (bool): Whether to list directories recursively
        include_hidden (bool): Whether to include hidden files (starting with .)
        
    Returns:
        List[Dict[str, Any]]: List of file/directory information
    """
    dir_path = Path(path)
    
    # Security checks
    if not dir_path.exists():
        return [{"error": f"Directory not found: {path}"}]
    
    if not dir_path.is_dir():
        return [{"error": f"Not a directory: {path}"}]
    
    try:
        result = []
        
        # Function to process a single directory
        def process_dir(dir_path, rel_path=""):
            items = []
            for item in dir_path.iterdir():
                # Skip hidden files if not requested
                if not include_hidden and item.name.startswith('.'):
                    continue
                
                rel_item_path = str(Path(rel_path) / item.name)
                
                if item.is_dir():
                    info = {
                        "name": item.name,
                        "path": str(item),
                        "type": "directory",
                        "relative_path": rel_item_path
                    }
                    items.append(info)
                    
                    # Process subdirectory if recursive
                    if recursive:
                        items.extend(process_dir(item, rel_item_path))
                else:
                    info = {
                        "name": item.name,
                        "path": str(item),
                        "type": "file",
                        "size": item.stat().st_size,
                        "relative_path": rel_item_path
                    }
                    items.append(info)
            
            return items
        
        # Process the directory
        result = process_dir(dir_path)
        
        return result
    
    except Exception as e:
        return [{"error": f"Error listing directory: {str(e)}"}]
```

