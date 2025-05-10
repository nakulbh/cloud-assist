// Package mock provides mock implementations for testing UI components
package mock

import (
	"fmt"
	"strings"
	"time"
)

// MockCommand represents a command and its simulated response
type MockCommand struct {
	Command   string
	Response  string
	Delay     time.Duration
	IsSuccess bool
}

// DockerCommandService simulates Docker command execution for UI testing
type DockerCommandService struct {
	commands          map[string]MockCommand
	fallbackResponses map[string]string
}

// NewDockerCommandService creates a new mock Docker command service with pre-defined commands
func NewDockerCommandService() *DockerCommandService {
	service := &DockerCommandService{
		commands:          make(map[string]MockCommand),
		fallbackResponses: make(map[string]string),
	}

	// Add pre-defined Docker commands and responses
	service.AddCommand("docker ps", `CONTAINER ID   IMAGE          COMMAND                  CREATED          STATUS          PORTS                    NAMES
3a0a11eb1f29   nginx:latest   "/docker-entrypoint.…"   10 minutes ago   Up 10 minutes   0.0.0.0:80->80/tcp       web-server
b8d5f65c9eff   redis:alpine   "docker-entrypoint.s…"   3 hours ago      Up 3 hours      0.0.0.0:6379->6379/tcp   redis-cache`, 800*time.Millisecond, true)

	service.AddCommand("docker ps -a", `CONTAINER ID   IMAGE              COMMAND                  CREATED          STATUS                      PORTS                    NAMES
3a0a11eb1f29   nginx:latest       "/docker-entrypoint.…"   10 minutes ago   Up 10 minutes            0.0.0.0:80->80/tcp       web-server
b8d5f65c9eff   redis:alpine       "docker-entrypoint.s…"   3 hours ago      Up 3 hours               0.0.0.0:6379->6379/tcp   redis-cache
c9f87e012d3a   postgres:14        "docker-entrypoint.s…"   1 day ago        Exited (0) 2 hours ago                            db
d7e42a11d892   myapp:latest       "npm start"              5 hours ago      Exited (1) 30 minutes ago                         app`, 1200*time.Millisecond, true)

	service.AddCommand("docker images", `REPOSITORY   TAG       IMAGE ID       CREATED        SIZE
nginx        latest    a6bd71f48f68   2 days ago     187MB
redis        alpine    3e52887d9762   3 days ago     28.3MB
postgres     14        a7d0e695d068   1 week ago     412MB
myapp        latest    b9c12f4e0d31   5 hours ago    345MB
ubuntu       22.04     c6b84b685f35   2 weeks ago    77.8MB`, 700*time.Millisecond, true)

	service.AddCommand("docker logs web-server", `192.168.1.5 - - [10/May/2025:10:12:01 +0000] "GET / HTTP/1.1" 200 615 "-" "Mozilla/5.0"
192.168.1.10 - - [10/May/2025:10:12:05 +0000] "GET /style.css HTTP/1.1" 200 1270 "http://localhost/" "Mozilla/5.0"
192.168.1.5 - - [10/May/2025:10:13:21 +0000] "GET /api/v1/users HTTP/1.1" 404 153 "-" "curl/7.68.0"
192.168.1.8 - - [10/May/2025:10:15:42 +0000] "GET / HTTP/1.1" 200 615 "-" "Mozilla/5.0"`, 1500*time.Millisecond, true)

	service.AddCommand("docker logs app", `Starting application...
Connected to database
Error connecting to Redis cache: connection refused
Retrying connection in 5 seconds...
Error connecting to Redis cache: connection refused
Application exited with code 1`, 900*time.Millisecond, true)

	service.AddCommand("docker logs redis-cache", `1:C 10 May 2025 09:55:12.912 # oO0OoO0OoO0Oo Redis is starting oO0OoO0OoO0Oo
1:C 10 May 2025 09:55:12.912 # Redis version=7.0.5, bits=64
1:M 10 May 2025 09:55:12.913 * monotonic clock: POSIX clock_gettime
1:M 10 May 2025 09:55:12.913 # Server initialized
1:M 10 May 2025 09:55:12.913 * Ready to accept connections`, 600*time.Millisecond, true)

	service.AddCommand("docker network ls", `NETWORK ID     NAME                   DRIVER    SCOPE
9b75b5dc5f3e   bridge                 bridge    local
6a1d07951642   host                   host      local
e09b90661293   my-application         bridge    local
37bfda7e4221   none                   null      local`, 500*time.Millisecond, true)

	service.AddCommand("docker start app", `app`, 2000*time.Millisecond, true)

	service.AddCommand("docker restart redis-cache", `redis-cache`, 3000*time.Millisecond, true)

	service.AddCommand("docker stop web-server", `web-server`, 1500*time.Millisecond, true)

	service.AddCommand("docker network inspect my-application", `[
    {
        "Name": "my-application",
        "Id": "e09b90661293",
        "Created": "2025-05-09T15:38:08.673432Z",
        "Scope": "local",
        "Driver": "bridge",
        "EnableIPv6": false,
        "IPAM": {
            "Driver": "default",
            "Options": {},
            "Config": [
                {
                    "Subnet": "172.20.0.0/16",
                    "Gateway": "172.20.0.1"
                }
            ]
        },
        "Internal": false,
        "Attachable": false,
        "Ingress": false,
        "ConfigFrom": {
            "Network": ""
        },
        "ConfigOnly": false,
        "Containers": {
            "3a0a11eb1f29": {
                "Name": "web-server",
                "EndpointID": "12a3456789bc",
                "MacAddress": "02:42:ac:14:00:02",
                "IPv4Address": "172.20.0.2/16",
                "IPv6Address": ""
            },
            "b8d5f65c9eff": {
                "Name": "redis-cache",
                "EndpointID": "de9f87654321",
                "MacAddress": "02:42:ac:14:00:03",
                "IPv4Address": "172.20.0.3/16",
                "IPv6Address": ""
            }
        },
        "Options": {},
        "Labels": {}
    }
]`, 800*time.Millisecond, true)

	service.AddCommand("docker network connect my-application app", ``, 1000*time.Millisecond, true)

	// Add fallback responses for unknown commands
	service.AddFallbackResponse("docker run", "Container started successfully")
	service.AddFallbackResponse("docker pull", "Image pulled successfully")
	service.AddFallbackResponse("docker build", "Image built successfully")
	service.AddFallbackResponse("docker exec", "Command executed in container")
	service.AddFallbackResponse("docker", "Unknown Docker command. Please use a valid Docker command.")

	return service
}

