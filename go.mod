module cloud-assist

go 1.21

require (
	github.com/charmbracelet/bubbles v0.16.1
	github.com/charmbracelet/bubbletea v0.24.2
	github.com/charmbracelet/lipgloss v0.9.1
	github.com/gorilla/websocket v1.5.1
)

// Additional dependencies will be listed here

// Use this if you want to keep the original import paths
replace cloud-assist/cli/client => ./cli/client
replace cloud-assist/internal/mock => ./internal/mock
