package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

const (
	serverName      = "omnivoice"
	serverVersion   = "1.0.0"
	protocolVersion = "2024-11-05"
)

// Server is an MCP server that communicates via stdio.
type Server struct {
	config  Config
	handler *ToolHandler
	logger  *slog.Logger

	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
}

// NewServer creates a new MCP server.
func NewServer(config Config) *Server {
	// Apply defaults
	if config.TranscriptTimeoutMs == 0 {
		config.TranscriptTimeoutMs = 180000
	}
	if config.STTLanguage == "" {
		config.STTLanguage = "en-US"
	}

	return &Server{
		config:  config,
		handler: NewToolHandler(config),
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		reader:  bufio.NewReader(os.Stdin),
		writer:  os.Stdout,
	}
}

// Run starts the MCP server and blocks until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("MCP server starting",
		"name", serverName,
		"version", serverVersion,
		"protocol", protocolVersion,
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.handleMessage(ctx); err != nil {
				if err == io.EOF {
					s.logger.Info("Client disconnected")
					return nil
				}
				s.logger.Error("Error handling message", "error", err)
			}
		}
	}
}

func (s *Server) handleMessage(ctx context.Context) error {
	line, err := s.reader.ReadBytes('\n')
	if err != nil {
		return err
	}

	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		s.sendError(nil, ParseError, "Parse error", nil)
		return nil
	}

	// Handle the request
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized":
		// Notification, no response needed
		s.logger.Info("Client initialized")
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(ctx, req)
	case "ping":
		s.handlePing(req)
	default:
		s.sendError(req.ID, MethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}

	return nil
}

func (s *Server) handleInitialize(req Request) {
	result := InitializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    serverName,
			Version: serverVersion,
		},
	}

	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsList(req Request) {
	result := ListToolsResult{
		Tools: s.handler.GetTools(),
	}

	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsCall(ctx context.Context, req Request) {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, InvalidParams, "Invalid parameters", nil)
		return
	}

	s.logger.Info("Tool call", "tool", params.Name)

	result := s.handler.CallTool(ctx, params.Name, params.Arguments)

	s.sendResult(req.ID, result)
}

func (s *Server) handlePing(req Request) {
	s.sendResult(req.ID, map[string]any{})
}

func (s *Server) sendResult(id json.RawMessage, result any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	s.send(resp)
}

func (s *Server) sendError(id json.RawMessage, code int, message string, data any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ErrorObject{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	s.send(resp)
}

func (s *Server) send(v any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		s.logger.Error("Failed to marshal response", "error", err)
		return
	}

	_, err = fmt.Fprintf(s.writer, "%s\n", data)
	if err != nil {
		s.logger.Error("Failed to write response", "error", err)
	}
}
