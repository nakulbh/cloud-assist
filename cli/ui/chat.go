package ui

import (
	"cloud-assist/internal/mock"
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
)

type message struct {
	content string
	msgType messageType
}

// ChatModel represents the chat interface
type ChatModel struct {
	messages       []message
	viewport       viewport.Model
	input          MultilineModel
	width          int
	height         int
	showInput      bool
	suggestionMode bool
	currentCommand string
	agentService   *mock.AgentService
}

// NewChatModel creates a new chat model
func NewChatModel(width, height int) ChatModel {
	input := NewMultiline("", "What would you like to do?", width-4, 5)
	vp := viewport.New(width, height-10) // Leave space for input
	vp.Style = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(1).Border(lipgloss.NormalBorder(), false, true)

	// Initialize the agent service
	agentService := mock.NewAgentService()

	// Process the first message from agent
	initialMessages := agentService.ProcessUserMessage("help")

	// Create the chat model
	model := ChatModel{
		messages:       []message{},
		viewport:       vp,
		input:          input,
		width:          width,
		height:         height,
		showInput:      true,
		suggestionMode: false,
		agentService:   agentService,
	}

	// Add initial messages from agent
	for _, msg := range initialMessages {
		var msgType messageType
		switch msg.Type {
		case mock.TypeAgent:
			msgType = agentMessage
		case mock.TypeCommand:
			msgType = commandSuggestion
			model.currentCommand = msg.Content
			model.suggestionMode = true
		case mock.TypeCommandOutput:
			msgType = commandOutput
		case mock.TypeError:
			msgType = errorMessage
		}

		model.messages = append(model.messages, message{
			content: msg.Content,
			msgType: msgType,
		})
	}

	model.updateViewport()
	return model
}

// Init initializes the chat model
func (m ChatModel) Init() tea.Cmd {
	return nil
}

// Update handles updates to the chat model
func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Handle exit with Ctrl+C
			return m, tea.Quit

		case "enter":
			if m.suggestionMode {
				// Handle command approval
				return m.handleCommandApproval("y")
			} else {
				// Handle user input
				return m.handleUserInput()
			}

		case "y":
			if m.suggestionMode {
				// Execute suggested command
				return m.handleCommandApproval("y")
			}

		case "n":
			if m.suggestionMode {
				// Skip suggested command
				return m.handleCommandApproval("n")
			}

		case "e":
			if m.suggestionMode {
				// Explain suggested command
				return m.handleCommandApproval("e")
			}

		case "q":
			if m.suggestionMode {
				// Quit command suggestion mode
				m.suggestionMode = false
				m.showInput = true
				m.messages = append(m.messages, message{
					content: "Command skipped.",
					msgType: agentMessage,
				})
				m.updateViewport()
				return m, nil
			}
		}
	}

	// Handle viewport updates
	newViewport, vpCmd := m.viewport.Update(msg)
	m.viewport = newViewport
	cmds = append(cmds, vpCmd)

	// Handle input updates if input is shown
	if m.showInput {
		newInput, inputCmd := m.input.Update(msg)
		m.input = newInput
		cmds = append(cmds, inputCmd)
	}

	return m, tea.Batch(cmds...)
}

// handleUserInput processes user input and returns agent responses
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

	// Process the message with the agent service
	agentResponses := m.agentService.ProcessUserMessage(userInput)

	// Add agent responses to chat
	for _, resp := range agentResponses {
		var msgType messageType
		switch resp.Type {
		case mock.TypeAgent:
			msgType = agentMessage
		case mock.TypeCommand:
			msgType = commandSuggestion
			m.currentCommand = resp.Content
			m.suggestionMode = true
			m.showInput = false
		case mock.TypeCommandOutput:
			msgType = commandOutput
		case mock.TypeError:
			msgType = errorMessage
		}

		m.messages = append(m.messages, message{
			content: resp.Content,
			msgType: msgType,
		})
	}

	m.updateViewport()
	return m, nil
}

