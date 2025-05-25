#!/usr/bin/env python3
"""
WebSocket server for the Human-in-the-Loop Command Agent
"""

import asyncio
import json
import logging
import websockets
from typing import Dict, Set
from uuid import uuid4
import signal
import sys

from .graph import create_agent_graph

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class AgentServer:
    def __init__(self, host="localhost", port=8765):
        self.host = host
        self.port = port
        self.clients: Dict[str, websockets.WebSocketServerProtocol] = {}
        self.agent_graph = create_agent_graph()
        self.active_sessions: Dict[str, dict] = {}
        
    async def register_client(self, websocket, client_id):
        """Register a new client connection"""
        self.clients[client_id] = websocket
        logger.info(f"Client {client_id} connected")
        
    async def unregister_client(self, client_id):
        """Unregister a client connection"""
        if client_id in self.clients:
            del self.clients[client_id]
            logger.info(f"Client {client_id} disconnected")
            
    async def send_message(self, client_id: str, message_type: str, content: str = "", **kwargs):
        """Send a message to a specific client"""
        if client_id not in self.clients:
            logger.warning(f"Client {client_id} not found")
            return
            
        message = {
            "type": message_type,
            "content": content,
            **kwargs
        }
        
        try:
            await self.clients[client_id].send(json.dumps(message))
        except websockets.exceptions.ConnectionClosed:
            await self.unregister_client(client_id)
        except Exception as e:
            logger.error(f"Error sending message to {client_id}: {e}")
            
    async def handle_client_message(self, client_id: str, message: dict):
        """Handle incoming message from client"""
        message_type = message.get("type")
        content = message.get("content", "")
        
        try:
            if message_type == "message":
                # Start a new agent session
                await self.start_agent_session(client_id, content)
                
            elif message_type == "command_approval":
                # Handle command approval response
                approved = message.get("approved", False)
                await self.handle_approval_response(client_id, approved)
                
            elif message_type == "retry_response":
                # Handle retry response
                retry = message.get("retry", False)
                await self.handle_retry_response(client_id, retry)
                
            else:
                await self.send_message(client_id, "error", f"Unknown message type: {message_type}")
                
        except Exception as e:
            logger.error(f"Error handling message from {client_id}: {e}")
            await self.send_message(client_id, "error", str(e))
            
    async def start_agent_session(self, client_id: str, user_prompt: str):
        """Start a new agent session for the user prompt"""
        try:
            # Create initial state
            initial_state = {
                "user_prompt": user_prompt,
                "messages": [],
                "generated_command": "",
                "command_output": "",
                "command_error": "",
                "execution_approved": False,
                "retry_count": 0
            }
            
            # Configuration for the graph execution
            config = {"configurable": {"thread_id": f"session_{client_id}_{uuid4()}"}}
            
            # Store session info
            self.active_sessions[client_id] = {
                "config": config,
                "waiting_for_response": False,
                "interrupt_type": None
            }
            
            # Start the agent graph
            await self.run_agent_graph(client_id, initial_state, config)
            
        except Exception as e:
            logger.error(f"Error starting agent session for {client_id}: {e}")
            await self.send_message(client_id, "error", str(e))
            
    async def run_agent_graph(self, client_id: str, state: dict, config: dict):
        """Run the agent graph and handle interrupts"""
        try:
            # Run the graph
            result = self.agent_graph.invoke(state, config=config)
            
            # Handle interrupts
            while result.get("__interrupt__"):
                await self.handle_interrupt(client_id, result, config)
                return  # Wait for user response
                
            # Graph completed successfully
            graph_state = self.agent_graph.get_state(config)
            final_state = graph_state.values
            
            if final_state.get("command_error"):
                await self.send_message(
                    client_id,
                    "command_output",
                    output=final_state.get("command_output", ""),
                    error=final_state.get("command_error", ""),
                    success=False
                )
            else:
                await self.send_message(
                    client_id,
                    "command_output", 
                    output=final_state.get("command_output", ""),
                    success=True
                )
            
            # Clean up session
            if client_id in self.active_sessions:
                del self.active_sessions[client_id]
                
        except Exception as e:
            logger.error(f"Error running agent graph for {client_id}: {e}")
            await self.send_message(client_id, "error", str(e))
            if client_id in self.active_sessions:
                del self.active_sessions[client_id]
                
    async def handle_interrupt(self, client_id: str, result: dict, config: dict):
        """Handle interrupts from the agent graph"""
        interrupts = result.get("__interrupt__", [])
        
        for interrupt_obj in interrupts:
            interrupt_value = interrupt_obj.value
            
            if "command" in interrupt_value:
                # This is a command approval request
                self.active_sessions[client_id]["waiting_for_response"] = True
                self.active_sessions[client_id]["interrupt_type"] = "approval"
                self.active_sessions[client_id]["config"] = config
                
                await self.send_message(
                    client_id,
                    "command_approval",
                    command=interrupt_value.get("command", ""),
                    explanation=interrupt_value.get("question", ""),
                    retry_count=interrupt_value.get("retry_count", 0)
                )
                
            elif "error" in interrupt_value:
                # This is a retry request
                self.active_sessions[client_id]["waiting_for_response"] = True
                self.active_sessions[client_id]["interrupt_type"] = "retry"
                self.active_sessions[client_id]["config"] = config
                
                await self.send_message(
                    client_id,
                    "retry_request",
                    command=interrupt_value.get("command", ""),
                    error=interrupt_value.get("error", ""),
                    output=interrupt_value.get("output", ""),
                    retry_count=interrupt_value.get("retry_count", 0)
                )
                
    async def handle_approval_response(self, client_id: str, approved: bool):
        """Handle approval response from client"""
        if client_id not in self.active_sessions:
            await self.send_message(client_id, "error", "No active session found")
            return
            
        session = self.active_sessions[client_id]
        if not session.get("waiting_for_response") or session.get("interrupt_type") != "approval":
            await self.send_message(client_id, "error", "Not waiting for approval")
            return
            
        config = session["config"]
        
        try:
            # Resume the graph with the approval response
            response = "approve" if approved else "reject"
            
            from langgraph.types import Command
            result = self.agent_graph.invoke(Command(resume=response), config=config)
            
            session["waiting_for_response"] = False
            session["interrupt_type"] = None
            
            # Continue processing the result
            await self.continue_after_response(client_id, result, config)
            
        except Exception as e:
            logger.error(f"Error handling approval response for {client_id}: {e}")
            await self.send_message(client_id, "error", str(e))
            
    async def handle_retry_response(self, client_id: str, retry: bool):
        """Handle retry response from client"""
        if client_id not in self.active_sessions:
            await self.send_message(client_id, "error", "No active session found")
            return
            
        session = self.active_sessions[client_id]
        if not session.get("waiting_for_response") or session.get("interrupt_type") != "retry":
            await self.send_message(client_id, "error", "Not waiting for retry response")
            return
            
        config = session["config"]
        
        try:
            # Resume the graph with the retry response
            response = "retry" if retry else "cancel"
            
            from langgraph.types import Command
            result = self.agent_graph.invoke(Command(resume=response), config=config)
            
            session["waiting_for_response"] = False
            session["interrupt_type"] = None
            
            # Continue processing the result
            await self.continue_after_response(client_id, result, config)
            
        except Exception as e:
            logger.error(f"Error handling retry response for {client_id}: {e}")
            await self.send_message(client_id, "error", str(e))
            
    async def continue_after_response(self, client_id: str, result: dict, config: dict):
        """Continue processing after receiving user response"""
        # Handle any additional interrupts
        while result.get("__interrupt__"):
            await self.handle_interrupt(client_id, result, config)
            return  # Wait for next user response
            
        # Graph completed
        graph_state = self.agent_graph.get_state(config)
        final_state = graph_state.values
        
        if final_state.get("command_error"):
            await self.send_message(
                client_id,
                "command_output",
                output=final_state.get("command_output", ""),
                error=final_state.get("command_error", ""),
                success=False
            )
        else:
            await self.send_message(
                client_id,
                "command_output", 
                output=final_state.get("command_output", ""),
                success=True
            )
        
        # Clean up session
        if client_id in self.active_sessions:
            del self.active_sessions[client_id]
            
    async def handle_websocket(self, websocket, path=None):
        """Handle WebSocket connection"""
        client_id = str(uuid4())
        await self.register_client(websocket, client_id)
        
        try:
            # Send welcome message
            await self.send_message(client_id, "message", "Connected to Cloud Assist Agent")
            
            async for message in websocket:
                try:
                    data = json.loads(message)
                    await self.handle_client_message(client_id, data)
                except json.JSONDecodeError:
                    await self.send_message(client_id, "error", "Invalid JSON message")
                except Exception as e:
                    logger.error(f"Error processing message from {client_id}: {e}")
                    await self.send_message(client_id, "error", str(e))
                    
        except websockets.exceptions.ConnectionClosed:
            logger.info(f"Client {client_id} disconnected")
        finally:
            await self.unregister_client(client_id)
            if client_id in self.active_sessions:
                del self.active_sessions[client_id]
                
    async def start_server(self):
        """Start the WebSocket server"""
        logger.info(f"Starting agent server on {self.host}:{self.port}")
        
        # Set up signal handlers for graceful shutdown
        def signal_handler(signum, frame):
            logger.info("Received shutdown signal")
            sys.exit(0)
            
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)
        
        async with websockets.serve(self.handle_websocket, self.host, self.port):
            logger.info(f"Agent server listening on ws://{self.host}:{self.port}")
            await asyncio.Future()  # Run forever
            
def main():
    """Main function to start the agent server"""
    server = AgentServer()
    asyncio.run(server.start_server())
    
if __name__ == "__main__":
    main()
