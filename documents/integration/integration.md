Cloud Assist Agent CLI Integration Documentation
Overview
This document provides comprehensive documentation for integrating the LangGraph-based Cloud Assist Agent with a CLI user interface. The integration enables real-time, interactive command generation and execution with human-in-the-loop approval through a WebSocket-based architecture.

Architecture
System Components
```
┌─────────────────────┐    WebSocket     ┌─────────────────────┐    Direct Call   ┌─────────────────────┐
│                     │ ◄───────────────► │                     │ ◄───────────────► │                     │
│   CLI UI (Go)       │                   │  Agent Server       │                   │  LangGraph Agent    │
│  - Bubble Tea UI    │                   │  (FastAPI/Python)   │                   │  - State Management │
│  - User Interaction │                   │  - WebSocket Handler│                   │  - Command Gen      │
│  - Command Display  │                   │  - Session Manager  │                   │  - Human Approval   │
│                     │                   │                     │                   │  - Command Exec     │
└─────────────────────┘                   └─────────────────────┘                   └─────────────────────┘
```
Data Flow
User Input: CLI captures user command request
WebSocket Message: CLI sends request to Agent Server
Graph Execution: Server invokes LangGraph agent
Human Approval: Agent interrupts for approval, sent to CLI
User Decision: CLI presents approval dialog, returns response
Command Execution: If approved, command executes
Result Display: Output/errors sent back to CLI
Implementation
1. Agent Server (Python FastAPI)
Create a WebSocket server that wraps the existing LangGraph agent:
```python
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
import json
import asyncio
import uuid
from typing import Dict, Any, Optional
from datetime import datetime
import logging
from .graph import create_agent_graph

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Cloud Assist Agent Server", version="1.0.0")

# CORS middleware for development
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"]
)

class SessionManager:
    """Manages active WebSocket sessions and agent states"""
    
    def __init__(self):
        self.active_sessions: Dict[str, Dict] = {}
        self.agent_graph = create_agent_graph()
    
    def create_session(self, session_id: str, websocket: WebSocket) -> Dict:
        """Create a new session with WebSocket connection"""
        session_data = {
            "id": session_id,
            "websocket": websocket,
            "created_at": datetime.now(),
            "config": {"configurable": {"thread_id": session_id}},
            "status": "connected",
            "current_state": None
        }
        self.active_sessions[session_id] = session_data
        logger.info(f"Created session: {session_id}")
        return session_data
    
    def remove_session(self, session_id: str):
        """Remove session and cleanup resources"""
        if session_id in self.active_sessions:
            del self.active_sessions[session_id]
            logger.info(f"Removed session: {session_id}")
    
    def get_session(self, session_id: str) -> Optional[Dict]:
        """Get session data by ID"""
        return self.active_sessions.get(session_id)
    
    async def send_message(self, session_id: str, message: Dict):
        """Send message to specific session"""
        session = self.get_session(session_id)
        if session and session["status"] == "connected":
            try:
                await session["websocket"].send_text(json.dumps(message))
            except Exception as e:
                logger.error(f"Failed to send message to {session_id}: {e}")
                session["status"] = "disconnected"

class AgentManager:
    """Manages agent execution and handles interrupts"""
    
    def __init__(self, session_manager: SessionManager):
        self.session_manager = session_manager
        self.pending_interrupts: Dict[str, Dict] = {}
    
    async def process_request(self, session_id: str, user_prompt: str):
        """Process user request through the agent graph"""
        session = self.session_manager.get_session(session_id)
        if not session:
            raise ValueError(f"Session {session_id} not found")
        
        # Create initial state
        initial_state = {
            "user_prompt": user_prompt,
            "messages": [],
            "generated_command": "",
            "command_output": "",
            "command_error": "",
            "execution_approved": False,
            "retry_count": 0
        }
        
        config = session["config"]
        session["current_state"] = initial_state
        
        try:
            # Send start message
            await self.session_manager.send_message(session_id, {
                "type": "agent_started",
                "data": {
                    "user_prompt": user_prompt,
                    "session_id": session_id
                }
            })
            
            # Execute graph and handle interrupts
            async for event in self.session_manager.agent_graph.astream(initial_state, config=config):
                await self.handle_agent_event(session_id, event)
                
            # Get final state
            final_state = self.session_manager.agent_graph.get_state(config)
            await self.session_manager.send_message(session_id, {
                "type": "agent_completed",
                "data": {
                    "final_state": final_state.values,
                    "session_id": session_id
                }
            })
            
        except Exception as e:
            logger.error(f"Agent execution error for session {session_id}: {e}")
            await self.session_manager.send_message(session_id, {
                "type": "agent_error",
                "data": {
                    "error": str(e),
                    "session_id": session_id
                }
            })
    
    async def handle_agent_event(self, session_id: str, event: Dict):
        """Handle different types of agent events"""
        logger.info(f"Agent event for {session_id}: {list(event.keys())}")
        
        # Check for interrupts in any node output
        for node_name, node_output in event.items():
            if hasattr(node_output, "__interrupt__"):
                # This is an interrupt - store it and send to client
                interrupt_data = node_output.__interrupt__
                self.pending_interrupts[session_id] = {
                    "node": node_name,
                    "data": interrupt_data,
                    "timestamp": datetime.now()
                }
                
                await self.session_manager.send_message(session_id, {
                    "type": "human_input_required",
                    "data": {
                        "node": node_name,
                        "interrupt_data": interrupt_data,
                        "session_id": session_id
                    }
                })
                return
        
        # Regular event update
        await self.session_manager.send_message(session_id, {
            "type": "agent_update",
            "data": {
                "event": event,
                "session_id": session_id
            }
        })
    
    async def handle_user_response(self, session_id: str, response: str):
        """Handle user response to interrupt"""
        if session_id not in self.pending_interrupts:
            raise ValueError(f"No pending interrupt for session {session_id}")
        
        interrupt_info = self.pending_interrupts[session_id]
        config = self.session_manager.get_session(session_id)["config"]
        
        # Resume graph execution with user response
        try:
            # Update the graph state with the user response
            self.session_manager.agent_graph.update_state(
                config, 
                {"human_response": response},
                as_node=interrupt_info["node"]
            )
            
            # Clear the pending interrupt
            del self.pending_interrupts[session_id]
            
            # Continue graph execution
            async for event in self.session_manager.agent_graph.astream(None, config=config):
                await self.handle_agent_event(session_id, event)
                
        except Exception as e:
            logger.error(f"Error handling user response for {session_id}: {e}")
            await self.session_manager.send_message(session_id, {
                "type": "agent_error",
                "data": {
                    "error": f"Failed to process response: {str(e)}",
                    "session_id": session_id
                }
            })

# Initialize managers
session_manager = SessionManager()
agent_manager = AgentManager(session_manager)

@app.websocket("/ws/{session_id}")
async def websocket_endpoint(websocket: WebSocket, session_id: str):
    """WebSocket endpoint for CLI communication"""
    await websocket.accept()
    logger.info(f"WebSocket connection established: {session_id}")
    
    # Create session
    session_manager.create_session(session_id, websocket)
    
    try:
        # Send connection confirmation
        await session_manager.send_message(session_id, {
            "type": "connection_established",
            "data": {
                "session_id": session_id,
                "timestamp": datetime.now().isoformat()
            }
        })
        
        while True:
            # Receive message from CLI
            data = await websocket.receive_text()
            message = json.loads(data)
            
            logger.info(f"Received message from {session_id}: {message.get('type')}")
            
            if message["type"] == "user_request":
                # Start agent processing
                await agent_manager.process_request(
                    session_id, 
                    message["data"]["prompt"]
                )
                
            elif message["type"] == "user_response":
                # Handle user response to interrupt
                await agent_manager.handle_user_response(
                    session_id, 
                    message["data"]["response"]
                )
                
            elif message["type"] == "ping":
                # Health check
                await session_manager.send_message(session_id, {
                    "type": "pong",
                    "data": {"timestamp": datetime.now().isoformat()}
                })
                
    except WebSocketDisconnect:
        logger.info(f"WebSocket disconnected: {session_id}")
    except Exception as e:
        logger.error(f"WebSocket error for {session_id}: {e}")
    finally:
        session_manager.remove_session(session_id)

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "active_sessions": len(session_manager.active_sessions),
        "timestamp": datetime.now().isoformat()
    }

@app.get("/sessions")
async def list_sessions():
    """List active sessions (for debugging)"""
    return {
        "sessions": [
            {
                "id": sid,
                "created_at": session["created_at"].isoformat(),
                "status": session["status"]
            }
            for sid, session in session_manager.active_sessions.items()
        ]
    }

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
````

2. WebSocket Client (Go)
Create a WebSocket client to communicate with the agent server:
```go
package agent