// handleCommandApproval processes command approval responses
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

	// Process the response with the agent service
	var agentResponses []mock.AgentMessage
	if response == "y" {
		agentResponses = m.agentService.ExecuteSuggestedCommand()
	} else if response == "e" {
		agentResponses = m.agentService.ProcessUserMessage("e")
		// Keep suggestion mode active after explanation
		m.suggestionMode = true
		m.showInput = false
	} else {
		// For "n" response, just skip this command
		m.suggestionMode = false
		m.showInput = true
		m.messages = append(m.messages, message{
			content: "Command skipped. What would you like to do instead?",
			msgType: agentMessage,
		})
		m.updateViewport()
		return m, nil
	}

	// Add agent responses to chat
	for _, resp := range agentResponses {
		var msgType messageType
		switch resp.Type {
		case mock.TypeAgent:
			msgType = agentMessage
		case mock.TypeCommand:
			msgType = commandSuggestion
			m.currentCommand = resp.Content
			m.suggestionMode = true
			m.showInput = false
		case mock.TypeCommandOutput:
			msgType = commandOutput
		case mock.TypeError:
			msgType = errorMessage
		}

		m.messages = append(m.messages, message{
			content: resp.Content,
			msgType: msgType,
		})
	}

	m.updateViewport()
	return m, nil
}

// updateViewport updates the viewport content
func (m *ChatModel) updateViewport() {
	var viewportContent strings.Builder

	for i, msg := range m.messages {
		// Add a newline between messages
		if i > 0 {
			viewportContent.WriteString("\n\n")
		}

		switch msg.msgType {
		case userMessage:
			viewportContent.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("5")).
				Render("You: "))
			viewportContent.WriteString(msg.content)

		case agentMessage:
			viewportContent.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("2")).
				Bold(true).
				Render("Cloud-Assist: "))
			viewportContent.WriteString(msg.content)

		case commandSuggestion:
			// Center the command suggestion with box styling
			viewportContent.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("3")).
				Bold(true).
				Align(lipgloss.Center).
				Width(m.width - 10).
				Render("Suggested command:"))

			viewportContent.WriteString("\n")

			// Command box with improved styling
			viewportContent.WriteString(lipgloss.NewStyle().
				Align(lipgloss.Center).
				Width(m.width - 10).
				Render(
					lipgloss.NewStyle().
						Background(lipgloss.Color("8")).
						Foreground(lipgloss.Color("15")).
						Padding(1, 2).
						Border(lipgloss.RoundedBorder()).
						BorderForeground(lipgloss.Color("12")).
						Width(m.width / 2).
						Align(lipgloss.Center).
						Render(msg.content),
				))

			viewportContent.WriteString("\n\n")

			// Center the options with improved styling
			viewportContent.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("6")).
				Align(lipgloss.Center).
				Width(m.width - 10).
				Render("[y] Execute  [n] Skip  [e] Explain  [q] Quit"))

		case commandOutput:
			// Add header for output with box styling
			viewportContent.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Bold(true).
				Render("Output:"))

			viewportContent.WriteString("\n")

			// Output box with improved styling
			viewportContent.WriteString(lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("8")).
				Padding(0, 1).
				Width(m.width - 10).
				Render(
					lipgloss.NewStyle().
						Foreground(lipgloss.Color("7")).
						Render(msg.content),
				))

		case errorMessage:
			viewportContent.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color("1")).
				Bold(true).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("1")).
				Padding(0, 1).
				Width(m.width - 10).
				Render("Error: " + msg.content))
		}
	}

	m.viewport.SetContent(viewportContent.String())
	m.viewport.GotoBottom()
}

// View renders the chat model
func (m ChatModel) View() string {
	var view strings.Builder

	// Create a centered container for the entire view
	mainStyle := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center)

	// Chat history with improved styling
	view.WriteString(mainStyle.Render(m.viewport.View()) + "\n\n")

	// Input header with consistent styling
	inputHeader := "What would you like to help with?"
	if m.suggestionMode {
		inputHeader = "Command suggestion active. Press y to execute, n to skip, e to explain, or q to quit."
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true).
		Width(m.width - 10).
		Align(lipgloss.Center)

	view.WriteString(mainStyle.Render(headerStyle.Render(inputHeader)))
	view.WriteString("\n")

	// Render input with improved styling
	if m.showInput {
		// Create a consistent input box style
		inputBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")).
			Padding(1, 2).
			Width(m.width - 20)

		// Wrap the input in the styled box
		inputContent := inputBoxStyle.Render(m.input.View())
		view.WriteString(mainStyle.Render(inputContent))
	}

	return view.String()
}