// AddCommand adds a new mock command and response
func (s *DockerCommandService) AddCommand(command, response string, delay time.Duration, isSuccess bool) {
	s.commands[command] = MockCommand{
		Command:   command,
		Response:  response,
		Delay:     delay,
		IsSuccess: isSuccess,
	}
}

// AddFallbackResponse adds a fallback response for partial command matches
func (s *DockerCommandService) AddFallbackResponse(commandPrefix, response string) {
	s.fallbackResponses[commandPrefix] = response
}

// ExecuteCommand simulates executing a command and returns the result
func (s *DockerCommandService) ExecuteCommand(command string) (string, error) {
	// Check for exact match
	if mockCmd, ok := s.commands[command]; ok {
		// Simulate command execution delay
		time.Sleep(mockCmd.Delay)

		if mockCmd.IsSuccess {
			return mockCmd.Response, nil
		}
		return mockCmd.Response, fmt.Errorf("command failed: %s", command)
	}

	// Check for fallback response
	for prefix, response := range s.fallbackResponses {
		if strings.HasPrefix(command, prefix) {
			// Simulate command execution delay
			time.Sleep(1 * time.Second)
			return response, nil
		}
	}

	// Default response for unknown commands
	return "", fmt.Errorf("unknown command: %s", command)
}

// SuggestNextCommand suggests the next command based on previous command
func (s *DockerCommandService) SuggestNextCommand(previousCommand string) string {
	suggestions := map[string]string{
		"docker ps":                  "docker logs web-server",
		"docker ps -a":               "docker start app",
		"docker logs web-server":     "docker restart web-server",
		"docker logs app":            "docker restart redis-cache",
		"docker restart redis-cache": "docker logs app",
		"docker start app":           "docker network connect my-application app",
		"docker network ls":          "docker network inspect my-application",
		"docker network inspect":     "docker ps",
		"docker images":              "docker pull nginx:latest",
		"docker pull":                "docker run -d --name test-container nginx:latest",
		"docker run":                 "docker ps",
		"docker logs redis-cache":    "docker ps -a",
		"docker stop":                "docker ps -a",
	}

	// Find a matching suggestion
	for cmd, suggestion := range suggestions {
		if strings.HasPrefix(previousCommand, cmd) {
			return suggestion
		}
	}

	// Default suggestion
	return "docker ps"
}
