package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerModel represents a spinner component
type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	style    lipgloss.Style
	isActive bool
}

// NewSpinner creates a new spinner component
func NewSpinner(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return SpinnerModel{
		spinner:  s,
		message:  message,
		style:    lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		isActive: true,
	}
}

// WithSpinnerType sets the spinner type
func (m SpinnerModel) WithSpinnerType(spinnerType spinner.Spinner) SpinnerModel {
	m.spinner.Spinner = spinnerType
	return m
}

// WithColor sets the spinner color
func (m SpinnerModel) WithColor(color string) SpinnerModel {
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	m.style = lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return m
}

// Init initializes the spinner component
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles updates to the spinner component
func (m SpinnerModel) Update(msg tea.Msg) (SpinnerModel, tea.Cmd) {
	if !m.isActive {
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// View renders the spinner component
func (m SpinnerModel) View() string {
	if !m.isActive {
		return ""
	}
	return m.spinner.View() + " " + m.style.Render(m.message)
}

// Start starts the spinner
func (m *SpinnerModel) Start() {
	m.isActive = true
}

// Stop stops the spinner
func (m *SpinnerModel) Stop() {
	m.isActive = false
}

// SetMessage sets the spinner message
func (m *SpinnerModel) SetMessage(message string) {
	m.message = message
}

// TickMsg is a message that is sent when the spinner should tick
type TickMsg time.Time

// Tick is a command that sends a tick message
func Tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}
