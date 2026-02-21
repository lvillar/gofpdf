package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func sendRequest(t *testing.T, s *Server, method string, id int, params interface{}) jsonrpcResponse {
	t.Helper()

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshaling request: %v", err)
	}
	reqBytes = append(reqBytes, '\n')

	var output bytes.Buffer
	s.input = bytes.NewReader(reqBytes)
	s.output = &output

	s.Run()

	var resp jsonrpcResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response %q: %v", output.String(), err)
	}
	return resp
}

func TestServerInitialize(t *testing.T) {
	s := NewServerWithIO(nil, nil)
	RegisterDefaultTools(s)

	resp := sendRequest(t, s, "initialize", 1, map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "test", "version": "1.0"},
	})

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Fatalf("unexpected protocol version: %v", result["protocolVersion"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("missing serverInfo")
	}
	if serverInfo["name"] != "gofpdf-mcp" {
		t.Fatalf("unexpected server name: %v", serverInfo["name"])
	}
}

func TestServerToolsList(t *testing.T) {
	s := NewServerWithIO(nil, nil)
	RegisterDefaultTools(s)

	resp := sendRequest(t, s, "tools/list", 2, nil)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatal("tools is not an array")
	}

	if len(tools) < 5 {
		t.Fatalf("expected at least 5 tools, got %d", len(tools))
	}

	// Check that key tools exist
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		tm, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := tm["name"].(string); ok {
			toolNames[name] = true
		}
	}

	expectedTools := []string{"create_pdf", "read_pdf", "read_pdf_text", "merge_pdfs", "add_watermark", "fill_form", "pdf_info"}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("expected tool %q not found", name)
		}
	}
}

func TestServerResourcesList(t *testing.T) {
	s := NewServerWithIO(nil, nil)
	RegisterDefaultResources(s)

	resp := sendRequest(t, s, "resources/list", 3, nil)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	resources, ok := result["resources"].([]interface{})
	if !ok {
		t.Fatal("resources is not an array")
	}

	if len(resources) != 4 {
		t.Fatalf("expected 4 resources, got %d", len(resources))
	}
}

func TestServerPing(t *testing.T) {
	s := NewServerWithIO(nil, nil)

	resp := sendRequest(t, s, "ping", 4, nil)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}
}

func TestServerUnknownMethod(t *testing.T) {
	s := NewServerWithIO(nil, nil)

	resp := sendRequest(t, s, "nonexistent/method", 5, nil)

	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Fatalf("expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestServerUnknownTool(t *testing.T) {
	s := NewServerWithIO(nil, nil)
	RegisterDefaultTools(s)

	resp := sendRequest(t, s, "tools/call", 6, map[string]interface{}{
		"name":      "nonexistent_tool",
		"arguments": map[string]interface{}{},
	})

	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestServerCreatePDFTool(t *testing.T) {
	s := NewServerWithIO(nil, nil)
	RegisterDefaultTools(s)

	resp := sendRequest(t, s, "tools/call", 7, map[string]interface{}{
		"name": "create_pdf",
		"arguments": map[string]interface{}{
			"template": map[string]interface{}{
				"title": "Test PDF",
				"pages": []interface{}{
					map[string]interface{}{
						"elements": []interface{}{
							map[string]interface{}{
								"type": "heading",
								"text": "Hello MCP",
								"level": 1,
							},
							map[string]interface{}{
								"type": "paragraph",
								"text": "Created via MCP tool.",
							},
						},
					},
				},
			},
		},
	})

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}

	// Verify result contains base64 PDF data
	resultBytes, _ := json.Marshal(resp.Result)
	resultStr := string(resultBytes)

	if !strings.Contains(resultStr, "PDF created successfully") {
		t.Fatalf("unexpected result: %s", resultStr)
	}
	if !strings.Contains(resultStr, "Base64") {
		t.Fatalf("expected base64 data in result: %s", resultStr)
	}
}

func TestServerMultipleRequests(t *testing.T) {
	// Test that the server can handle multiple requests in sequence
	requests := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":4,"method":"ping"}`,
	}

	input := strings.Join(requests, "\n") + "\n"
	var output bytes.Buffer

	s := NewServerWithIO(strings.NewReader(input), &output)
	RegisterDefaultTools(s)
	RegisterDefaultResources(s)

	s.Run()

	// Each line should be a valid JSON response
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 responses, got %d: %s", len(lines), output.String())
	}

	for i, line := range lines {
		var resp jsonrpcResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("response %d: unmarshal error: %v\nline: %s", i, err, line)
		}
		if resp.Error != nil {
			t.Errorf("response %d: unexpected error: %s", i, resp.Error.Message)
		}
	}
}

func TestToolAddTool(t *testing.T) {
	s := NewServerWithIO(nil, nil)

	customTool := Tool{
		Name:        "custom_tool",
		Description: "A custom test tool",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: func(args map[string]interface{}) (ToolResult, error) {
			return ToolResult{
				Content: []ContentBlock{{Type: "text", Text: "custom result"}},
			}, nil
		},
	}

	s.AddTool(customTool)

	resp := sendRequest(t, s, "tools/call", 1, map[string]interface{}{
		"name":      "custom_tool",
		"arguments": map[string]interface{}{},
	})

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(resultBytes), "custom result") {
		t.Fatalf("unexpected result: %s", string(resultBytes))
	}
}
