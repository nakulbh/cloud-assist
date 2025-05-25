package client

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

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

// AgentClient handles communication with the agent server
type AgentClient struct {
	conn              *websocket.Conn
	serverURL         string
	connected         bool
	connectionMutex   sync.Mutex
	sendMutex         sync.Mutex
	done              chan struct{}
	onMessage         func(string)
	onCommandApproval func([]string, string)
	onCommandOutput   func(string)
	onError           func(string)
}

// NewAgentClient creates a new agent client
func NewAgentClient(serverURL string) *AgentClient {
	return &AgentClient{
		serverURL: serverURL,
		done:      make(chan struct{}),
	}
}

// Connect establishes a connection to the WebSocket server
func (c *AgentClient) Connect() error {
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

	// Start listening for messages
	go c.readPump()
	return nil
}

// SendMessage sends a message to the agent
func (c *AgentClient) SendMessage(content string) error {
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
func (c *AgentClient) SendApproval(approved bool) error {
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
func (c *AgentClient) Close() {
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
}

// IsConnected returns the connection status
func (c *AgentClient) IsConnected() bool {
	c.connectionMutex.Lock()
	defer c.connectionMutex.Unlock()
	return c.connected
}

// SetMessageHandler sets the callback for text messages
func (c *AgentClient) SetMessageHandler(handler func(string)) {
	c.onMessage = handler
}

// SetCommandApprovalHandler sets the callback for command approval requests
func (c *AgentClient) SetCommandApprovalHandler(handler func([]string, string)) {
	c.onCommandApproval = handler
}

// SetCommandOutputHandler sets the callback for command output
func (c *AgentClient) SetCommandOutputHandler(handler func(string)) {
	c.onCommandOutput = handler
}

// SetErrorHandler sets the callback for error messages
func (c *AgentClient) SetErrorHandler(handler func(string)) {
	c.onError = handler
}

// readPump handles incoming messages
func (c *AgentClient) readPump() {
	defer func() {
		c.connectionMutex.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.connected = false
		c.connectionMutex.Unlock()
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
				if c.onCommandApproval != nil {
					c.onCommandApproval(serverMsg.Command, serverMsg.Explanation)
				}
			case TypeCommandOutput:
				if c.onCommandOutput != nil {
					c.onCommandOutput(serverMsg.Content)
				}
			case TypeError:
				if c.onError != nil {
					c.onError(serverMsg.Error)
				}
			}
		}
	}
}
