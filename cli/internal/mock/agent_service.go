// Package mock provides mock implementations for testing UI components
package mock

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// AgentMessageType defines the type of message in the conversation
type AgentMessageType string

const (
	// TypeUser represents a message from the user
	TypeUser AgentMessageType = "user"
	// TypeAgent represents a message from the agent
	TypeAgent AgentMessageType = "agent"
	// TypeCommand represents a command suggestion
	TypeCommand AgentMessageType = "command"
	// TypeCommandOutput represents the output of an executed command
	TypeCommandOutput AgentMessageType = "command_output"
	// TypeError represents an error message
	TypeError AgentMessageType = "error"
)

// AgentMessage represents a message in the conversation
type AgentMessage struct {
	Type    AgentMessageType
	Content string
	Time    time.Time
}

// AgentService simulates the Cloud-Assist agent for UI testing
type AgentService struct {
	dockerService      *DockerCommandService
	conversationState  string
	previousCommand    string
	welcomeMessages    []string
	responseTemplates  map[string][]string
	explanationMethods map[string]func(string) string
	contextHistory     []string
	scenario           string
	userPreferences    map[string]string
	scenarioProgress   int
}

// NewAgentService creates a new mock agent service
func NewAgentService() *AgentService {
	agent := &AgentService{
		dockerService:     NewDockerCommandService(),
		conversationState: "initial",
		welcomeMessages: []string{
			"Welcome to Cloud-Assist! I'm your AI-powered DevOps assistant. How can I help you today?",
			"Hi there! I'm Cloud-Assist, your AI DevOps agent. What would you like to accomplish?",
			"Hello! I'm your Cloud-Assist AI agent, ready to help with your infrastructure needs. What are you working on?",
		},
		responseTemplates:  make(map[string][]string),
		explanationMethods: make(map[string]func(string) string),
		contextHistory:     []string{},
		scenario:           "docker",
		userPreferences:    make(map[string]string),
		scenarioProgress:   0,
	}

	// Add response templates for different contexts - make these more AI-like
	agent.responseTemplates["initial"] = []string{
		"I can help you manage your Docker containers. Let me first check what containers are currently running:",
		"Let's start by getting an overview of your Docker environment. I'll check your running containers:",
		"I'll assist you with Docker operations. First, let me see what containers you currently have:",
	}

	agent.responseTemplates["after_docker_ps"] = []string{
		"I see you have %d containers running. Based on your environment, would you like to examine logs for any specific container or address any particular issues?",
		"Your environment has %d running containers. I notice a mix of services - is there a specific container you're interested in working with?",
		"I've found %d active containers in your environment. Would you like me to help troubleshoot any of them or perform specific management tasks?",
	}

	agent.responseTemplates["after_logs"] = []string{
		"I've analyzed the logs and notice %s. Would you like me to suggest a solution or perform another operation on this container?",
		"Based on these logs, I can see %s. What would you like me to help you with next?",
		"The logs show %s. Based on this information, I could help you restart the service or investigate further. What would you prefer?",
	}

	agent.responseTemplates["after_network"] = []string{
		"I've analyzed your network configuration and found %d connected containers. I notice that the 'app' container isn't connected to the shared network, which might explain the connectivity issues.",
		"Your network setup shows %d containers on the shared network. There might be networking issues with containers that aren't properly connected.",
		"Looking at your network configuration, I can see %d containers are connected. Would you like me to help connect other containers to resolve communication issues?",
	}

	agent.responseTemplates["after_start"] = []string{
		"I've successfully started the container. Let me check to confirm it's running properly and check for any potential issues in the startup logs:",
		"The container is now running. To ensure everything is working correctly, I should examine if it's connected to the necessary networks and check its logs:",
		"Container started successfully. Based on your environment, I recommend verifying that it can communicate with other services. Let me check its network configuration:",
	}

	// Enhancement: Add specific log analysis patterns
	agent.responseTemplates["log_analysis"] = []string{
		"connections to the Redis cache failing",
		"normal startup sequence with no errors",
		"several HTTP 404 errors that might need attention",
		"potential memory issues based on resource utilization patterns",
	}

	// Enhancement: Add network analysis patterns
	agent.responseTemplates["network_analysis"] = []string{
		"possible container isolation issues",
		"a misconfigured DNS resolution between services",
		"proper network connectivity between key services",
		"potential firewall or security group restrictions",
	}

	// Add explanation methods for different commands - make these more AI-like and educational
	agent.explanationMethods["docker ps"] = func(cmd string) string {
		return "The `docker ps` command lists all running containers on your system. It displays container IDs, " +
			"the image used, when they were created, their current status, exposed ports, and assigned names. This gives you " +
			"a quick overview of what's active in your Docker environment.\n\n" +
			"I recommended this command because it's essential to understand what's currently running before taking further actions."
	}

	agent.explanationMethods["docker ps -a"] = func(cmd string) string {
		return "The `docker ps -a` command shows all containers on your system, including those that have stopped or exited. " +
			"The `-a` flag stands for 'all' and provides a complete view of your container environment.\n\n" +
			"This command is particularly useful when troubleshooting because it shows containers that may have crashed or " +
			"exited unexpectedly, along with their exit codes which can help diagnose issues."
	}

	agent.explanationMethods["docker logs"] = func(cmd string) string {
		parts := strings.Split(cmd, " ")
		if len(parts) > 2 {
			containerName := parts[2]
			return fmt.Sprintf("The `docker logs %s` command fetches and displays the logs generated by the '%s' container. "+
				"This includes both stdout and stderr output streams.\n\n"+
				"I recommended checking the logs because they often contain valuable diagnostic information that can help "+
				"identify why a container is behaving unexpectedly or what errors it might be encountering.", containerName, containerName)
		}
		return "The `docker logs` command displays the logs from a specified container, showing its stdout and stderr output streams. " +
			"This is crucial for debugging container issues and understanding application behavior."
	}

	agent.explanationMethods["docker start"] = func(cmd string) string {
		parts := strings.Split(cmd, " ")
		if len(parts) > 2 {
			containerName := parts[2]
			return fmt.Sprintf("The `docker start %s` command starts the stopped container named '%s'. "+
				"This resumes the container in its previous state without creating a new container instance.\n\n"+
				"I recommended starting this container because it appears to be stopped but needed for your application stack "+
				"to function properly. Starting it will restore the service without losing any container-specific data.", containerName, containerName)
		}
		return "The `docker start` command resumes a stopped container while preserving its state, volumes, and configuration. " +
			"This is more efficient than creating a new container when you simply need to resume operations."
	}

	agent.explanationMethods["docker restart"] = func(cmd string) string {
		parts := strings.Split(cmd, " ")
		if len(parts) > 2 {
			containerName := parts[2]
			return fmt.Sprintf("The `docker restart %s` command stops and then starts the '%s' container in one operation. "+
				"This can resolve many common issues by refreshing the container's processes and connections.\n\n"+
				"I recommended restarting this container because the logs indicated connection issues that are often fixed "+
				"by a clean restart, which clears temporary state and re-establishes connections.", containerName, containerName)
		}
		return "The `docker restart` command stops and then starts a container in one operation. It's an efficient way to " +
			"refresh a container's state when it's encountering transient issues without having to manually stop and start it separately."
	}

	agent.explanationMethods["docker network"] = func(cmd string) string {
		if strings.Contains(cmd, "inspect") {
			parts := strings.Split(cmd, " ")
			if len(parts) > 3 {
				networkName := parts[3]
				return fmt.Sprintf("The `docker network inspect %s` command provides detailed information about the '%s' network. "+
					"It shows the network's configuration, connected containers, IP addresses, and gateway information.\n\n"+
					"I suggested inspecting this network because understanding the current network topology is essential for "+
					"diagnosing communication issues between containers.", networkName, networkName)
			}
			return "This command provides detailed information about a Docker network's configuration and which containers are attached to it."
		} else if strings.Contains(cmd, "ls") {
			return "The `docker network ls` command lists all networks on your Docker system. " +
				"I recommended this to get an overview of the available networks, which is essential for understanding " +
				"how your containers can communicate with each other. Container networking issues are a common source of problems in " +
				"multi-container applications."
		} else if strings.Contains(cmd, "connect") {
			parts := strings.Split(cmd, " ")
			if len(parts) >= 4 {
				networkName := parts[2]
				containerName := parts[3]
				return fmt.Sprintf("The `docker network connect %s %s` command connects the '%s' container to the '%s' network. "+
					"This allows the container to communicate with other containers on that network.\n\n"+
					"I suggested connecting this container to the network because the error logs indicated connection issues "+
					"that are likely due to network isolation.", networkName, containerName, containerName, networkName)
			}
			return "This command connects a container to a network, enabling it to communicate with other containers on that network."
		}
		return "Docker network commands manage container networking, allowing you to create, inspect, and modify networks " +
			"to control how containers communicate with each other and the outside world."
	}

	// Add fallback explanation method with more AI-like language
	agent.explanationMethods["default"] = func(cmd string) string {
		return fmt.Sprintf("The `%s` command is a Docker operation that interacts with your container environment. Based on your current context, I recommended it because it addresses the specific issue or task you're working on. Would you like me to provide a more detailed explanation of what this command does and why I suggested it?", cmd)
	}

	return agent
}

