package ui

import (
	"cloud-assist/client"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type messageType int

const (
	userMessage messageType = iota
	agentMessage
	commandSuggestion
	commandOutput
	errorMessage
	retryRequest
)

type message struct {
	content     string
	msgType     messageType
	command     []string
	explanation string
	retryCount  int
}

// Custom messages for Bubble Tea updates
type AgentConnectedMsg struct{}
type AgentDisconnectedMsg struct{}
type AgentMessageMsg struct {
	Content string
}
type CommandApprovalMsg struct {
	Command     []string
	Explanation string
}
type CommandOutputMsg struct {
	Output string
}
type RetryRequestMsg struct {
	Content    string
	RetryCount int
}
type AgentErrorMsg struct {
	Error string
}

// ChatModel represents the chat interface
type ChatModel struct {
	messages            []message
	viewport            viewport.Model
	input               MultilineModel
	width               int
	height              int
	showInput           bool
	suggestionMode      bool
	retryMode           bool
	currentCommand      []string
	currentExplanation  string
	currentRetryContent string
	currentRetryCount   int
	agentClient         *client.AgentClient
	connected           bool
	messageChannel      chan tea.Msg
}

// NewChatModel creates a new chat model
func NewChatModel(width, height int) ChatModel {
	input := NewMultiline("", "What would you like to do?", width-4, 5)
	vp := viewport.New(width, height-10)
	vp.Style = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(1).Border(lipgloss.NormalBorder(), false, true)

	agentClient := client.NewAgentClient("ws://localhost:8765")
	messageChannel := make(chan tea.Msg, 100)

	model := ChatModel{
		messages:       []message{},
		viewport:       vp,
		input:          input,
		width:          width,
		height:         height,
		showInput:      true,
		agentClient:    agentClient,
		messageChannel: messageChannel,
	}

	model.setupWebSocketHandlers()
	return model
}

// setupWebSocketHandlers configures the WebSocket client event handlers
func (m *ChatModel) setupWebSocketHandlers() {
	m.agentClient.SetMessageHandler(func(content string) {
		select {
		case m.messageChannel <- AgentMessageMsg{Content: content}:
		default:
		}
	})

	m.agentClient.SetCommandApprovalHandler(func(command []string, explanation string) {
		select {
		case m.messageChannel <- CommandApprovalMsg{Command: command, Explanation: explanation}:
		default:
		}
	})

	m.agentClient.SetCommandOutputHandler(func(output string) {
		select {
		case m.messageChannel <- CommandOutputMsg{Output: output}:
		default:
		}
	})

	m.agentClient.SetRetryRequestHandler(func(content string, retryCount int) {
		select {
		case m.messageChannel <- RetryRequestMsg{Content: content, RetryCount: retryCount}:
		default:
		}
	})

	m.agentClient.SetErrorHandler(func(error string) {
		select {
		case m.messageChannel <- AgentErrorMsg{Error: error}:
		default:
		}
	})

	m.agentClient.SetConnectionLostHandler(func() {
		select {
		case m.messageChannel <- AgentDisconnectedMsg{}:
		default:
		}
	})
}

// ConnectToAgent attempts to connect to the agent WebSocket server
func (m *ChatModel) ConnectToAgent() tea.Cmd {
	return func() tea.Msg {
		if err := m.agentClient.Connect(); err != nil {
			return AgentErrorMsg{Error: fmt.Sprintf("Failed to connect to agent: %v", err)}
		}
		return AgentConnectedMsg{}
	}
}

// DisconnectFromAgent disconnects from the agent WebSocket server
func (m *ChatModel) DisconnectFromAgent() {
	if m.agentClient != nil {
		m.agentClient.Disconnect()
	}
	m.connected = false
}

// Init initializes the chat model
func (m ChatModel) Init() tea.Cmd {
	return tea.Batch(
		m.input.Init(),
		m.ConnectToAgent(),
		m.listenForWebSocketMessages(),
	)
}

// listenForWebSocketMessages creates a command that listens for WebSocket messages
func (m ChatModel) listenForWebSocketMessages() tea.Cmd {
	return func() tea.Msg {
		return <-m.messageChannel
	}
}

// Update handles chat updates
func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 10
		m.input.SetWidth(msg.Width - 4)

	case AgentConnectedMsg:
		m.connected = true
		m.addMessage("How can I help you today?", agentMessage)
		cmds = append(cmds, m.listenForWebSocketMessages())

	case AgentDisconnectedMsg:
		m.connected = false
		m.addMessage("Disconnected from agent", errorMessage)
		cmds = append(cmds, m.listenForWebSocketMessages())

	case AgentMessageMsg:
		m.addMessage(msg.Content, agentMessage)
		cmds = append(cmds, m.listenForWebSocketMessages())

	case CommandApprovalMsg:
		m.suggestionMode = true
		m.currentCommand = msg.Command
		m.currentExplanation = msg.Explanation
		m.addCommandSuggestion(msg.Command, msg.Explanation)
		cmds = append(cmds, m.listenForWebSocketMessages())

	case CommandOutputMsg:
		m.addMessage(msg.Output, commandOutput)
		cmds = append(cmds, m.listenForWebSocketMessages())

	case RetryRequestMsg:
		m.retryMode = true
		m.currentRetryContent = msg.Content
		m.currentRetryCount = msg.RetryCount
		m.addRetryRequest(msg.Content, msg.RetryCount)
		cmds = append(cmds, m.listenForWebSocketMessages())

	case AgentErrorMsg:
		m.addMessage(fmt.Sprintf("Error: %s", msg.Error), errorMessage)
		cmds = append(cmds, m.listenForWebSocketMessages())

	case tea.KeyMsg:
		// Handle global keys first
		switch msg.String() {
		case "ctrl+c":
			if m.suggestionMode || m.retryMode {
				m.suggestionMode = false
				m.retryMode = false
				m.showInput = true
				m.input.Focus()

				if m.suggestionMode && m.connected {
					m.agentClient.SendCommandApproval(false)
				}
				if m.retryMode && m.connected {
					m.agentClient.SendRetryResponse(false)
				}
				return m, nil
			} else {
				// Exit the application
				return m, tea.Quit
			}

		case "enter":
			if m.showInput && !m.suggestionMode && !m.retryMode {
				userInput := strings.TrimSpace(m.input.Value())
				if userInput != "" && m.connected {
					m.addMessage(userInput, userMessage)
					m.input.SetValue("")

					err := m.agentClient.SendMessage(userInput)
					if err != nil {
						m.addMessage(fmt.Sprintf("Error sending message: %v", err), errorMessage)
					}
				}
				return m, tea.Batch(cmds...)
			}

		case "ctrl+enter":
			if m.showInput && !m.suggestionMode && !m.retryMode {
				userInput := strings.TrimSpace(m.input.Value())
				if userInput != "" && m.connected {
					m.addMessage(userInput, userMessage)
					m.input.SetValue("")

					err := m.agentClient.SendMessage(userInput)
					if err != nil {
						m.addMessage(fmt.Sprintf("Error sending message: %v", err), errorMessage)
					}
				}
				return m, tea.Batch(cmds...)
			}

		case "y", "Y":
			if m.suggestionMode && m.connected {
				m.suggestionMode = false
				m.showInput = true
				m.input.Focus()
				m.addMessage("Command approved and executing...", userMessage)

				err := m.agentClient.SendCommandApproval(true)
				if err != nil {
					m.addMessage(fmt.Sprintf("Error sending approval: %v", err), errorMessage)
				}
			} else if m.retryMode && m.connected {
				m.retryMode = false
				m.showInput = true
				m.input.Focus()
				m.addMessage("Retrying with a different approach...", userMessage)

				err := m.agentClient.SendRetryResponse(true)
				if err != nil {
					m.addMessage(fmt.Sprintf("Error sending retry response: %v", err), errorMessage)
				}
			}

		case "n", "N":
			if m.suggestionMode && m.connected {
				m.suggestionMode = false
				m.showInput = true
				m.input.Focus()
				m.addMessage("Command rejected", userMessage)

				err := m.agentClient.SendCommandApproval(false)
				if err != nil {
					m.addMessage(fmt.Sprintf("Error sending rejection: %v", err), errorMessage)
				}
			} else if m.retryMode && m.connected {
				m.retryMode = false
				m.showInput = true
				m.input.Focus()
				m.addMessage("Retry cancelled", userMessage)

				err := m.agentClient.SendRetryResponse(false)
				if err != nil {
					m.addMessage(fmt.Sprintf("Error sending retry response: %v", err), errorMessage)
				}
			}
		default:
			// Only update input for other keys (not enter/ctrl+enter)
			if m.showInput && !m.suggestionMode && !m.retryMode {
				m.input, cmd = m.input.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	default:
		// Update input for non-key messages
		if m.showInput && !m.suggestionMode && !m.retryMode {
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Always update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	m.updateViewportContent()

	return m, tea.Batch(cmds...)
}

// View renders the chat interface
func (m ChatModel) View() string {
	var sections []string

	connectionStatus := "⚫ Disconnected"
	if m.connected {
		connectionStatus = "🟢 Connected to cloud-assist"
	}

	chatView := m.viewport.View()

	var inputView string
	if m.suggestionMode {
		inputView = m.renderCommandSuggestion()
	} else if m.retryMode {
		inputView = m.renderRetryPrompt()
	} else if m.showInput {
		inputView = m.renderInput()
	}

	sections = append(sections, connectionStatus)
	sections = append(sections, chatView)
	sections = append(sections, inputView)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// addMessage adds a message to the chat
func (m *ChatModel) addMessage(content string, msgType messageType) {
	m.messages = append(m.messages, message{
		content: content,
		msgType: msgType,
	})
	m.updateViewportContent()
}

// addCommandSuggestion adds a command suggestion to the chat
func (m *ChatModel) addCommandSuggestion(command []string, explanation string) {
	m.messages = append(m.messages, message{
		content:     explanation,
		msgType:     commandSuggestion,
		command:     command,
		explanation: explanation,
	})
	m.showInput = false
	m.updateViewportContent()
}

// addRetryRequest adds a retry request to the chat
func (m *ChatModel) addRetryRequest(content string, retryCount int) {
	m.messages = append(m.messages, message{
		content:    content,
		msgType:    retryRequest,
		retryCount: retryCount,
	})
	m.showInput = false
	m.updateViewportContent()
}

// updateViewportContent updates the viewport with current messages
func (m *ChatModel) updateViewportContent() {
	var content strings.Builder

	for _, msg := range m.messages {
		switch msg.msgType {
		case userMessage:
			content.WriteString(m.formatUserMessage(msg.content))
		case agentMessage:
			content.WriteString(m.formatAgentMessage(msg.content))
		case commandSuggestion:
			content.WriteString(m.formatCommandSuggestion(msg.command, msg.explanation))
		case commandOutput:
			content.WriteString(m.formatCommandOutput(msg.content))
		case errorMessage:
			content.WriteString(m.formatErrorMessage(msg.content))
		case retryRequest:
			content.WriteString(m.formatRetryRequest(msg.content, msg.retryCount))
		}
		content.WriteString("\n\n")
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

// Message formatting methods
func (m *ChatModel) formatUserMessage(content string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff00")).
		Bold(true)
	return style.Render("You: ") + content
}

func (m *ChatModel) formatAgentMessage(content string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0099ff")).
		Bold(true)
	return style.Render("Agent: ") + content
}

func (m *ChatModel) formatCommandSuggestion(command []string, explanation string) string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffaa00")).
		Bold(true)
	commandStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	header := headerStyle.Render("Command Suggestion:")
	commandText := commandStyle.Render(strings.Join(command, " "))

	return fmt.Sprintf("%s\n%s\n%s", header, explanation, commandText)
}

func (m *ChatModel) formatCommandOutput(content string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true)
	return style.Render("Output: ") + content
}

func (m *ChatModel) formatErrorMessage(content string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff0000")).
		Bold(true)
	return style.Render("Error: ") + content
}

func (m *ChatModel) formatRetryRequest(content string, retryCount int) string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff6600")).
		Bold(true)

	header := headerStyle.Render(fmt.Sprintf("Retry Request (Attempt %d):", retryCount))
	return fmt.Sprintf("%s\n%s", header, content)
}

// renderInput renders the input area
func (m *ChatModel) renderInput() string {
	inputView := m.input.View()

	// Add instructions below the input
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true).
		Render("Press Enter to send • Ctrl+C to exit")

	return lipgloss.JoinVertical(lipgloss.Left, inputView, instructions)
}

// renderCommandSuggestion renders the command approval prompt
func (m *ChatModel) renderCommandSuggestion() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Foreground(lipgloss.Color("#ffaa00"))

	commandStr := strings.Join(m.currentCommand, " ")
	prompt := fmt.Sprintf("Execute command: %s\n\n%s\n\nApprove? (y/n)",
		commandStr, m.currentExplanation)

	return style.Render(prompt)
}

// renderRetryPrompt renders the retry request prompt
func (m *ChatModel) renderRetryPrompt() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Foreground(lipgloss.Color("#ff6600"))

	prompt := fmt.Sprintf("%s\n\nRetry with a different approach? (y/n)",
		m.currentRetryContent)

	return style.Render(prompt)
}
