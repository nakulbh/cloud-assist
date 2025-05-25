from langchain_core.messages import SystemMessage


def get_command_generation_prompt():
    """System prompt for generating commands based on user requests"""
    return SystemMessage(content="""
You are a command-line assistant. Your job is to generate shell commands based on user requests.
Generate ONLY the command, no explanations or additional text.
The command should be safe and appropriate for the user's request.

Examples:
- User: "show all running docker containers" -> Response: "docker ps -a"
- User: "list all files in current directory" -> Response: "ls -la"
- User: "show system disk usage" -> Response: "df -h"

Generate a single, safe command for the following request:
""")


def get_retry_command_prompt(user_prompt: str, previous_command: str, error: str, retry_count: int):
    """System prompt for generating alternative commands after failures"""
    return SystemMessage(content=f"""
You are a command-line assistant. The previous command failed with an error.
Generate a different, alternative command to accomplish the same goal.

Original request: {user_prompt}
Previous command that failed: {previous_command}
Error encountered: {error}
Retry attempt: {retry_count}/3

Generate ONLY a single alternative command, no explanations or additional text.
Make sure the new command is different from the previous one and addresses the error.
""")


def get_approval_request_data(generated_command: str, user_prompt: str, retry_count: int):
    """Data structure for human approval requests"""
    return {
        "question": "Do you want to execute this command?",
        "command": generated_command,
        "user_prompt": user_prompt,
        "retry_count": retry_count,
        "options": {
            "approve": "Execute the command",
            "reject": "Generate a different command", 
            "cancel": "Cancel the operation"
        }
    }


def get_retry_request_data(original_command: str, error: str, output: str, retry_count: int):
    """Data structure for retry confirmation requests"""
    return {
        "question": "Command failed. Do you want me to try a different approach?",
        "original_command": original_command,
        "error": error,
        "output": output,
        "retry_count": retry_count,
        "options": {
            "retry": "Yes, try a different command",
            "stop": "No, stop here"
        }
    }