import (
    "encoding/json"
    "fmt"
    "log"
    "net/url"
    "sync"
    "time"

    "github.com/gorilla/websocket"
    "github.com/google/uuid"
)

// Message types for WebSocket communication
type MessageType string

const (
    UserRequest           MessageType = "user_request"
    UserResponse          MessageType = "user_response"
    HumanInputRequired    MessageType = "human_input_required"
    AgentStarted          MessageType = "agent_started"
    AgentUpdate           MessageType = "agent_update"
    AgentCompleted        MessageType = "agent_completed"
    AgentError            MessageType = "agent_error"
    ConnectionEstablished MessageType = "connection_established"
    Ping                  MessageType = "ping"
    Pong                  MessageType = "pong"
)

// WebSocket message structure
type WebSocketMessage struct {
    Type MessageType    `json:"type"`
    Data interface{}    `json:"data"`
}

// Specific message data structures
type UserRequestData struct {
    Prompt string `json:"prompt"`
}

type UserResponseData struct {
    Response string `json:"response"`
}

type HumanInputData struct {
    Node         string      `json:"node"`
    InterruptData interface{} `json:"interrupt_data"`
    SessionID    string      `json:"session_id"`
}

type AgentEventData struct {
    Event     map[string]interface{} `json:"event"`
    SessionID string                 `json:"session_id"`
}

