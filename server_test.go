package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/puttsk/go-mcp"
)

func simpleFunc(ctx context.Context, a, b int) int {
	return a + b
}

func scalarTypeFunc(a string, b float64, c bool) (string, error) {
	return fmt.Sprintf("String: %s\nNumber: %0.2f\nBoolean: %t", a, b, c), nil
}

func contextFunc(ctx context.Context, a int, b int) (string, error) {
	sessionId, _ := mcp.GetSessionFromContext(ctx)
	request, _ := mcp.GetRequestIDFromContext(ctx)

	return fmt.Sprintf("Session ID: %s\nRequest ID: %s\nSum: %d", sessionId.SessionID, request, a+b), nil
}

func errorFunc(ctx context.Context, a int, b int) (int, error) {
	return 0, fmt.Errorf("this is an error")
}

func imageFunc(ctx context.Context, img mcp.McpImage) (mcp.McpImage, error) {
	return img, nil
}

type TestTool struct {
	Name        string
	Description string
	Parameters  []mcp.McpToolParameter
	Function    any
	TestCases   []ToolTestCase
}

type ToolTestCase struct {
	Parameters []any
	Output     []mcp.McpToolOutput
}

var testTools = map[string]TestTool{
	"simple_func": {
		Name:        "simple_func",
		Description: "Simple function that adds two integers",
		Parameters: []mcp.McpToolParameter{
			{Name: "a", Description: "First integer"},
			{Name: "b", Description: "Second integer"},
		},
		Function: simpleFunc,
		TestCases: []ToolTestCase{
			{
				Parameters: []any{2, 3},
				Output: []mcp.McpToolOutput{
					{
						Type: mcp.McpToolOutputTypeText,
						Text: "5",
					},
				},
			},
		},
	},
	"scalar_type_func": {
		Name:        "scalar_type_func",
		Description: "Function that accepts scalar types: string, number, and boolean",
		Parameters: []mcp.McpToolParameter{
			{Name: "a", Type: mcp.McpToolDataTypeString},
			{Name: "b", Type: mcp.McpToolDataTypeNumber},
			{Name: "c", Type: mcp.McpToolDataTypeBoolean},
		},
		Function: scalarTypeFunc,
		TestCases: []ToolTestCase{
			{
				Parameters: []any{"hello", 3.0, true},
				Output: []mcp.McpToolOutput{
					{
						Type: mcp.McpToolOutputTypeText,
						Text: "String: hello\nNumber: 3.00\nBoolean: true",
					},
				},
			},
		},
	},
	"context_func": {
		Name:        "context_func",
		Description: "Function that uses context to access session ID and request ID",
		Parameters: []mcp.McpToolParameter{
			{Name: "a", Description: "First integer"},
			{Name: "b", Description: "Second integer"},
		},
		Function: contextFunc,
		TestCases: []ToolTestCase{
			{
				Parameters: []any{1, 2},
				Output: []mcp.McpToolOutput{
					{
						Type: mcp.McpToolOutputTypeText,
						Text: "Session ID: test-session\nRequest ID: test-request\nSum: 3",
					},
				},
			},
		},
	},
	"error_func": {
		Name:        "error_func",
		Description: "Function that returns an error",
		Parameters: []mcp.McpToolParameter{
			{Name: "a", Description: "First integer"},
			{Name: "b", Description: "Second integer"},
		},
		Function: errorFunc,
		TestCases: []ToolTestCase{
			{
				Parameters: []any{1, 2},
				Output: []mcp.McpToolOutput{
					{
						Type: mcp.McpToolOutputTypeError,
						Text: "this is an error",
					},
				},
			},
		},
	},
	"image_func": {
		Name:        "image_func",
		Description: "Function that accepts an image and returns it",
		Parameters: []mcp.McpToolParameter{
			{Name: "img", Description: "Image"},
		},
		Function: imageFunc,
		TestCases: []ToolTestCase{
			{
				Parameters: []any{mcp.McpImage{Data: "base64image", MimeType: "image/png"}},
				Output: []mcp.McpToolOutput{
					{
						Type:     mcp.McpToolOutputTypeImage,
						Data:     "base64image",
						MimeType: "image/png",
					},
				},
			},
		},
	},
}

func NewTestMcpServer() (*mcp.McpServer, error) {
	// Create a new MCP server
	server, err := mcp.NewMcpServer("test_server", "1.0.0", mcp.McpProtocol2025_30_26)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP server: %w", err)
	}
	server.LogLevel = mcp.LogLevelDebug

	for _, tool := range testTools {
		// Register the tool
		err = server.RegisterTool(tool.Name, tool.Function, tool.Parameters...)
		if err != nil {
			return nil, fmt.Errorf("failed to register tool: %w", err)
		}
		err = server.SetToolDescription(tool.Name, tool.Description)
		if err != nil {
			return nil, fmt.Errorf("failed to set tool description: %w", err)
		}
	}

	return server, nil
}

