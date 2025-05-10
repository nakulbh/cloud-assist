// filepath: /home/nakulbh/Desktop/Projects/PersonalProjects/cloud-assist/cli/ui/statusbar.go
package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusBarModel represents a status bar component
type StatusBarModel struct {
	mode        string
	contextSize int
	status      string
	keyBindings []KeyBinding
	clock       bool
	width       int
	style       lipgloss.Style
}

// KeyBinding represents a keyboard shortcut and its description
type KeyBinding struct {
	Key         string
	Description string
}

// NewStatusBar creates a new status bar component
func NewStatusBar(width int) StatusBarModel {
	return StatusBarModel{
		mode:        "normal",
		contextSize: 0,
		status:      "ready",
		keyBindings: []KeyBinding{
			{Key: "esc", Description: "back"},
			{Key: "ctrl+c", Description: "quit"},
		},
		clock: true,
		width: width,
		style: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("0")).
			Bold(true).
			Width(width).
			Padding(0, 1),
	}
}

// SetMode sets the current mode
func (m *StatusBarModel) SetMode(mode string) {
	m.mode = mode
}

// SetContextSize sets the current context size
func (m *StatusBarModel) SetContextSize(size int) {
	m.contextSize = size
}

// SetStatus sets the current status
func (m *StatusBarModel) SetStatus(status string) {
	m.status = status
}

// SetKeyBindings sets the displayed keyboard shortcuts
func (m *StatusBarModel) SetKeyBindings(keyBindings []KeyBinding) {
	m.keyBindings = keyBindings
}

// SetWidth sets the width of the status bar
func (m *StatusBarModel) SetWidth(width int) {
	m.width = width
	m.style = m.style.Width(width)
}

// EnableClock enables or disables the clock
func (m *StatusBarModel) EnableClock(enabled bool) {
	m.clock = enabled
}

// WithStyle sets the style of the status bar
func (m StatusBarModel) WithStyle(style lipgloss.Style) StatusBarModel {
	m.style = style.Width(m.width)
	return m
}

// Init initializes the status bar
func (m StatusBarModel) Init() tea.Cmd {
	return nil
}

// Update handles updates to the status bar
func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
	return m, nil
}

// View renders the status bar
func (m StatusBarModel) View() string {
	var leftSections []string
	var rightSections []string

	// Left sections: mode and status
	modeSection := fmt.Sprintf("[%s]", strings.ToUpper(m.mode))
	leftSections = append(leftSections, modeSection)

	if m.status != "" {
		statusSection := fmt.Sprintf("%s", m.status)
		leftSections = append(leftSections, statusSection)
	}

	if m.contextSize > 0 {
		contextSection := fmt.Sprintf("ctx:%d", m.contextSize)
		leftSections = append(leftSections, contextSection)
	}

	// Right sections: key bindings and clock
	for _, kb := range m.keyBindings {
		keySection := fmt.Sprintf("%s:%s", kb.Key, kb.Description)
		rightSections = append(rightSections, keySection)
	}

	if m.clock {
		clockSection := time.Now().Format("15:04:05")
		rightSections = append(rightSections, clockSection)
	}

	// Combine left and right sections with proper spacing
	leftPart := strings.Join(leftSections, " | ")
	rightPart := strings.Join(rightSections, " • ")

	// Calculate spacing between left and right parts
	totalContentLength := len(leftPart) + len(rightPart)
	spacingLength := m.width - totalContentLength - 2 // -2 for padding
	if spacingLength < 1 {
		spacingLength = 1
	}
	spacing := strings.Repeat(" ", spacingLength)

	return m.style.Render(leftPart + spacing + rightPart)
}