// Event handler interface
type EventHandler interface {
    OnConnectionEstablished(sessionID string)
    OnAgentStarted(data AgentEventData)
    OnAgentUpdate(data AgentEventData)
    OnAgentCompleted(data AgentEventData)
    OnAgentError(error string)
    OnHumanInputRequired(data HumanInputData) string
}

// Agent client structure
type Client struct {
    serverURL    string
    sessionID    string
    conn         *websocket.Conn
    handler      EventHandler
    mu           sync.RWMutex
    connected    bool
    done         chan struct{}
    reconnectWait time.Duration
}

// Create new agent client
func NewClient(serverURL string, handler EventHandler) *Client {
    sessionID := uuid.New().String()
    
    return &Client{
        serverURL:     serverURL,
        sessionID:     sessionID,
        handler:       handler,
        connected:     false,
        done:          make(chan struct{}),
        reconnectWait: 5 * time.Second,
    }
}

// Connect to the agent server
func (c *Client) Connect() error {
    u, err := url.Parse(c.serverURL)
    if err != nil {
        return fmt.Errorf("invalid server URL: %w", err)
    }
    
    u.Path = fmt.Sprintf("/ws/%s", c.sessionID)
    
    conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        return fmt.Errorf("failed to connect to server: %w", err)
    }
    
    c.mu.Lock()
    c.conn = conn
    c.connected = true
    c.mu.Unlock()
    
    // Start message handler
    go c.messageHandler()
    
    log.Printf("Connected to agent server with session ID: %s", c.sessionID)
    return nil
}

// Disconnect from server
func (c *Client) Disconnect() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if !c.connected {
        return nil
    }
    
    close(c.done)
    c.connected = false
    
    if c.conn != nil {
        return c.conn.Close()
    }
    
    return nil
}

// Send user request to agent
func (c *Client) SendRequest(prompt string) error {
    return c.sendMessage(WebSocketMessage{
        Type: UserRequest,
        Data: UserRequestData{Prompt: prompt},
    })
}

// Send user response to interrupt
func (c *Client) SendResponse(response string) error {
    return c.sendMessage(WebSocketMessage{
        Type: UserResponse,
        Data: UserResponseData{Response: response},
    })
}

// Send ping for health check
func (c *Client) Ping() error {
    return c.sendMessage(WebSocketMessage{
        Type: Ping,
        Data: map[string]interface{}{"timestamp": time.Now().Unix()},
    })
}

// Send message to server
func (c *Client) sendMessage(message WebSocketMessage) error {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if !c.connected || c.conn == nil {
        return fmt.Errorf("not connected to server")
    }
    
    data, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }
    
    return c.conn.WriteMessage(websocket.TextMessage, data)
}

