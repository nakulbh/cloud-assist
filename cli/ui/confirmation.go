package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmationModel represents a confirmation dialog component
type ConfirmationModel struct {
	question   string
	yesText    string
	noText     string
	yesStyle   lipgloss.Style
	noStyle    lipgloss.Style
	focusStyle lipgloss.Style
	selected   bool // true for yes, false for no
	result     *bool
	style      lipgloss.Style
}

// NewConfirmation creates a new confirmation dialog component
func NewConfirmation(question string) ConfirmationModel {
	return ConfirmationModel{
		question:   question,
		yesText:    "Yes",
		noText:     "No",
		yesStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("10")),  // Green
		noStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("9")),   // Red
		focusStyle: lipgloss.NewStyle().Background(lipgloss.Color("240")), // Dark gray
		selected:   true,                                                  // Default to "Yes"
		style:      lipgloss.NewStyle().BorderForeground(lipgloss.Color("62")).BorderStyle(lipgloss.RoundedBorder()).Padding(1),
	}
}

// WithYesText sets the text for the "yes" option
func (m ConfirmationModel) WithYesText(text string) ConfirmationModel {
	m.yesText = text
	return m
}

// WithNoText sets the text for the "no" option
func (m ConfirmationModel) WithNoText(text string) ConfirmationModel {
	m.noText = text
	return m
}

// Init initializes the confirmation dialog component
func (m ConfirmationModel) Init() tea.Cmd {
	return nil
}

// Update handles updates to the confirmation dialog component
func (m ConfirmationModel) Update(msg tea.Msg) (ConfirmationModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.selected = true
		case "right", "l":
			m.selected = false
		case "enter":
			result := m.selected
			m.result = &result
			return m, nil
		case "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the confirmation dialog component
func (m ConfirmationModel) View() string {
	var yesButton, noButton string

	if m.selected {
		yesButton = m.focusStyle.Render(m.yesStyle.Render(" " + m.yesText + " "))
		noButton = m.noStyle.Render(" " + m.noText + " ")
	} else {
		yesButton = m.yesStyle.Render(" " + m.yesText + " ")
		noButton = m.focusStyle.Render(m.noStyle.Render(" " + m.noText + " "))
	}

	return m.style.Render(
		m.question + "\n\n" +
			"  " + yesButton + "  " + noButton + "\n\n" +
			"  ← / → to navigate • enter to select",
	)
}

// Result returns the result of the confirmation dialog
func (m ConfirmationModel) Result() *bool {
	return m.result
}

// HasResult returns true if the confirmation dialog has a result
func (m ConfirmationModel) HasResult() bool {
	return m.result != nil
}
