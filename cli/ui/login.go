package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// LoginModel represents a login screen
type LoginModel struct {
	input         textinput.Model
	err           string
	authenticated bool
	apiKey        string
}

// InitialLoginModel creates a new login model
func InitialLoginModel() tea.Model {
	ti := textinput.New()
	ti.Placeholder = "Enter API Key"
	ti.Focus()
	ti.CharLimit = 64
	ti.Width = 30
	return LoginModel{input: ti}
}

func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC: // Handle Ctrl+C
			return m, tea.Quit
		case tea.KeyEnter:
			// For demo purpose, accept any non-empty API key
			if m.input.Value() != "" {
				m.authenticated = true
				m.apiKey = m.input.Value()
				return m, tea.Quit
			} else {
				m.err = "API key cannot be empty"
				m.input.SetValue("")
			}
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m LoginModel) View() string {
	view := fmt.Sprintf("🔐 %s\n\n%s\n\nPress Enter to submit", m.input.Placeholder, m.input.View())
	if m.err != "" {
		view += fmt.Sprintf("\n\n❌ %s", m.err)
	}
	return view
}

// Authenticated returns whether the user is authenticated
func (m LoginModel) Authenticated() bool {
	return m.authenticated
}

// GetAPIKey returns the user's API key
func (m LoginModel) GetAPIKey() string {
	return m.apiKey
}
