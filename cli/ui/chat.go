package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ChatModel represents a chat interface
type ChatModel struct {
	viewport     viewport.Model
	messages     []Message
	input        MultilineModel
	spinner      SpinnerModel
	width        int
	height       int
	ready        bool
	isProcessing bool
}

// Message represents a chat message
type Message struct {
	Content   string
	IsUser    bool
	Timestamp time.Time
}

// NewChatModel creates a new chat model
func NewChatModel(width, height int) ChatModel {
	input := NewMultiline("", "Type your message here...", width, 5)
	spinner := NewSpinner("Processing...").WithColor("205")
	spinner.Stop()

	// Initialize the viewport with default dimensions
	vp := viewport.New(width, height-10)
	vp.SetContent("")

	return ChatModel{
		viewport:     vp,
		messages:     []Message{},
		input:        input,
		spinner:      spinner,
		width:        width,
		height:       height,
		ready:        true, // Set ready to true since viewport is initialized
		isProcessing: false,
	}
}

// Init initializes the chat model
func (m ChatModel) Init() tea.Cmd {
	return tea.Batch(
		m.input.Init(),
		m.spinner.Init(),
	)
}

// Update handles updates to the chat model
func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if !m.isProcessing && msg.Type == tea.KeyEnter && !msg.Alt {
				// Submit message
				content := strings.TrimSpace(m.input.Value())
				if content != "" {
					m.messages = append(m.messages, Message{
						Content:   content,
						IsUser:    true,
						Timestamp: time.Now(),
					})
					m.input.SetValue("")
					m.isProcessing = true
					m.spinner.Start()

					// Simulate AI response after a delay
					return m, tea.Batch(
						tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
							return aiResponseMsg{
								content: "This is a simulated AI response to: " + content,
							}
						}),
					)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			// Initialize viewport
			m.viewport = viewport.New(m.width, m.height-10) // Leave room for input
			m.viewport.SetContent(m.renderMessages())
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = m.height - 10
		}

		// Also update width of input
		m.input.width = m.width

	case aiResponseMsg:
		m.isProcessing = false
		m.spinner.Stop()
		m.messages = append(m.messages, Message{
			Content:   msg.content,
			IsUser:    false,
			Timestamp: time.Now(),
		})
		m.viewport.SetContent(m.renderMessages())
		cmds = append(cmds, viewport.Sync(m.viewport))
	}

	// Handle viewport updates
	if m.ready {
		m.viewport.SetContent(m.renderMessages())
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Handle input updates
	if !m.isProcessing {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Handle spinner updates
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the chat model
func (m ChatModel) View() string {
	if !m.ready {
		return "Initializing..."
	}
	if m.isProcessing {
		return m.spinner.View()
	}

	viewportView := m.viewport.View()
	inputView := m.input.View()
	spinnerView := ""
	if m.isProcessing {
		spinnerView = "\n" + m.spinner.View()
	}

	// Combine the views
	return fmt.Sprintf("%s\n\n%s%s", viewportView, inputView, spinnerView)
}

// renderMessages renders all messages
func (m ChatModel) renderMessages() string {
	var sb strings.Builder

	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true) // Blue
	aiStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)   // Green
	contentStyle := lipgloss.NewStyle().PaddingLeft(2)
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

	for _, msg := range m.messages {
		// Format timestamp
		timestamp := msg.Timestamp.Format("15:04:05")

		// Add sender and timestamp
		if msg.IsUser {
			sb.WriteString(userStyle.Render("You") + " " + timeStyle.Render(timestamp) + "\n")
		} else {
			sb.WriteString(aiStyle.Render("AI") + " " + timeStyle.Render(timestamp) + "\n")
		}

		// Add content
		sb.WriteString(contentStyle.Render(msg.Content) + "\n\n")
	}

	return sb.String()
}

// aiResponseMsg is a message containing an AI response
type aiResponseMsg struct {
	content string
}