// Handle incoming messages
func (c *Client) messageHandler() {
    defer func() {
        c.mu.Lock()
        c.connected = false
        c.mu.Unlock()
    }()
    
    for {
        select {
        case <-c.done:
            return
        default:
            _, messageData, err := c.conn.ReadMessage()
            if err != nil {
                log.Printf("Error reading message: %v", err)
                return
            }
            
            var message WebSocketMessage
            if err := json.Unmarshal(messageData, &message); err != nil {
                log.Printf("Error unmarshaling message: %v", err)
                continue
            }
            
            c.handleMessage(message)
        }
    }
}

// Handle different message types
func (c *Client) handleMessage(message WebSocketMessage) {
    switch message.Type {
    case ConnectionEstablished:
        c.handler.OnConnectionEstablished(c.sessionID)
        
    case AgentStarted:
        data := parseAgentEventData(message.Data)
        c.handler.OnAgentStarted(data)
        
    case AgentUpdate:
        data := parseAgentEventData(message.Data)
        c.handler.OnAgentUpdate(data)
        
    case AgentCompleted:
        data := parseAgentEventData(message.Data)
        c.handler.OnAgentCompleted(data)
        
    case AgentError:
        errorMsg := fmt.Sprintf("%v", message.Data)
        c.handler.OnAgentError(errorMsg)
        
    case HumanInputRequired:
        data := parseHumanInputData(message.Data)
        response := c.handler.OnHumanInputRequired(data)
        if response != "" {
            c.SendResponse(response)
        }
        
    case Pong:
        log.Printf("Received pong from server")
        
    default:
        log.Printf("Unknown message type: %s", message.Type)
    }
}

// Helper functions to parse message data
func parseAgentEventData(data interface{}) AgentEventData {
    dataMap, ok := data.(map[string]interface{})
    if !ok {
        return AgentEventData{}
    }
    
    event, _ := dataMap["event"].(map[string]interface{})
    sessionID, _ := dataMap["session_id"].(string)
    
    return AgentEventData{
        Event:     event,
        SessionID: sessionID,
    }
}

func parseHumanInputData(data interface{}) HumanInputData {
    dataMap, ok := data.(map[string]interface{})
    if !ok {
        return HumanInputData{}
    }
    
    node, _ := dataMap["node"].(string)
    interruptData := dataMap["interrupt_data"]
    sessionID, _ := dataMap["session_id"].(string)
    
    return HumanInputData{
        Node:         node,
        InterruptData: interruptData,
        SessionID:    sessionID,
    }
}

// Check if client is connected
func (c *Client) IsConnected() bool {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.connected
}

// Get session ID
func (c *Client) GetSessionID() string {
    return c.sessionID
}
```
3. CLI UI Integration
Create UI components to handle agent interactions:
```go
package ui

import (
    "fmt"
    "log"
    "strings"
    
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    
    "your-project/internal/agent"
)

// AgentHandler implements the agent.EventHandler interface
type AgentHandler struct {
    program     *tea.Program
    model       *Model
    messages    []string
    currentStep string
}

// Create new agent handler
func NewAgentHandler(program *tea.Program, model *Model) *AgentHandler {
    return &AgentHandler{
        program:  program,
        model:    model,
        messages: make([]string, 0),
    }
}

// EventHandler interface implementations
func (h *AgentHandler) OnConnectionEstablished(sessionID string) {
    h.addMessage(fmt.Sprintf("🔗 Connected to agent (Session: %s)", sessionID))
    h.updateUI()
}

func (h *AgentHandler) OnAgentStarted(data agent.AgentEventData) {
    h.currentStep = "Command Generation"
    h.addMessage("🤖 Agent started processing your request...")
    h.updateUI()
}

func (h *AgentHandler) OnAgentUpdate(data agent.AgentEventData) {
    // Parse the event to understand what's happening
    for nodeName, nodeData := range data.Event {
        switch nodeName {
        case "generate_command":
            h.currentStep = "Command Generated"
            if nodeOutput, ok := nodeData.(map[string]interface{}); ok {
                if cmd, exists := nodeOutput["generated_command"]; exists {
                    h.addMessage(fmt.Sprintf("💡 Generated command: %s", cmd))
                }
            }
        case "execute_command":
            h.currentStep = "Executing Command"
            h.addMessage("⚙️  Executing command...")
        case "check_result":
            h.currentStep = "Checking Results"
            if nodeOutput, ok := nodeData.(map[string]interface{}); ok {
                if output, exists := nodeOutput["command_output"]; exists {
                    h.addMessage(fmt.Sprintf("📄 Command output: %s", output))
                }
                if err, exists := nodeOutput["command_error"]; exists && err != "" {
                    h.addMessage(fmt.Sprintf("❌ Command error: %s", err))
                }
            }
        }
    }
    h.updateUI()
}

