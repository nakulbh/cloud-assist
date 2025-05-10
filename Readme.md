# Cloud-Assist: DevOps Terminal Agent

![Version](https://img.shields.io/badge/version-0.1.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

A powerful terminal-based DevOps agent that helps automate common cloud and infrastructure tasks through an intuitive command-line interface. Cloud-Assist uses AI to understand intent, suggest commands, and learn from execution patterns to streamline complex DevOps workflows.

![Cloud-Assist Demo](https://example.com/cloud-assist-demo.gif)

## Features

- **Conversational Interface**: Describe what you want to accomplish in plain language
- **Command Suggestions**: Receive contextually appropriate shell commands
- **Command Approval**: Review and approve commands before execution
- **Output Analysis**: AI analyzes command output to determine next steps
- **Persistent Context**: Maintains understanding of your infrastructure throughout a session
- **Security-Focused**: All commands require explicit approval by default

## Use Cases

- Setting up and configuring cloud resources
- Troubleshooting infrastructure issues
- Deploying applications to various environments
- Managing Kubernetes clusters
- Configuring networking and security
- Automating repetitive DevOps tasks

## Installation

### Prerequisites

- Go 1.24 or higher
- Access to an AI model API (OpenAI, Anthropic, etc.)

### From Source

```bash
git clone https://github.com/yourusername/cloud-assist.git
cd cloud-assist/cli
go build -o cloud-assist cmd/main.go
```

### Using Go Install

```bash
go install github.com/yourusername/cloud-assist/cli/cmd@latest
```

## Getting Started

### First Run

1. Start a new session:

```bash
cloud-assist
```

2. Enter your API key when prompted (this will be securely stored for future sessions)

### Basic Usage

1. Start a new session with a task description:

```bash
cloud-assist "setup a monitoring stack for my web service"
```

2. Review and approve suggested commands
3. See results and next suggestions
4. Continue until your task is complete

## Command Approval

Cloud-Assist never executes commands without your approval. For each suggested command, you have the following options:

- **Yes (y)**: Execute the command
- **No (n)**: Skip this command and try a different approach
- **Explain (e)**: Request more detailed explanation before deciding
- **Quit (q)**: Exit the current session

## Configuration

Cloud-Assist can be configured through environment variables or a config file at `~/.config/cloud-assist/config.json`.

Key configuration options:
- `CLOUD_ASSIST_MODEL`: AI model to use
- `CLOUD_ASSIST_APPROVAL_MODE`: Default approval behavior (suggest, auto, full-auto)
- `CLOUD_ASSIST_HISTORY_FILE`: Where to store command history

## Example Workflows

### Setting Up Kubernetes Monitoring

```
User: "I need to set up monitoring for my Kubernetes cluster"

Cloud-Assist: "I'll help you set up monitoring for your Kubernetes cluster. First, let's check if you have Helm installed:

Suggested command:
helm version

[y] Execute  [n] Skip  [e] Explain  [q] Quit"

User: y

[Command output shows Helm is installed]

Cloud-Assist: "Great! Now let's add the Prometheus community Helm chart repository..."
```

### Troubleshooting a Web Service

```
User: "My nginx web server is returning 503 errors. Help me troubleshoot."

Cloud-Assist: "I'll help you troubleshoot the 503 errors from your nginx server. First, let's check if nginx is running:

Suggested command:
systemctl status nginx

[y] Execute  [n] Skip  [e] Explain  [q] Quit"
```

## Architecture

Cloud-Assist is built using Go with the following components:

- **UI Layer**: Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **API Integration**: Secure communication with AI model providers
- **Command Execution**: Safe execution and output capture for shell commands
- **Context Management**: Tracking session state and command history
- **Model Context Protocol (MCP)**: Standardized communication with AI tools

## Development

### Project Structure

```
cli/
  ├── cmd/           # Application entrypoint
  ├── internal/      # Private application logic
  └── ui/            # User interface components
documents/          # Documentation and specifications
```

### Building and Testing

```bash
cd cli
go test ./...
go build -o cloud-assist cmd/main.go
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) for terminal styling

---

Built with ❤️ for DevOps engineers who value both automation and control.