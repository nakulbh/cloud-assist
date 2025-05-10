# Cloud-Assist User Flow

## Initial Flow

1. **Launch the Application**
   Users start the CLI by running the command:
   ```
   cloud-assist
   ```

2. **Authentication**
   - First-time users will see the login screen
   - Returning users with a saved API key will skip to the main interface

   Example:
   ```
   # First time launching
   $ cloud-assist
   
   Please enter your API key to authenticate:
   [Input field where user enters their API key]
   ```

try to add mock data for showing different use case of this cli

3. **Direct to Chat Interface**
   After successful authentication, users are taken directly to the chat interface, which is the primary interaction point of Cloud-Assist.

   Example:
   ```
   Welcome to Cloud-Assist! I'm ready to help with your DevOps tasks.
   
   What would you like to accomplish today?
   
   [STATUS BAR] [CHAT] cloud-assist ready | ctx:0 | esc:back • ctrl+c:quit • 15:30:45
   ```

## Main Interaction Flow

The primary interaction happens in the chat interface, which follows this pattern:

1. **User Describes Intent**
   Users type what they want to accomplish in natural language.

   Example:
   ```
   I need to set up monitoring for my Docker containers running in production
   ```

2. **AI Suggests Commands**
   Cloud-Assist processes the request and suggests appropriate commands.

   Example:
   ```
   I'll help you set up container monitoring. First, let's install Prometheus and Grafana:

   Suggested command:
   docker-compose up -d prometheus grafana
   
   [y] Execute  [n] Skip  [e] Explain  [q] Quit
   ```

3. **Command Approval**
   The user reviews the suggested command and chooses an action:
   - Yes (y): Execute the command
   - No (n): Skip and try a different approach
   - Explain (e): Get more details about what the command does
   - Quit (q): Exit the session

4. **Command Execution and Output Analysis**
   If approved, the command executes and Cloud-Assist analyzes the output.

   Example:
   ```
   $ docker-compose up -d prometheus grafana
   Creating network "monitoring_default" with the default driver
   Creating prometheus ... done
   Creating grafana ... done

   Great! Prometheus and Grafana are now running. Let's configure Prometheus to scrape your Docker containers.

   Suggested command:
   cat > prometheus.yml << EOF
   global:
     scrape_interval: 15s
   scrape_configs:
     - job_name: 'docker'
       docker_sd_configs:
         - host: unix:///var/run/docker.sock
   EOF

   [y] Execute  [n] Skip  [e] Explain  [q] Quit
   ```

5. **Continuous Context-Aware Interaction**
   Cloud-Assist maintains context throughout the session, building on previous commands.

## Real-World Example Workflows

### Example 1: Setting Up Kubernetes Monitoring

```
User: "I need to set up monitoring for my Kubernetes cluster"

Cloud-Assist: "I'll help you set up monitoring for your Kubernetes cluster. First, let's check if you have Helm installed:

Suggested command:
helm version

[y] Execute  [n] Skip  [e] Explain  [q] Quit"

User: y

[Command output shows Helm is installed]

Cloud-Assist: "Great! Now let's add the Prometheus community Helm chart repository:

Suggested command:
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

[y] Execute  [n] Skip  [e] Explain  [q] Quit"

User: e

Cloud-Assist: "This command adds the official Prometheus community Helm chart repository to your Helm configuration. This repository contains charts for Prometheus, Grafana, and related monitoring tools. The second command updates your local repository cache to ensure you have the latest charts.

[y] Execute  [n] Skip  [q] Quit"

User: y

[Command executes successfully]

Cloud-Assist: "Now let's install the kube-prometheus-stack which includes Prometheus, Grafana, and several exporters:

Suggested command:
helm install monitoring prometheus-community/kube-prometheus-stack --namespace monitoring --create-namespace

[y] Execute  [n] Skip  [e] Explain  [q] Quit"
```

### Example 2: Troubleshooting a Web Service

```
User: "My nginx web server is returning 503 errors. Help me troubleshoot."

Cloud-Assist: "I'll help you troubleshoot the 503 errors from your nginx server. First, let's check if nginx is running:

Suggested command:
systemctl status nginx

[y] Execute  [n] Skip  [e] Explain  [q] Quit"

User: y

[Command shows nginx is running but with errors]

Cloud-Assist: "Let's check the nginx error logs for more details:

Suggested command:
sudo tail -n 50 /var/log/nginx/error.log

[y] Execute  [n] Skip  [e] Explain  [q] Quit"

User: y

[Logs show connection issues to upstream server]

Cloud-Assist: "I see the issue. Nginx can't connect to your backend service. Let's check if your backend service is running:

Suggested command:
systemctl status my-backend-service

[y] Execute  [n] Skip  [e] Explain  [q] Quit"
```

## Key UI Components in the Flow

1. **Status Bar**
   - Shows the current mode (chat, confirmation)
   - Displays key bindings for quick reference
   - Shows the current time and context size

2. **Chat Interface**
   - Displays the conversation history
   - Shows command suggestions and execution results
   - Provides the input area for typing new requests

3. **Confirmation Dialog**
   - Appears when Cloud-Assist needs explicit approval for potentially impactful operations
   - Provides yes/no options with clear explanation of consequences

4. **Multiline Input**
   - Allows users to enter complex requests that span multiple lines
   - Useful for pasting configuration snippets or complex queries

## User Experience Features

1. **Persistent Context**: The CLI maintains understanding of your infrastructure throughout a session
2. **Security-Focused**: All suggested commands require explicit approval
3. **Intelligent Suggestions**: Commands are contextually relevant based on previous actions
4. **Helpful Explanations**: Users can request detailed explanations before executing commands

This user flow creates a guided, conversational DevOps experience where users describe their goals in natural language and Cloud-Assist handles the technical implementation details, making complex operations more accessible while maintaining user control over all actions.