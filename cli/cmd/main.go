package main

import (
	"cloud-assist/internal/auth"
	"cloud-assist/ui"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// AppModel is the main application model
type AppModel struct {
	currentScreen string
	loginModel    tea.Model
	chatModel     ui.ChatModel
	textInput     ui.TextInputModel
	multiline     ui.MultilineModel
	Select        ui.SelectModel
	confirmation  ui.ConfirmationModel
	statusBar     ui.StatusBarModel
	width         int
	height        int
	authenticated bool
}

// Messages for app state transitions
type authSuccessMsg struct{}
type showScreenMsg struct {
	screen string
}

// App state constants
const (
	screenLogin        = "login"
	screenChat         = "chat"
	screenTextInput    = "textInput"
	screenMultiline    = "multiline"
	screenSelect       = "select"
	screenConfirmation = "confirmation"
)

// NewAppModel creates a new application model
func NewAppModel() AppModel {
	// Create default initial models for all components
	loginModel := InitLoginModel()
	chatModel := ui.NewChatModel(100, 40)
	textInput := ui.NewTextInput("Sample Text Input", "Type something...", 30)
	multiline := ui.NewMultiline("Sample Multiline", "Type multiple lines...", 40, 10)

	// Create select items
	selectItems := []ui.SelectItem{
		{Title: "Chat Interface", Description: "Shows the chat UI", Value: screenChat},
		{Title: "Text Input", Description: "Shows a text input component", Value: screenTextInput},
		{Title: "Multiline Editor", Description: "Shows a multiline text editor", Value: screenMultiline},
		{Title: "Confirmation Dialog", Description: "Shows a confirmation dialog", Value: screenConfirmation},
	}
	selectModel := ui.NewSelect("Select a component to view", selectItems, 60, 10)

	// Create confirmation dialog
	confirmationModel := ui.NewConfirmation("Do you want to exit the application?")

	// Create status bar
	statusBar := ui.NewStatusBar(100)

	// Check if user is already authenticated
	initialScreen := screenLogin
	authenticated := false
	if _, err := auth.GetAPIKey(); err == nil {
		// API key exists, skip login and go directly to chat
		initialScreen = screenChat
		authenticated = true
	}

	return AppModel{
		currentScreen: initialScreen,
		authenticated: authenticated,
		loginModel:    loginModel,
		chatModel:     chatModel,
		textInput:     textInput,
		multiline:     multiline,
		Select:        selectModel,
		confirmation:  confirmationModel,
		statusBar:     statusBar,
	}
}

// InitLoginModel creates a new login model
func InitLoginModel() tea.Model {
	return ui.InitialLoginModel()
}

// Init initializes the application
func (m AppModel) Init() tea.Cmd {
	// Initialize all models to ensure they're ready for use
	cmds := []tea.Cmd{
		m.loginModel.Init(),
		m.chatModel.Init(),
		m.textInput.Init(),
		m.multiline.Init(),
		m.Select.Init(),
		m.confirmation.Init(),
		m.statusBar.Init(),
	}

	return tea.Batch(cmds...)
}

// Update handles app-wide updates
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update statusBar width
		m.statusBar.SetWidth(msg.Width)

	case authSuccessMsg:
		m.authenticated = true
		m.currentScreen = screenChat // Go directly to chat interface in production
		// Update status bar mode and status
		m.statusBar.SetMode("chat")
		m.statusBar.SetStatus("cloud-assist ready")
		return m, nil

	case showScreenMsg:
		m.currentScreen = msg.screen
		// Update status bar mode
		m.statusBar.SetMode(msg.screen)
		return m, nil
	}

	// Handle updates based on current screen
	switch m.currentScreen {
	case screenLogin:
		// Set status bar for login screen
		m.statusBar.SetMode("login")
		m.statusBar.SetStatus("please authenticate")
		m.statusBar.SetKeyBindings([]ui.KeyBinding{
			{Key: "enter", Description: "submit"},
			{Key: "ctrl+c", Description: "quit"},
		})

		newLoginModel, loginCmd := m.loginModel.Update(msg)
		m.loginModel = newLoginModel

		// Check if login was successful
		if loginModel, ok := m.loginModel.(ui.LoginModel); ok && loginModel.Authenticated() {
			// Save API key securely
			apiKey := loginModel.GetAPIKey()
			err := auth.SaveAPIKey(apiKey)
			if err != nil {
				fmt.Println("Error saving API key:", err)
			}
			// Transition to chat screen directly
			return m, func() tea.Msg { return authSuccessMsg{} }
		}

		cmd = loginCmd

	case screenSelect:
		// Set status bar for select screen
		m.statusBar.SetMode("select")
		m.statusBar.SetStatus("choose component")
		m.statusBar.SetKeyBindings([]ui.KeyBinding{
			{Key: "↑/↓", Description: "navigate"},
			{Key: "enter", Description: "select"},
			{Key: "ctrl+c", Description: "quit"},
		})

		newSelectModel, selectCmd := m.Select.Update(msg)
		m.Select = newSelectModel

		// Check if an item was selected
		if selected := m.Select.Selected(); selected != nil {
			screen, ok := selected.Value.(string)
			if ok {
				return m, func() tea.Msg { return showScreenMsg{screen: screen} }
			}
		}

		cmd = selectCmd

	case screenChat:
		// Set status bar for chat screen
		m.statusBar.SetMode("chat")
		m.statusBar.SetStatus("cloud-assist ready")
		m.statusBar.SetKeyBindings([]ui.KeyBinding{
			{Key: "enter", Description: "send"},
			{Key: "esc", Description: "back"},
			{Key: "ctrl+c", Description: "quit"},
		})

		newChatModel, chatCmd := m.chatModel.Update(msg)
		if updatedModel, ok := newChatModel.(ui.ChatModel); ok {
			m.chatModel = updatedModel
		}
		cmd = chatCmd

	case screenTextInput:
		// Set status bar for text input screen
		m.statusBar.SetMode("input")
		m.statusBar.SetStatus("enter text")
		m.statusBar.SetKeyBindings([]ui.KeyBinding{
			{Key: "enter", Description: "submit"},
			{Key: "esc", Description: "back"},
		})

		newTextInput, textInputCmd := m.textInput.Update(msg)
		m.textInput = newTextInput
		cmd = textInputCmd

	case screenMultiline:
		// Set status bar for multiline screen
		m.statusBar.SetMode("multiline")
		m.statusBar.SetStatus("edit text")
		m.statusBar.SetKeyBindings([]ui.KeyBinding{
			{Key: "ctrl+j", Description: "insert line"},
			{Key: "esc", Description: "back"},
		})

		newMultiline, multilineCmd := m.multiline.Update(msg)
		m.multiline = newMultiline
		cmd = multilineCmd

	case screenConfirmation:
		// Set status bar for confirmation screen
		m.statusBar.SetMode("confirm")
		m.statusBar.SetStatus("please confirm")
		m.statusBar.SetKeyBindings([]ui.KeyBinding{
			{Key: "←/→", Description: "navigate"},
			{Key: "enter", Description: "select"},
		})

		newConfirmation, confirmationCmd := m.confirmation.Update(msg)
		m.confirmation = newConfirmation

		// Check if confirmation dialog has a result
		if m.confirmation.HasResult() {
			result := m.confirmation.Result()
			if result != nil && *result {
				// User confirmed exit
				return m, tea.Quit
			} else {
				// Return to chat screen if canceled
				return m, func() tea.Msg { return showScreenMsg{screen: screenChat} }
			}
		}

		cmd = confirmationCmd
	}

	// Handle escape key globally
	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" && m.currentScreen != screenLogin {
		if m.currentScreen != screenChat {
			// Go back to chat screen from any other screen (except login)
			return m, func() tea.Msg { return showScreenMsg{screen: screenChat} }
		}
		// In chat screen, ESC does nothing special
	}

	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// View renders the app
func (m AppModel) View() string {
	var content string

	switch m.currentScreen {
	case screenLogin:
		content = m.loginModel.View()
	case screenSelect:
		content = "Cloud-Assist CLI\n\nSelect a component to view:\n\n" + m.Select.View() + "\n\nPress ESC to return to this menu from any component."
	case screenChat:
		content = m.chatModel.View()
	case screenTextInput:
		content = "Text Input Demo\n\n" + m.textInput.View() + "\n\nCurrent value: " + m.textInput.Value() + "\n\nPress ESC to go back."
	case screenMultiline:
		content = "Multiline Editor Demo\n\n" + m.multiline.View() + "\n\nCurrent content:\n" + m.multiline.Value() + "\n\nPress ESC to go back."
	case screenConfirmation:
		content = "Confirmation Dialog Demo\n\n" + m.confirmation.View()
	default:
		content = "Unknown screen"
	}

	// Add status bar at the bottom of the screen
	return content + "\n\n" + m.statusBar.View()
}

func main() {
	// Check if we have a saved API key
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		fmt.Println("Starting with login screen...")
	} else {
		fmt.Println("API key found:", apiKey[:4]+"...")
	}

	app := NewAppModel()
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
