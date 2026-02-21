// Package mcp implements a Model Context Protocol (MCP) server that exposes
// gofpdf's PDF capabilities as tools and resources for AI assistants.
//
// The server communicates via JSON-RPC 2.0 over stdio and implements the
// MCP specification (2024-11-05) for tools and resources.
//
// # Usage with Claude Desktop
//
// Add to your claude_desktop_config.json:
//
//	{
//	  "mcpServers": {
//	    "gofpdf": {
//	      "command": "gofpdf-mcp"
//	    }
//	  }
//	}
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// Server is an MCP server that handles JSON-RPC 2.0 messages over stdio.
type Server struct {
	tools     map[string]Tool
	resources map[string]Resource
	input     io.Reader
	output    io.Writer
	mu        sync.Mutex
}

// Tool defines an MCP tool that can be called by the client.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Handler     ToolHandler            `json:"-"`
}

// ToolHandler is a function that executes a tool with the given arguments.
type ToolHandler func(args map[string]interface{}) (ToolResult, error)

// ToolResult is the result returned by a tool execution.
type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock is a piece of content in a tool result.
type ContentBlock struct {
	Type     string `json:"type"`               // "text" or "resource"
	Text     string `json:"text,omitempty"`
	MIMEType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"` // base64 for binary
}

// Resource defines an MCP resource.
type Resource struct {
	URI         string          `json:"uri"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	MIMEType    string          `json:"mimeType,omitempty"`
	Handler     ResourceHandler `json:"-"`
}

// ResourceHandler reads a resource and returns its content.
type ResourceHandler func(uri string) ([]ResourceContent, error)

// ResourceContent is the content of a read resource.
type ResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64
}

// JSON-RPC types
type jsonrpcRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *jsonrpcError    `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewServer creates a new MCP server reading from stdin and writing to stdout.
func NewServer() *Server {
	return &Server{
		tools:     make(map[string]Tool),
		resources: make(map[string]Resource),
		input:     os.Stdin,
		output:    os.Stdout,
	}
}

// NewServerWithIO creates a new MCP server with custom I/O for testing.
func NewServerWithIO(in io.Reader, out io.Writer) *Server {
	return &Server{
		tools:     make(map[string]Tool),
		resources: make(map[string]Resource),
		input:     in,
		output:    out,
	}
}

// AddTool registers a tool with the server.
func (s *Server) AddTool(t Tool) {
	s.tools[t.Name] = t
}

// AddResource registers a resource with the server.
func (s *Server) AddResource(r Resource) {
	s.resources[r.URI] = r
}

// Run starts the server and processes messages until EOF.
func (s *Server) Run() error {
	scanner := bufio.NewScanner(s.input)
	// MCP uses newline-delimited JSON
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonrpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		s.handleRequest(req)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(req jsonrpcRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized":
		// Notification, no response needed
	case "ping":
		s.sendResult(req.ID, map[string]interface{}{})
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	case "resources/list":
		s.handleResourcesList(req)
	case "resources/read":
		s.handleResourcesRead(req)
	default:
		s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

func (s *Server) handleInitialize(req jsonrpcRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":     map[string]interface{}{},
			"resources": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "gofpdf-mcp",
			"version": "1.0.0",
		},
	}
	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsList(req jsonrpcRequest) {
	tools := make([]map[string]interface{}, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		})
	}
	s.sendResult(req.ID, map[string]interface{}{"tools": tools})
}

func (s *Server) handleToolsCall(req jsonrpcRequest) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	tool, ok := s.tools[params.Name]
	if !ok {
		s.sendError(req.ID, -32602, "Unknown tool", params.Name)
		return
	}

	result, err := tool.Handler(params.Arguments)
	if err != nil {
		s.sendResult(req.ID, ToolResult{
			Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		})
		return
	}

	s.sendResult(req.ID, result)
}

func (s *Server) handleResourcesList(req jsonrpcRequest) {
	resources := make([]map[string]interface{}, 0, len(s.resources))
	for _, r := range s.resources {
		res := map[string]interface{}{
			"uri":  r.URI,
			"name": r.Name,
		}
		if r.Description != "" {
			res["description"] = r.Description
		}
		if r.MIMEType != "" {
			res["mimeType"] = r.MIMEType
		}
		resources = append(resources, res)
	}
	s.sendResult(req.ID, map[string]interface{}{"resources": resources})
}

func (s *Server) handleResourcesRead(req jsonrpcRequest) {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	resource, ok := s.resources[params.URI]
	if !ok {
		s.sendError(req.ID, -32602, "Unknown resource", params.URI)
		return
	}

	contents, err := resource.Handler(params.URI)
	if err != nil {
		s.sendError(req.ID, -32603, "Resource error", err.Error())
		return
	}

	s.sendResult(req.ID, map[string]interface{}{"contents": contents})
}

func (s *Server) sendResult(id *json.RawMessage, result interface{}) {
	s.send(jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *Server) sendError(id *json.RawMessage, code int, message string, data interface{}) {
	s.send(jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonrpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	})
}

func (s *Server) send(resp jsonrpcResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	data = append(data, '\n')
	s.output.Write(data)
}