// ProcessUserMessage processes a user message and returns agent responses
func (a *AgentService) ProcessUserMessage(message string) []AgentMessage {
	var responses []AgentMessage

	// Track user input in context history
	if message != "help" && message != "e" && message != "y" && message != "n" && message != "q" {
		a.contextHistory = append(a.contextHistory, "User: "+message)
	}

	// If this is a command execution acceptance (y, yes)
	if a.conversationState == "awaiting_approval" && (strings.ToLower(message) == "y" || strings.ToLower(message) == "yes") {
		return a.ExecuteSuggestedCommand()
	}

	// If this is a command explanation request (e, explain)
	if a.conversationState == "awaiting_approval" && (strings.ToLower(message) == "e" || strings.ToLower(message) == "explain") {
		explanation := a.ExplainCommand(a.previousCommand)
		responses = append(responses, AgentMessage{
			Type:    TypeAgent,
			Content: explanation,
			Time:    time.Now(),
		})
		// Don't change the state, still awaiting approval
		return responses
	}

	// If this is the start of the conversation
	if a.conversationState == "initial" {
		// Choose a random welcome message
		welcomeMsg := a.welcomeMessages[rand.Intn(len(a.welcomeMessages))]
		responses = append(responses, AgentMessage{
			Type:    TypeAgent,
			Content: welcomeMsg,
			Time:    time.Now(),
		})

		// Choose a random initial response
		initialTemplates := a.responseTemplates["initial"]
		initialResponse := initialTemplates[rand.Intn(len(initialTemplates))]
		responses = append(responses, AgentMessage{
			Type:    TypeAgent,
			Content: initialResponse,
			Time:    time.Now().Add(1 * time.Second),
		})

		// Suggest the first command
		suggestedCmd := "docker ps"
		responses = append(responses, AgentMessage{
			Type:    TypeCommand,
			Content: suggestedCmd,
			Time:    time.Now().Add(2 * time.Second),
		})

		a.previousCommand = suggestedCmd
		a.conversationState = "awaiting_approval"
		a.contextHistory = append(a.contextHistory, "Agent suggested: "+suggestedCmd)
		return responses
	}

	// Process user messages based on intent detection
	// This better mimics how a real AI agent would work by detecting intents and topics
	lowerMessage := strings.ToLower(message)

	// Track the inferred intent for better context awareness
	var detectedIntent string

	if strings.Contains(lowerMessage, "list") && (strings.Contains(lowerMessage, "container") || strings.Contains(lowerMessage, "running")) {
		detectedIntent = "list_containers"
	} else if strings.Contains(lowerMessage, "logs") || strings.Contains(lowerMessage, "output") {
		detectedIntent = "check_logs"
	} else if strings.Contains(lowerMessage, "network") || strings.Contains(lowerMessage, "connect") {
		detectedIntent = "network_operations"
	} else if strings.Contains(lowerMessage, "image") || strings.Contains(lowerMessage, "pull") {
		detectedIntent = "image_operations"
	} else if (strings.Contains(lowerMessage, "start") || strings.Contains(lowerMessage, "run")) && strings.Contains(lowerMessage, "app") {
		detectedIntent = "start_container"
	} else if strings.Contains(lowerMessage, "restart") {
		detectedIntent = "restart_container"
	} else if strings.Contains(lowerMessage, "stop") || strings.Contains(lowerMessage, "kill") {
		detectedIntent = "stop_container"
	} else if strings.Contains(lowerMessage, "error") || strings.Contains(lowerMessage, "issue") || strings.Contains(lowerMessage, "problem") || strings.Contains(lowerMessage, "troubleshoot") {
		detectedIntent = "troubleshooting"
	} else {
		detectedIntent = "general_docker"
	}

	// Add more AI-like context-aware responses
	var agentResponse string
	var suggestedCmd string

	switch detectedIntent {
	case "list_containers":
		agentResponse = "I'll help you check your containers. Let me get a list of all containers including stopped ones for a complete picture:"
		suggestedCmd = "docker ps -a"
	case "check_logs":
		if strings.Contains(lowerMessage, "web") || strings.Contains(lowerMessage, "server") {
			agentResponse = "Let me check the logs for the web-server container to help diagnose any issues:"
			suggestedCmd = "docker logs web-server"
		} else if strings.Contains(lowerMessage, "app") {
			agentResponse = "I'll examine the logs for the app container to see why it might be failing:"
			suggestedCmd = "docker logs app"
		} else {
			agentResponse = "I'll check the logs for the redis-cache service since that's a common dependency that might be causing issues:"
			suggestedCmd = "docker logs redis-cache"
		}
	case "network_operations":
		if strings.Contains(lowerMessage, "inspect") || strings.Contains(lowerMessage, "detail") {
			agentResponse = "Let me inspect the application network to see which containers are connected and their IP configurations:"
			suggestedCmd = "docker network inspect my-application"
		} else {
			agentResponse = "I'll list all the networks in your environment so we can see what's available:"
			suggestedCmd = "docker network ls"
		}
	case "image_operations":
		agentResponse = "Here are the Docker images currently available on your system:"
		suggestedCmd = "docker images"
	case "start_container":
		agentResponse = "I'll start the app container for you. This will bring it online without creating a new container instance:"
		suggestedCmd = "docker start app"
	case "restart_container":
		if strings.Contains(lowerMessage, "redis") || strings.Contains(lowerMessage, "cache") {
			agentResponse = "I'll restart the redis-cache container to refresh its connections:"
			suggestedCmd = "docker restart redis-cache"
		} else {
			agentResponse = "I'll restart the web-server container to apply any configuration changes:"
			suggestedCmd = "docker restart web-server"
		}
	case "stop_container":
		agentResponse = "I'll stop the web-server container safely, allowing it to shutdown gracefully:"
		suggestedCmd = "docker stop web-server"
	case "troubleshooting":
		// For troubleshooting, simulate a more thoughtful analysis
		agentResponse = "Based on the information you've provided, there might be an issue with container networking or service dependencies. Let me first check which containers are running and their status:"
		suggestedCmd = "docker ps -a"
	default:
		// Default behavior with more AI-like reasoning
		agentResponse = "I understand you're working with Docker containers. To best assist you, let me first understand your current environment by checking your running containers:"
		suggestedCmd = "docker ps"
	}

	// Add agent response with AI-like behavior
	responses = append(responses, AgentMessage{
		Type:    TypeAgent,
		Content: agentResponse,
		Time:    time.Now(),
	})

	// Add command suggestion
	responses = append(responses, AgentMessage{
		Type:    TypeCommand,
		Content: suggestedCmd,
		Time:    time.Now().Add(1 * time.Second),
	})

	// Update state
	a.previousCommand = suggestedCmd
	a.conversationState = "awaiting_approval"
	a.contextHistory = append(a.contextHistory, "Agent analyzed: "+detectedIntent)
	a.contextHistory = append(a.contextHistory, "Agent suggested: "+suggestedCmd)

	return responses
}

