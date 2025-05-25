API Reference
WebSocket Messages
Client to Server
User Request:
{
  "type": "user_request",
  "data": {
    "prompt": "show all running docker containers"
  }
}

User Response:
{
  "type": "user_response", 
  "data": {
    "response": "approve"
  }
}

Server to Client
Human Input Required:
{
  "type": "human_input_required",
  "data": {
    "node": "human_approval",
    "interrupt_data": {
      "command": "docker ps",
      "user_prompt": "show running containers",
      "retry_count": 0
    },
    "session_id": "uuid-here"
  }
}

Agent Update:
{
  "type": "agent_update",
  "data": {
    "event": {
      "generate_command": {
        "generated_command": "docker ps",
        "messages": [...],
        "retry_count": 0
      }
    },
    "session_id": "uuid-here"
  }
}


