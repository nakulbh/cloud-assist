package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TextInputModel represents a text input component
type TextInputModel struct {
	textInput textinput.Model
	label     string
	width     int
	style     lipgloss.Style
}

// NewTextInput creates a new text input component
func NewTextInput(label string, placeholder string, width int) TextInputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 150
	ti.Width = width

	return TextInputModel{
		textInput: ti,
		label:     label,
		width:     width,
		style:     lipgloss.NewStyle().BorderForeground(lipgloss.Color("62")).BorderStyle(lipgloss.RoundedBorder()),
	}
}

// Init initializes the text input component
func (m TextInputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles updates to the text input component
func (m TextInputModel) Update(msg tea.Msg) (TextInputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the text input component
func (m TextInputModel) View() string {
	return m.style.Render(m.label + "\n" + m.textInput.View())
}

// Value returns the current value of the text input
func (m TextInputModel) Value() string {
	return m.textInput.Value()
}

// SetValue sets the value of the text input
func (m TextInputModel) SetValue(value string) {
	m.textInput.SetValue(value)
}

// Focus focuses the text input
func (m *TextInputModel) Focus() {
	m.textInput.Focus()
}

// Blur removes focus from the text input
func (m *TextInputModel) Blur() {
	m.textInput.Blur()
}
