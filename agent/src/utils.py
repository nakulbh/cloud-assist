import subprocess
from langgraph.types import Command


def clean_command_output(command: str) -> str:
    """Remove markdown code blocks from generated command if present"""
    if command.startswith("```"):
        lines = command.split('\n')
        command = lines[1] if len(lines) > 1 else command
    if command.endswith("```"):
        command = command[:-3].strip()
    return command.strip()


def execute_shell_command(command: str, timeout: int = 30):
    """Execute a shell command and return the result"""
    try:
        result = subprocess.run(
            command, 
            shell=True, 
            capture_output=True, 
            text=True, 
            timeout=timeout
        )
        
        if result.returncode == 0:
            return {
                "command_output": result.stdout,
                "command_error": "",
                "execution_approved": True
            }
        else:
            return {
                "command_output": result.stdout,
                "command_error": result.stderr,
                "execution_approved": True
            }
            
    except subprocess.TimeoutExpired:
        return {
            "command_output": "",
            "command_error": f"Command timed out after {timeout} seconds",
            "execution_approved": True
        }
    except Exception as e:
        return {
            "command_output": "",
            "command_error": f"Error executing command: {str(e)}",
            "execution_approved": True
        }


def print_result(result, state):
    """Print the execution result in a formatted way"""
    print("\n" + "="*50)
    print("EXECUTION RESULT")
    print("="*50)
    
    if state.get("command_output"):
        print(f"Command: {state['generated_command']}")
        print(f"Output:\n{state['command_output']}")
        
    if state.get("command_error"):
        print(f"Error: {state['command_error']}")
        
    print("="*50)


def handle_user_interrupt(result, config, app):
    """Handle user interrupts and get responses"""
    while result.get("__interrupt__"):
        print("\n" + "-"*50)
        interrupts = result["__interrupt__"]
        
        for interrupt_obj in interrupts:
            interrupt_value = interrupt_obj.value
            
            # Handle different types of interrupts
            if "question" in interrupt_value:
                print(f"\n{interrupt_value['question']}")
                
                if "command" in interrupt_value:
                    print(f"Generated command: {interrupt_value['command']}")
                    
                if "error" in interrupt_value:
                    print(f"Error: {interrupt_value['error']}")
                    
                if "options" in interrupt_value:
                    print("\nOptions:")
                    for key, desc in interrupt_value["options"].items():
                        print(f"  {key}: {desc}")
                
                # Get user response
                while True:
                    if "options" in interrupt_value:
                        valid_options = list(interrupt_value["options"].keys())
                        user_response = input(f"\nYour choice ({'/'.join(valid_options)}): ").strip().lower()
                        if user_response in valid_options:
                            break
                        else:
                            print(f"Please enter one of: {', '.join(valid_options)}")
                    else:
                        user_response = input("\nYour response: ").strip()
                        break
                
                # Resume the graph with the user's response
                result = app.invoke(Command(resume=user_response), config=config)
                break
    
    return result