func (h *AgentHandler) OnAgentCompleted(data agent.AgentEventData) {
    h.currentStep = "Completed"
    h.addMessage("✅ Agent completed successfully!")
    
    // Display final state if available
    if finalState, ok := data.Event["final_state"].(map[string]interface{}); ok {
        if output, exists := finalState["command_output"]; exists && output != "" {
            h.addMessage(fmt.Sprintf("📋 Final output:\n%s", output))
        }
    }
    
    h.updateUI()
}

func (h *AgentHandler) OnAgentError(error string) {
    h.currentStep = "Error"
    h.addMessage(fmt.Sprintf("❌ Agent error: %s", error))
    h.updateUI()
}

func (h *AgentHandler) OnHumanInputRequired(data agent.HumanInputData) string {
    h.addMessage("⏸️  Human input required...")
    
    switch data.Node {
    case "human_approval":
        return h.handleApprovalRequest(data.InterruptData)
    case "check_result":
        return h.handleRetryRequest(data.InterruptData)
    default:
        return h.handleGenericInput(data.InterruptData)
    }
}

// Handle approval requests
func (h *AgentHandler) handleApprovalRequest(interruptData interface{}) string {
    data, ok := interruptData.(map[string]interface{})
    if !ok {
        return "reject"
    }
    
    command, _ := data["command"].(string)
    userPrompt, _ := data["user_prompt"].(string)
    retryCount, _ := data["retry_count"].(float64)
    
    // Show approval dialog
    title := "Command Approval Required"
    content := fmt.Sprintf(
        "User Request: %s\n\nGenerated Command: %s\n\nRetry Count: %.0f\n\nDo you want to execute this command?",
        userPrompt, command, retryCount,
    )
    
    response := h.showConfirmationDialog(title, content, []string{"Approve", "Reject", "Cancel"})
    
    switch response {
    case 0:
        h.addMessage("✅ Command approved by user")
        return "approve"
    case 1:
        h.addMessage("❌ Command rejected by user")
        return "reject"
    default:
        h.addMessage("🚫 Operation cancelled by user")
        return "cancel"
    }
}

// Handle retry requests
func (h *AgentHandler) handleRetryRequest(interruptData interface{}) string {
    data, ok := interruptData.(map[string]interface{})
    if !ok {
        return "cancel"
    }
    
    command, _ := data["command"].(string)
    error, _ := data["error"].(string)
    retryCount, _ := data["retry_count"].(float64)
    
    title := "Command Failed - Retry?"
    content := fmt.Sprintf(
        "Command: %s\n\nError: %s\n\nRetry Count: %.0f\n\nWould you like to retry with a different approach?",
        command, error, retryCount,
    )
    
    response := h.showConfirmationDialog(title, content, []string{"Retry", "Cancel"})
    
    switch response {
    case 0:
        h.addMessage("🔄 User chose to retry with different approach")
        return "retry"
    default:
        h.addMessage("🚫 User cancelled retry")
        return "cancel"
    }
}

// Handle generic input requests
func (h *AgentHandler) handleGenericInput(interruptData interface{}) string {
    // For generic inputs, show a text input dialog
    title := "Input Required"
    content := fmt.Sprintf("Agent requires input: %v", interruptData)
    
    response := h.showTextInputDialog(title, content)
    h.addMessage(fmt.Sprintf("📝 User provided input: %s", response))
    return response
}

// Show confirmation dialog
func (h *AgentHandler) showConfirmationDialog(title, content string, options []string) int {
    // This is a simplified implementation
    // In a real implementation, you would integrate this with your Bubble Tea model
    
    // For now, we'll use a basic approach
    // You should implement proper dialog components using Bubble Tea
    
    log.Printf("Confirmation Dialog - %s: %s", title, content)
    
    // Default to first option for demo
    // In real implementation, this would show an interactive dialog
    return 0
}

