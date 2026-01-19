package mcp

import (
	"context"
	"fmt"
)

// Tool definitions

var toolDefinitions = []Tool{
	{
		Name:        "initiate_call",
		Description: "Start a phone call with the user. Use when you need voice input, want to report completed work, or need real-time discussion.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"message": {
					Type:        "string",
					Description: "What you want to say to the user. Be natural and conversational.",
				},
			},
			Required: []string{"message"},
		},
	},
	{
		Name:        "continue_call",
		Description: "Continue an active call with a follow-up message.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"call_id": {
					Type:        "string",
					Description: "The call ID from initiate_call",
				},
				"message": {
					Type:        "string",
					Description: "Your follow-up message",
				},
			},
			Required: []string{"call_id", "message"},
		},
	},
	{
		Name:        "speak_to_user",
		Description: "Speak a message on an active call without waiting for a response. Use this to acknowledge requests or provide status updates before starting time-consuming operations.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"call_id": {
					Type:        "string",
					Description: "The call ID from initiate_call",
				},
				"message": {
					Type:        "string",
					Description: "What to say to the user",
				},
			},
			Required: []string{"call_id", "message"},
		},
	},
	{
		Name:        "end_call",
		Description: "End an active call with a closing message.",
		InputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"call_id": {
					Type:        "string",
					Description: "The call ID from initiate_call",
				},
				"message": {
					Type:        "string",
					Description: "Your closing message (say goodbye!)",
				},
			},
			Required: []string{"call_id", "message"},
		},
	},
}

// ToolHandler handles tool calls.
type ToolHandler struct {
	sessions *SessionManager
	config   Config
}

// NewToolHandler creates a new tool handler.
func NewToolHandler(config Config) *ToolHandler {
	return &ToolHandler{
		sessions: NewSessionManager(config),
		config:   config,
	}
}

// GetTools returns the list of available tools.
func (h *ToolHandler) GetTools() []Tool {
	return toolDefinitions
}

// CallTool executes a tool and returns the result.
func (h *ToolHandler) CallTool(ctx context.Context, name string, args map[string]any) CallToolResult {
	switch name {
	case "initiate_call":
		return h.initiateCall(ctx, args)
	case "continue_call":
		return h.continueCall(ctx, args)
	case "speak_to_user":
		return h.speakToUser(ctx, args)
	case "end_call":
		return h.endCall(ctx, args)
	default:
		return CallToolResult{
			Content: []ContentBlock{TextContent(fmt.Sprintf("Unknown tool: %s", name))},
			IsError: true,
		}
	}
}

func (h *ToolHandler) initiateCall(ctx context.Context, args map[string]any) CallToolResult {
	message, ok := args["message"].(string)
	if !ok || message == "" {
		return CallToolResult{
			Content: []ContentBlock{TextContent("Missing required parameter: message")},
			IsError: true,
		}
	}

	// Create session
	session, err := h.sessions.CreateSession(ctx)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{TextContent(fmt.Sprintf("Failed to create session: %v", err))},
			IsError: true,
		}
	}

	// Speak and listen for response
	response, err := session.SpeakAndListen(ctx, message, h.config.TranscriptTimeoutMs)
	if err != nil {
		_ = session.Close()
		h.sessions.RemoveSession(session.ID)
		return CallToolResult{
			Content: []ContentBlock{TextContent(fmt.Sprintf("Call failed: %v", err))},
			IsError: true,
		}
	}

	// Record history
	session.AddHistory("claude", message)
	session.AddHistory("user", response)

	result := fmt.Sprintf("Call initiated successfully.\n\nCall ID: %s\n\nUser's response:\n%s\n\nUse continue_call to ask follow-ups or end_call to hang up.", session.ID, response)

	return CallToolResult{
		Content: []ContentBlock{TextContent(result)},
	}
}

func (h *ToolHandler) continueCall(ctx context.Context, args map[string]any) CallToolResult {
	callID, ok := args["call_id"].(string)
	if !ok || callID == "" {
		return CallToolResult{
			Content: []ContentBlock{TextContent("Missing required parameter: call_id")},
			IsError: true,
		}
	}

	message, ok := args["message"].(string)
	if !ok || message == "" {
		return CallToolResult{
			Content: []ContentBlock{TextContent("Missing required parameter: message")},
			IsError: true,
		}
	}

	session, err := h.sessions.GetSession(callID)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{TextContent(fmt.Sprintf("Session not found: %s", callID))},
			IsError: true,
		}
	}

	response, err := session.SpeakAndListen(ctx, message, h.config.TranscriptTimeoutMs)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{TextContent(fmt.Sprintf("Call failed: %v", err))},
			IsError: true,
		}
	}

	session.AddHistory("claude", message)
	session.AddHistory("user", response)

	return CallToolResult{
		Content: []ContentBlock{TextContent(fmt.Sprintf("User's response:\n%s", response))},
	}
}

func (h *ToolHandler) speakToUser(ctx context.Context, args map[string]any) CallToolResult {
	callID, ok := args["call_id"].(string)
	if !ok || callID == "" {
		return CallToolResult{
			Content: []ContentBlock{TextContent("Missing required parameter: call_id")},
			IsError: true,
		}
	}

	message, ok := args["message"].(string)
	if !ok || message == "" {
		return CallToolResult{
			Content: []ContentBlock{TextContent("Missing required parameter: message")},
			IsError: true,
		}
	}

	session, err := h.sessions.GetSession(callID)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{TextContent(fmt.Sprintf("Session not found: %s", callID))},
			IsError: true,
		}
	}

	if err := session.Speak(ctx, message); err != nil {
		return CallToolResult{
			Content: []ContentBlock{TextContent(fmt.Sprintf("Failed to speak: %v", err))},
			IsError: true,
		}
	}

	session.AddHistory("claude", message)

	return CallToolResult{
		Content: []ContentBlock{TextContent(fmt.Sprintf("Message spoken: \"%s\"", message))},
	}
}

func (h *ToolHandler) endCall(ctx context.Context, args map[string]any) CallToolResult {
	callID, ok := args["call_id"].(string)
	if !ok || callID == "" {
		return CallToolResult{
			Content: []ContentBlock{TextContent("Missing required parameter: call_id")},
			IsError: true,
		}
	}

	message, ok := args["message"].(string)
	if !ok || message == "" {
		return CallToolResult{
			Content: []ContentBlock{TextContent("Missing required parameter: message")},
			IsError: true,
		}
	}

	session, err := h.sessions.GetSession(callID)
	if err != nil {
		return CallToolResult{
			Content: []ContentBlock{TextContent(fmt.Sprintf("Session not found: %s", callID))},
			IsError: true,
		}
	}

	// Speak closing message
	if err := session.Speak(ctx, message); err != nil {
		return CallToolResult{
			Content: []ContentBlock{TextContent(fmt.Sprintf("Failed to speak: %v", err))},
			IsError: true,
		}
	}

	duration := session.Duration()

	// Close session
	_ = session.Close()
	h.sessions.RemoveSession(callID)

	return CallToolResult{
		Content: []ContentBlock{TextContent(fmt.Sprintf("Call ended. Duration: %ds", int(duration.Seconds())))},
	}
}