func NewTestContext() context.Context {
	ctx := context.TODO()
	// Set a test session in the context
	session := mcp.McpSession{
		SessionID:   "test-session",
		Initialized: true,
	}
	ctx = mcp.SetSessionInContext(ctx, session)
	ctx = mcp.SetRequestIDInContext(ctx, "test-request")
	return ctx
}

func TestMcpServerRegisterTool(t *testing.T) {
	// Test registering tools with different signatures

	// Create a new MCP server
	server, _ := mcp.NewMcpServer("test_server", "1.0.0", mcp.McpProtocol2025_30_26)
	server.LogLevel = mcp.LogLevelDebug

	// Register the tool
	err := server.RegisterTool("simple_func", simpleFunc, mcp.McpToolParameter{Name: "a", Description: "First integer"}, mcp.McpToolParameter{Name: "b", Description: "Second integer"})
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	err = server.RegisterTool("scalar_type_func", scalarTypeFunc, mcp.McpToolParameter{Name: "a"}, mcp.McpToolParameter{Name: "b"}, mcp.McpToolParameter{Name: "c"})
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	err = server.RegisterTool("context_func", contextFunc, mcp.McpToolParameter{Name: "a", Description: "First integer"}, mcp.McpToolParameter{Name: "b", Description: "Second integer"})
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	err = server.RegisterTool("error_func", errorFunc, mcp.McpToolParameter{Name: "a", Description: "First integer"}, mcp.McpToolParameter{Name: "b", Description: "Second integer"})
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	err = server.RegisterTool("image_func", imageFunc, mcp.McpToolParameter{Name: "img", Description: "Image"})
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Register a tool with duplicate name
	err = server.RegisterTool(
		"scalar_type_func",
		scalarTypeFunc,
		mcp.McpToolParameter{Name: "a"},
		mcp.McpToolParameter{Name: "b"},
		mcp.McpToolParameter{Name: "c"},
	)
	if err == nil {
		t.Fatalf("Tool can be registered with duplicate name")
	}

	// Register a tool with invalid number of parameters
	err = server.RegisterTool(
		"scalar_type_func",
		scalarTypeFunc,
		mcp.McpToolParameter{Name: "a"},
	)
	if err == nil {
		t.Fatalf("Tool can be registered with invalid number of parameters")
	}
}

func TestMcpServerListTools(t *testing.T) {
	server, err := NewTestMcpServer()
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}

	list, err := server.ListTools()
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	if len(list) != 5 {
		t.Fatalf("Expected 5 tools, got %d", len(list))
	}

	for _, toolDesc := range list {
		if toolDesc.Name == "" {
			t.Fatalf("Tool name is empty")
		}
		tool := testTools[toolDesc.Name]
		if tool.Name != toolDesc.Name {
			t.Fatalf("Tool name mismatch: expected %s, got %s", tool.Name, toolDesc.Name)
		}
		if tool.Description != toolDesc.Description {
			t.Fatalf("Tool description mismatch: expected %s, got %s", tool.Description, toolDesc.Description)
		}
		if len(toolDesc.InputSchema.Properties) != len(tool.Parameters) {
			t.Fatalf("Tool parameters count mismatch for %s: expected %d, got %d", tool.Name, len(tool.Parameters), len(toolDesc.InputSchema.Properties))
		}
	}
	j, _ := json.MarshalIndent(list, "", "  ")
	t.Logf("Response: %s", string(j))
}

func TestMcpServerCallTool(t *testing.T) {
	server, err := NewTestMcpServer()
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}
	ctx := NewTestContext()

	for _, tool := range testTools {
		t.Logf("Testing tool: %s", tool.Name)
		for _, testCase := range tool.TestCases {
			t.Logf("Running test case with parameters: %v", testCase.Parameters)
			if out, err := server.CallTool(ctx, tool.Name, testCase.Parameters...); err == nil {
				if len(out) != len(testCase.Output) {
					t.Fatalf("Expected %d output, got %d", len(testCase.Output), len(out))
				}
				for i, expected := range testCase.Output {
					if out[i].Type != expected.Type {
						t.Fatalf("Expected output type to be %s, got %s", expected.Type, out[0].Type)
					}
					switch expected.Type {
					case mcp.McpToolOutputTypeText:
						if out[i].Text != expected.Text {
							t.Fatalf("Expected output value to be %s, got %s", expected.Text, out[0].Text)
						}
					case mcp.McpToolOutputTypeError:
						if out[i].Text != expected.Text {
							t.Fatalf("Expected error message to be %s, got %s", expected.Text, out[0].Text)
						}
					case mcp.McpToolOutputTypeImage:
						if out[i].MimeType != expected.MimeType {
							t.Fatalf("Expected output mime type to be %s, got %s", expected.MimeType, out[0].MimeType)
						}
						if out[i].Data != expected.Data {
							t.Fatalf("Expected output data to be %s, got %s", expected.Data, out[0].Data)
						}
					}
				}
			} else {
				t.Fatalf("Failed to call tool: %v", err)
			}
		}
	}
}