// Show text input dialog
func (h *AgentHandler) showTextInputDialog(title, content string) string {
    // This is a simplified implementation
    // In a real implementation, you would integrate this with your Bubble Tea model
    
    log.Printf("Text Input Dialog - %s: %s", title, content)
    
    // Default response for demo
    return "user input"
}

// Add message to the chat
func (h *AgentHandler) addMessage(message string) {
    timestamp := "[" + getCurrentTime() + "] "
    h.messages = append(h.messages, timestamp+message)
    
    // Keep only last 100 messages
    if len(h.messages) > 100 {
        h.messages = h.messages[1:]
    }
}

// Update the UI
func (h *AgentHandler) updateUI() {
    if h.program != nil {
        h.program.Send(AgentUpdateMsg{
            Messages:    h.messages,
            CurrentStep: h.currentStep,
        })
    }
}

// Custom message type for agent updates
type AgentUpdateMsg struct {
    Messages    []string
    CurrentStep string
}

// Get current time formatted
func getCurrentTime() string {
    return "15:04:05" // This should use actual time formatting
}
```
4. Main CLI Application
Integrate everything in the main CLI application:
package main

import (
    "fmt"
    "log"
    "os"
    "strings"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/spf13/cobra"
    
    "your-project/internal/agent"
    "your-project/internal/ui"
)

var (
    serverURL = "ws://localhost:8000"
    debug     = false
)

func main() {
    var rootCmd = &cobra.Command{
        Use:   "cloud-assist",
        Short: "AI-powered cloud command assistant",
        Long:  "An interactive CLI tool that uses AI to generate and execute cloud commands with human oversight.",
        Run:   runInteractive,
    }
    
    rootCmd.Flags().StringVar(&serverURL, "server", serverURL, "Agent server URL")
    rootCmd.Flags().BoolVar(&debug, "debug", debug, "Enable debug logging")
    
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func runInteractive(cmd *cobra.Command, args []string) {
    if debug {
        log.SetOutput(os.Stderr)
    }
    
    // Initialize Bubble Tea program
    model := ui.NewModel()
    program := tea.NewProgram(model, tea.WithAltScreen())
    
    // Create agent handler
    agentHandler := ui.NewAgentHandler(program, model)
    
    // Create agent client
    client := agent.NewClient(serverURL, agentHandler)
    
    // Connect to agent server
    if err := client.Connect(); err != nil {
        log.Fatalf("Failed to connect to agent server: %v", err)
    }
    defer client.Disconnect()
    
    // Set the client in the model
    model.SetAgentClient(client)
    
    // Run the program
    if _, err := program.Run(); err != nil {
        log.Fatalf("Error running program: %v", err)
    }
}

Configuration
Server Configuration
Create configuration files for the server:

import os
from typing import Optional

class ServerConfig:
    """Server configuration settings"""
    
    HOST: str = os.getenv("AGENT_HOST", "0.0.0.0")
    PORT: int = int(os.getenv("AGENT_PORT", "8000"))
    DEBUG: bool = os.getenv("AGENT_DEBUG", "false").lower() == "true"
    
    # WebSocket settings
    WEBSOCKET_TIMEOUT: int = int(os.getenv("WEBSOCKET_TIMEOUT", "300"))
    MAX_CONNECTIONS: int = int(os.getenv("MAX_CONNECTIONS", "100"))
    
    # Agent settings
    MAX_RETRIES: int = int(os.getenv("MAX_RETRIES", "3"))
    COMMAND_TIMEOUT: int = int(os.getenv("COMMAND_TIMEOUT", "30"))
    
    # Logging
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "INFO")
    LOG_FORMAT: str = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
    
    @classmethod
    def validate(cls) -> bool:
        """Validate configuration"""
        if cls.PORT < 1 or cls.PORT > 65535:
            raise ValueError("Invalid port number")
        return True


Environment Variables
Create a .env file for configuration:

# Agent Server Configuration
AGENT_HOST=0.0.0.0
AGENT_PORT=8000
AGENT_DEBUG=false

# WebSocket Configuration
WEBSOCKET_TIMEOUT=300
MAX_CONNECTIONS=100

# Agent Configuration
MAX_RETRIES=3
COMMAND_TIMEOUT=30

# Logging Configuration
LOG_LEVEL=INFO

# LLM Configuration (from existing setup)
GROQ_API_KEY=your_groq_api_key_here