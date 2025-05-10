package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectItem represents an item in a select input
type SelectItem struct {
	Title       string
	Description string
	Value       interface{}
}

// FilterValue implements the list.Item interface
func (i SelectItem) FilterValue() string { return i.Title }

// SelectModel represents a select input component
type SelectModel struct {
	list     list.Model
	label    string
	selected *SelectItem
	style    lipgloss.Style
}

// NewSelect creates a new select input component
func NewSelect(label string, items []SelectItem, width, height int) SelectModel {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	l := list.New(listItems, list.NewDefaultDelegate(), width, height)
	l.Title = label
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().MarginLeft(2).Bold(true)

	return SelectModel{
		list:  l,
		label: label,
		style: lipgloss.NewStyle().BorderForeground(lipgloss.Color("62")).BorderStyle(lipgloss.RoundedBorder()),
	}
}

// Init initializes the select input component
func (m SelectModel) Init() tea.Cmd {
	return nil
}

// Update handles updates to the select input component
func (m SelectModel) Update(msg tea.Msg) (SelectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.list.SelectedItem().(SelectItem); ok {
				m.selected = &i
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the select input component
func (m SelectModel) View() string {
	return m.style.Render(m.list.View())
}

// Selected returns the currently selected item
func (m SelectModel) Selected() *SelectItem {
	return m.selected
}

// SetItems sets the items in the select input
func (m *SelectModel) SetItems(items []SelectItem) {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	m.list.SetItems(listItems)
}

// SetFilter sets the filter for the select input
func (m *SelectModel) SetFilter(filter string) {
	m.list.SetFilterText(filter)
}
