package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MultilineModel represents a multiline text editor component
type MultilineModel struct {
	textarea textarea.Model
	label    string
	width    int
	height   int
	style    lipgloss.Style
}

// NewMultiline creates a new multiline text editor component
func NewMultiline(label string, placeholder string, width, height int) MultilineModel {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.Focus()
	ta.CharLimit = 1000
	ta.SetWidth(width)
	ta.SetHeight(height)
	ta.ShowLineNumbers = true

	return MultilineModel{
		textarea: ta,
		label:    label,
		width:    width,
		height:   height,
		style:    lipgloss.NewStyle().BorderForeground(lipgloss.Color("62")).BorderStyle(lipgloss.RoundedBorder()),
	}
}

// Init initializes the multiline text editor component
func (m MultilineModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles updates to the multiline text editor component
func (m MultilineModel) Update(msg tea.Msg) (MultilineModel, tea.Cmd) {
	var cmd tea.Cmd

	// Check for special keys before passing to textarea
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "shift+enter":
			// Allow Shift+Enter to add new lines - convert to regular enter for textarea
			enterMsg := tea.KeyMsg{
				Type:  tea.KeyEnter,
				Runes: []rune{'\n'},
			}
			m.textarea, cmd = m.textarea.Update(enterMsg)
			return m, cmd
		case "enter", "ctrl+enter":
			// Don't pass Enter or Ctrl+Enter to textarea, let parent handle them
			// But still update textarea for other keys
			return m, nil
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the multiline text editor component
func (m MultilineModel) View() string {
	return m.style.Render(m.label + "\n" + m.textarea.View())
}

// Value returns the current value of the multiline text editor
func (m MultilineModel) Value() string {
	return m.textarea.Value()
}

// SetValue sets the value of the multiline text editor
func (m MultilineModel) SetValue(value string) {
	m.textarea.SetValue(value)
}

// SetWidth sets the width of the multiline text editor
func (m *MultilineModel) SetWidth(width int) {
	m.width = width
	m.textarea.SetWidth(width)
}

// Focus focuses the multiline text editor
func (m *MultilineModel) Focus() {
	m.textarea.Focus()
}

// Blur removes focus from the multiline text editor
func (m *MultilineModel) Blur() {
	m.textarea.Blur()
}

// Lines returns the lines of text in the multiline text editor
func (m MultilineModel) Lines() []string {
	return strings.Split(m.textarea.Value(), "\n")
}