// ExecuteSuggestedCommand simulates executing the previously suggested command
func (a *AgentService) ExecuteSuggestedCommand() []AgentMessage {
	var responses []AgentMessage

	// Execute the command
	output, err := a.dockerService.ExecuteCommand(a.previousCommand)

	// Command output
	if err == nil {
		responses = append(responses, AgentMessage{
			Type:    TypeCommandOutput,
			Content: output,
			Time:    time.Now(),
		})

		// Update state based on the command
		newState := "after_command"
		if strings.HasPrefix(a.previousCommand, "docker ps") {
			newState = "after_docker_ps"
		} else if strings.HasPrefix(a.previousCommand, "docker logs") {
			newState = "after_logs"
		} else if strings.HasPrefix(a.previousCommand, "docker network") {
			newState = "after_network"
		} else if strings.HasPrefix(a.previousCommand, "docker start") ||
			strings.HasPrefix(a.previousCommand, "docker restart") {
			newState = "after_start"
		}

		a.conversationState = newState

		// Check for templates for this state
		if templates, ok := a.responseTemplates[newState]; ok {
			var responseContent string
			template := templates[rand.Intn(len(templates))]

			// For some states, format the response with additional info
			if newState == "after_docker_ps" {
				// Count the number of running containers from output
				containerCount := len(strings.Split(output, "\n")) - 1
				responseContent = fmt.Sprintf(template, containerCount)
			} else {
				responseContent = template
			}

			responses = append(responses, AgentMessage{
				Type:    TypeAgent,
				Content: responseContent,
				Time:    time.Now().Add(1 * time.Second),
			})
		}

		// Suggest next command
		nextCommand := a.dockerService.SuggestNextCommand(a.previousCommand)
		responses = append(responses, AgentMessage{
			Type:    TypeCommand,
			Content: nextCommand,
			Time:    time.Now().Add(2 * time.Second),
		})

		a.previousCommand = nextCommand
		a.conversationState = "awaiting_approval"
	} else {
		// Error case
		responses = append(responses, AgentMessage{
			Type:    TypeError,
			Content: err.Error(),
			Time:    time.Now(),
		})

		responses = append(responses, AgentMessage{
			Type:    TypeAgent,
			Content: "There was an error executing that command. Would you like to try something else?",
			Time:    time.Now().Add(1 * time.Second),
		})

		// Suggest a fallback command
		fallbackCmd := "docker ps -a"
		responses = append(responses, AgentMessage{
			Type:    TypeCommand,
			Content: fallbackCmd,
			Time:    time.Now().Add(2 * time.Second),
		})

		a.previousCommand = fallbackCmd
		a.conversationState = "awaiting_approval"
	}

	return responses
}

// ExplainCommand provides an explanation for a Docker command
func (a *AgentService) ExplainCommand(command string) string {
	// Find the most specific explanation method
	for cmdPrefix, explainFunc := range a.explanationMethods {
		if strings.HasPrefix(command, cmdPrefix) {
			return explainFunc(command)
		}
	}

	// Use default explanation if no specific one is found
	return a.explanationMethods["default"](command)
}
